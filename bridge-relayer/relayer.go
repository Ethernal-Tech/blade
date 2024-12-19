package bridgerelayer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/btcsuite/btcd/btcec/v2"

	"github.com/0xPolygon/polygon-edge/types"
)

type BridgeRelayer struct {
	internalRPCAddr     string
	internalClient      txrelayer.TxRelayer
	externalRPCAddr     string
	externalClient      txrelayer.TxRelayer
	externalChainID     *big.Int
	bridgeStorageAddr   types.Address
	internalGatewayAddr types.Address
	externalGatewayAddr types.Address
	pollInterval        time.Duration
	privateKey          *crypto.ECDSAKey
}

type BridgeRelayerOption func(options *options) error

type options struct {
	externalRPCAddr     *string
	externalChainID     *uint64
	genesisPath         *string
	bridgeStorageAddr   *types.Address
	internalGatewayAddr *types.Address
	externalGatewayAddr *types.Address
	pollInterval        *time.Duration
	privateKey          *string
}

func WithExternalRPCAddr(address string) BridgeRelayerOption {
	return func(options *options) error {
		options.externalRPCAddr = &address

		return nil
	}
}

func WithExternalChainID(chainID uint64) BridgeRelayerOption {
	return func(options *options) error {
		options.externalChainID = &chainID

		return nil
	}
}

func WithGenesisPath(path string) BridgeRelayerOption {
	return func(options *options) error {
		options.genesisPath = &path

		return nil
	}
}

func WithBridgeStorageAddr(address types.Address) BridgeRelayerOption {
	return func(options *options) error {
		options.bridgeStorageAddr = &address

		return nil
	}
}

func WithInternalGatewayCAddr(address types.Address) BridgeRelayerOption {
	return func(options *options) error {
		options.internalGatewayAddr = &address

		return nil
	}
}

func WithExternalGatewayAddr(address types.Address) BridgeRelayerOption {
	return func(options *options) error {
		options.externalGatewayAddr = &address

		return nil
	}
}

func WithPollInterval(interval time.Duration) BridgeRelayerOption {
	return func(options *options) error {
		options.pollInterval = &interval

		return nil
	}
}

func WithPrivateKey(key string) BridgeRelayerOption {
	return func(options *options) error {
		options.privateKey = &key

		return nil
	}
}

func NewBridgeRelayer(internalRPCAddr string, opts ...BridgeRelayerOption) (*BridgeRelayer, error) {
	errFunc := func(err error) error {
		return fmt.Errorf("cannot create a new bridge relayer: %w", err)
	}

	relayer := &BridgeRelayer{}

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(internalRPCAddr))
	if err != nil {
		return nil, errFunc(err)
	}

	relayer.internalRPCAddr = internalRPCAddr
	relayer.internalClient = txRelayer

	sopts := &options{}

	for _, option := range opts {
		err := option(sopts)

		if err != nil {
			return nil, errFunc(err)
		}
	}

	chainConfig, err := chain.ImportFromFile(*sopts.genesisPath)
	if err != nil {
		return nil, errFunc(err)
	}

	consensusConfig, err := config.GetPolyBFTConfig(chainConfig.Params)
	if err != nil {
		return nil, errFunc(err)
	}

	bridgeConfig := consensusConfig.Bridge[*sopts.externalChainID]

	txRelayer, err = txrelayer.NewTxRelayer(txrelayer.WithIPAddress(bridgeConfig.JSONRPCEndpoint))
	if err != nil {
		return nil, errFunc(err)
	}

	relayer.externalRPCAddr = bridgeConfig.JSONRPCEndpoint
	relayer.externalClient = txRelayer

	relayer.externalChainID = big.NewInt(int64(*sopts.externalChainID))

	// address of the bridge storage should also be read from the genesis
	relayer.bridgeStorageAddr = contracts.BridgeStorageContract
	relayer.internalGatewayAddr = bridgeConfig.InternalGatewayAddr
	relayer.externalGatewayAddr = bridgeConfig.ExternalGatewayAddr

	relayer.pollInterval = time.Second * 5

	privBytes, err := hex.DecodeString(*sopts.privateKey)
	if err != nil {
		return nil, errFunc(err)
	}

	x, y := btcec.S256().ScalarBaseMult(privBytes)

	privateKey := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(privBytes),
		PublicKey: ecdsa.PublicKey{
			Curve: btcec.S256(),
			X:     x,
			Y:     y,
		},
	}

	relayer.privateKey = crypto.NewECDSAKey(privateKey)

	return relayer, nil
}

func (r *BridgeRelayer) Start() {
	var lastBridged = big.NewInt(-1)

	t := time.NewTicker(r.pollInterval)

	for {
		select {
		case <-t.C:
			batches, err := GetBridgeBatchesFromNumber(big.NewInt(0).Add(lastBridged, big.NewInt(1)), r.internalClient)
			if err != nil {
				fmt.Println("err:", err)

				continue
			} else if len(batches) == 0 {
				fmt.Println("no new batches found")

				continue
			}

			for i, batch := range batches {
				fmt.Println("new batch found", batch.StartID.String(), "-", batch.EndID.String())

				var (
					sourceRelayer      txrelayer.TxRelayer
					sourceGateway      types.Address
					destinationRelayer txrelayer.TxRelayer
					destinationGateway types.Address
				)

				if batch.SourceChainID.Cmp(r.externalChainID) == 0 {
					sourceGateway = r.externalGatewayAddr
					sourceRelayer = r.externalClient

					destinationGateway = r.internalGatewayAddr
					destinationRelayer = r.internalClient
				} else {
					sourceGateway = r.internalGatewayAddr
					sourceRelayer = r.internalClient

					destinationGateway = r.externalGatewayAddr
					destinationRelayer = r.externalClient
				}

				messages, err := GetBridgeMessagesInRange(batches[i].StartID, batches[i].EndID, sourceRelayer, sourceGateway)
				if err != nil {
					fmt.Println("err:", err)

					continue
				}

				input, err := (&contractsapi.ReceiveBatchGatewayFn{
					BatchMessages:     messages,
					SignedBridgeBatch: &batches[i],
				}).EncodeAbi()
				if err != nil {
					fmt.Println("err:", err)

					continue
				}

				tx := types.NewTx(types.NewLegacyTx(
					types.WithFrom(r.privateKey.Address()),
					types.WithTo(&destinationGateway),
					types.WithInput(input),
				))

				_, err = destinationRelayer.SendTransaction(tx, r.privateKey)
				if err != nil {
					fmt.Println("err:", err)

					continue
				}

				lastBridged.Add(lastBridged, big.NewInt(1))
				fmt.Println("batch with", batch.StartID.String(), "-", batch.EndID.String(), "successfully transported")
			}
		}
	}
}
