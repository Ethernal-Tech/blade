package bridgerelayer

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"

	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/types"
)

type BridgeRelayer struct {
	internalRPCAddr     string
	internalClient      *jsonrpc.EthClient
	externalRPCAddr     string
	externalClient      *jsonrpc.EthClient
	externalChainID     *big.Int
	bridgeStorageAddr   types.Address
	internalGatewayAddr types.Address
	externalGatewayAddr types.Address
	pollInterval        time.Duration
	privateKey          *ecdsa.PrivateKey
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
	privateKey          *ecdsa.PrivateKey
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

func WithPrivateKey(key ecdsa.PrivateKey) BridgeRelayerOption {
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

	client, err := jsonrpc.NewEthClient(internalRPCAddr)
	if err != nil {
		return nil, errFunc(err)
	}

	relayer.internalRPCAddr = internalRPCAddr
	relayer.internalClient = client

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

	client, err = jsonrpc.NewEthClient(bridgeConfig.JSONRPCEndpoint)
	if err != nil {
		return nil, errFunc(err)
	}

	relayer.externalRPCAddr = *sopts.externalRPCAddr
	relayer.externalClient = client

	relayer.externalChainID = big.NewInt(int64(*sopts.externalChainID))

	// address of the bridge storage should also be read from the genesis
	relayer.bridgeStorageAddr = contracts.BridgeStorageContract
	relayer.internalGatewayAddr = bridgeConfig.InternalGatewayAddr
	relayer.externalGatewayAddr = bridgeConfig.ExternalGatewayAddr

	relayer.pollInterval = time.Second * 10

	privateKey, err := crypto.GenerateECDSAPrivateKey()
	if err != nil {
		return nil, errFunc(err)
	}

	relayer.privateKey = privateKey

	return relayer, nil
}

func (r BridgeRelayer) Start() {
	// bridge relayer logic ...
}
