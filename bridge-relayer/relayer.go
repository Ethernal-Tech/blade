package bridgerelayer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	bolt "go.etcd.io/bbolt"

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
	db                  *bolt.DB
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
	dbPath              *string
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

func WithDbPath(path string) BridgeRelayerOption {
	return func(options *options) error {
		options.dbPath = &path

		return nil
	}
}

func NewBridgeRelayer(internalRPCAddr string, privateKey string, opts ...BridgeRelayerOption) (*BridgeRelayer, error) {
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

	pkBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, errFunc(err)
	}

	x, y := btcec.S256().ScalarBaseMult(pkBytes)

	pk := &ecdsa.PrivateKey{
		D: new(big.Int).SetBytes(pkBytes),
		PublicKey: ecdsa.PublicKey{
			Curve: btcec.S256(),
			X:     x,
			Y:     y,
		},
	}

	relayer.privateKey = crypto.NewECDSAKey(pk)

	if sopts.dbPath == nil {
		relayer.db, err = bolt.Open("bridge-relayer.db", 0600, nil)
	} else {
		relayer.db, err = bolt.Open(*sopts.dbPath, 0600, nil)
	}

	if err != nil {
		return nil, errFunc(err)
	}

	err = relayer.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("lastBridgedBucket"))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, errFunc(err)
	}

	return relayer, nil
}

func (r *BridgeRelayer) Start() {
	var lastBridged = big.NewInt(-1)

	key := []byte{'l', 'a', 's', 't'}

	err := r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("lastBridgedBucket"))

		if bucket == nil {
			return errors.New("cannot find a bucket with the `lastBridgedBucket` name")
		}

		value := bucket.Get(key)

		if value != nil {
			var temp string
			if err := json.Unmarshal(value, &temp); err != nil {
				return err
			}

			if _, ok := lastBridged.SetString(temp, 10); !ok {
				return errors.New("cannot set lastBridge correctly")
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println(err)

		return
	}

	t := time.NewTicker(r.pollInterval)

	fmt.Println("[START] Starting the bridge relayer with the start batch id", lastBridged.String())

	for {
		select {
		case <-t.C:
			fmt.Println("[INFO] Trying to get a batches with the id higher than", lastBridged.String())
			batches, err := GetBridgeBatchesFromNumber(big.NewInt(0).Add(lastBridged, big.NewInt(1)), r.internalClient)
			if err != nil {
				fmt.Println("err:", err)

				continue
			} else if len(batches) == 0 {
				fmt.Println("[FAIL] Cannot find a new batches")

				continue
			} else {
				fmt.Println("[INFO] Found", len(batches), "new batches")
			}

			for i, batch := range batches {
				fmt.Println("[INFO]", big.NewInt(0).Add(lastBridged, big.NewInt(int64(1))), "id-ed batch was found with the messages:", batch.StartID.String(), "-", batch.EndID.String())

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
					fmt.Println("[FAIL]", big.NewInt(0).Add(lastBridged, big.NewInt(int64(1))), "id-ed batch has already been processed or cannot be processed")
				} else {
					fmt.Println("[INFO]", big.NewInt(0).Add(lastBridged, big.NewInt(int64(1))), "id-ed batch has been successfully processed/transfered")
				}

				lastBridged.Add(lastBridged, big.NewInt(1))

				err = r.db.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket([]byte("lastBridgedBucket"))

					if bucket == nil {
						return errors.New("cannot find a bucket with the `lastBridgedBucket` name")
					}

					if value, err := json.Marshal(lastBridged.String()); err != nil {
						return err
					} else {
						if err = bucket.Put(key, value); err != nil {
							return err
						}
					}

					return nil
				})

				if err != nil {
					fmt.Println(err)

					return
				}

				fmt.Println("[INFO]", lastBridged.String(), "batch id has been successfully stored into bolt DB")
			}
		}
	}
}
