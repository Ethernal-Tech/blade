package bridgerelayer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/hashicorp/go-hclog"

	"github.com/0xPolygon/polygon-edge/types"
)

// BridgeRelayer represents a relayer instance responsible for managing
// the token transfer process between an internal (Blade) and an EVM-based
// external blockchain network. It operates bidirectionaly and exclusively
// for one internal-external chain relation.
type BridgeRelayer struct {
	// internalChainRPC denotes the RPC endpoint of the internal blockchain
	// network (Blade).
	internalRPCAddr string

	// internalClient is a transaction relayer instance used for interacting
	// with the internal blockchain network (Blade). It manages transaction
	// submissions and queries, facilitating communication with the internal
	// network.
	internalClient txrelayer.TxRelayer

	// externalRPCAddr denotes the RPC endpoint of the external EVM-based
	// blockchain network.
	externalRPCAddr string

	// externalClient is a transaction relayer instance used for interacting
	// with the external EVM-based blockchain network. It manages transaction
	// submissions and queries, facilitating communication with the external
	// network.
	externalClient txrelayer.TxRelayer

	// externalChainID represents the unique ID of the external network.
	externalChainID *big.Int

	// bridgeStorageAddr contains the address of the Bridge Storage contract.
	bridgeStorageAddr types.Address

	// internalGatewayAddr contains the address of the Gateway contract on
	// the internal blockchain network.
	internalGatewayAddr types.Address

	// externalGatewayAddr contains the address of the Gateway contract on
	// the external blockchain network.
	externalGatewayAddr types.Address

	// pollInterval specifies the frequency at which the relayer polls for
	// new token-transfer events and processes them if any are found.
	pollInterval time.Duration

	// privateKey denotes the relayer's private key used to sign transactions.
	// The address derived from it represents the one to which the relayer
	// will receive the reward for successfully completed transfers.
	privateKey *crypto.ECDSAKey

	// db is a BoltDB instance used for (persistent) local storage.
	db *bolt.DB

	// he logger is an instance of the hclog logging library.
	// used to handle application logging.
	logger hclog.Logger
}

type BridgeRelayerOption func(*options) error

// options encapsulates all the configuration settings that can be used when
// creating a new bridge relayer. All fields are pointers, thus it is easy to
// make a difference between client-provided and default values. If a field
// is non-nil, it indicates that the client provided a value; otherwise, the
// default value should be used.
//
// An alternative would be to use non-pointer fields with prepopulated default
// values, but this approach is less suitable in our case due to the complexity
// of some defaults. For example, determining the RPC address of an external
// chain would require fetching (from the internal chain or local storage) and
// parsing genesis data, which can be resource-intensive. Out approach ensures
// that unnecessary computations are avoided when client-provided values are
// available.
//
// Additionally, a hybrid approach could mix pointer and non-pointer fields.
// However, for consistency, we keep all fields as pointers in our design.
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
	logLevel            hclog.Level
	jsonLogFormat       bool
	logDir              string
}

// WithExternalRPCAddr configures the relayer to use the specified RPC address
// for communication with an external blockchain network. The address must be
// a valid URL with scheme and host.
func WithExternalRPCAddr(address string) BridgeRelayerOption {
	return func(o *options) error {
		if address == "" {
			return fmt.Errorf("external RPC address cannot be empty")
		}

		if _, err := url.Parse(address); err != nil {
			return fmt.Errorf("invalid external RPC address format: %w", err)
		}

		o.externalRPCAddr = &address

		return nil
	}
}

// WithExternalChainID configures the relayer to use the specified chain ID
// for an external blockchain network. The chain ID must be a positive integer
// and is used to identify the network (e.g., 1 for Ethereum mainnet).
func WithExternalChainID(chainID uint64) BridgeRelayerOption {
	return func(o *options) error {
		if chainID <= 0 {
			return fmt.Errorf("external chain ID must be a positive number")
		}

		o.externalChainID = &chainID

		return nil
	}
}

// WithGenesisPath configures the relayer to use the specified path to read the
// genesis of the internal chain from local storage and obtain all the necessary
// information. Path must be specified as a relative to the executable's location.
// The genesis file must exist and be readable at the specified path.
func WithGenesisPath(path string) BridgeRelayerOption {
	return func(o *options) error {
		if path == "" {
			return fmt.Errorf("genesis file path cannot be empty")
		}

		fileInfo, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("genesis file not found at path: %s", path)
			}

			return fmt.Errorf("error accessing genesis file: %w", err)
		}

		if fileInfo.IsDir() {
			return fmt.Errorf("specified path is a directory, expected a file: %s", path)
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("genesis file is not readable: %w", err)
		}

		file.Close()

		o.genesisPath = &path

		return nil
	}
}

// WithBridgeStorageAddr configures the relayer to use the specified address
// as an address of the Bridge Storage smart contract.
func WithBridgeStorageAddr(address types.Address) BridgeRelayerOption {
	return func(o *options) error {
		o.bridgeStorageAddr = &address

		return nil
	}
}

// WithInternalGatewayCAddr configures the relayer to use the specified address
// as an address of the Gateway smart contract on the internal (Blade) chain.
func WithInternalGatewayCAddr(address types.Address) BridgeRelayerOption {
	return func(o *options) error {
		o.internalGatewayAddr = &address

		return nil
	}
}

// WithExternalGatewayAddr configures the relayer to use the specified address
// as an address of the Gateway smart contract on the external chain.
func WithExternalGatewayAddr(address types.Address) BridgeRelayerOption {
	return func(o *options) error {
		o.externalGatewayAddr = &address

		return nil
	}
}

// WithPollInterval configures the relayer to use the specified time interval
// as the frequency at which the relayer polls for new token-transfer events
// and processes them if any are found. The interval must be between 1 second
// and 10 minutes to prevent both excessive polling and unreasonably delays.
func WithPollInterval(interval time.Duration) BridgeRelayerOption {
	return func(o *options) error {
		const (
			minInterval = 1 * time.Second
			maxInterval = 10 * time.Minute
		)

		if interval < minInterval {
			return fmt.Errorf("poll interval too short, it must be at least 1 second")
		}

		if interval > maxInterval {
			return fmt.Errorf("poll interval too long, it can not exceed 10 minutes")
		}

		o.pollInterval = &interval

		return nil
	}
}

// WithDBPath configures the relayer to use the Bolt DB at the specified path
// as persistent storage for token cross-chain transfer information. In case
// Bolt DB already exists, it will be used; otherwise, a new database at the
// specified path will be created. Path must be specified as a relative to the
// executable's location.
func WithDBPath(path string) BridgeRelayerOption {
	return func(o *options) error {
		if path == "" {
			return fmt.Errorf("database path cannot be empty")
		}

		o.dbPath = &path

		return nil
	}
}

func WithLogLevel(level hclog.Level) BridgeRelayerOption {
	return func(options *options) error {
		options.logLevel = level

		return nil
	}
}

func WithLogJSONFormat(jsonFormat bool) BridgeRelayerOption {
	return func(options *options) error {
		options.jsonLogFormat = jsonFormat

		return nil
	}
}

func WithLogDir(logDir string) BridgeRelayerOption {
	return func(options *options) error {
		options.logDir = logDir

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

	// default options
	sopts := &options{
		logDir:        "",
		logLevel:      hclog.Info,
		jsonLogFormat: false,
	}

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

	logger, err := newLoggerFromConfig(sopts)
	if err != nil {
		return nil, errFunc(err)
	}

	relayer.logger = logger
	relayer.privateKey = crypto.NewECDSAKey(pk)

	logger.Info("opening bolt db...", "path", *sopts.dbPath)

	relayer.db, err = openDB(sopts.dbPath)
	if err != nil {
		return nil, errFunc(err)
	}

	return relayer, nil
}

func (r *BridgeRelayer) Start() {
	lastBridged := big.NewInt(-1)

	bucketName := []byte("lastBridgedBucket")
	key := []byte("lastBridgedKey")

	err := r.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return errors.New("cannot find a lastBridgedBucket bucket")
		}

		value := bucket.Get(key)

		if value != nil {
			var temp string
			if err := json.Unmarshal(value, &temp); err != nil {
				return err
			}

			if _, ok := lastBridged.SetString(temp, 10); !ok {
				return errors.New("cannot set lastBridged correctly")
			}
		}

		return nil
	})

	if err != nil {
		r.logger.Error("failed to read lastBridgedBucket", "err", err)

		return
	}

	t := time.NewTicker(r.pollInterval)

	r.logger.Info("starting the bridge relayer...", "start batch id", lastBridged.String())

	for {
		select {
		case <-t.C:
			r.logger.Info(fmt.Sprintf("getting new batches with id > %s", lastBridged.String()))

			batches, err := GetBridgeBatchesFromNumber(big.NewInt(0).Add(lastBridged, big.NewInt(1)), r.internalClient)
			if err != nil {
				r.logger.Error("failed to get batches from BridgeStorage contract", "err", err)

				continue
			} else {
				r.logger.Info("received new bridge batches", "total", len(batches))
			}

			for _, batch := range batches {
				if batch.ValidatorSetBatchID.Cmp(big.NewInt(0)) > 0 {
					r.logger.Info(fmt.Sprintf("getting new validator set batch with id > %s", lastBridged.String()))

					newValidatorSet, err := GetBridgeValidatorSet(batch.ValidatorSetBatchID, r.internalClient)
					if err != nil {
						r.logger.Error("failed to get validator set from BridgeStorage contract", "err", err)

						continue
					}

					if err := r.sendCommitValidatorSet(newValidatorSet); err != nil {
						r.logger.Error("failed to send validator set batch on gateway", "err", err)

						continue
					}
				} else {
					r.logger.Info("found batch", "events start-id", batch.StartID.String(),
						"events end-id", batch.EndID.String(), "is rollback batch", batch.IsRollback)

					if err := r.sendSignedBridgeMessageBatch(&batch); err != nil {
						r.logger.Error("failed to send bridge batch on gateway", "err", err)

						continue
					}
				}

				lastBridged.Add(lastBridged, big.NewInt(1))

				r.logger.Info("batch has been successfully processed/sent", "batch id", lastBridged)

				err = r.db.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(bucketName)

					if bucket == nil {
						return errors.New("cannot find a lastBridgedBucket bucket")
					}

					value, err := json.Marshal(lastBridged.String())
					if err != nil {
						return err
					}

					if err = bucket.Put(key, value); err != nil {
						return err
					}

					return nil
				})

				if err != nil {
					r.logger.Error("failed to update lastBridgedBucket", "err", err)

					return
				}

				r.logger.Info("batch has been successfully saved into bolt DB", "batch id", lastBridged.String())
			}
		}
	}
}

func openDB(dbPath *string) (*bolt.DB, error) {
	var (
		retVal *bolt.DB
		err    error
	)

	if dbPath == nil {
		retVal, err = bolt.Open("bridge-relayer.db", 0666, nil)
	} else {
		retVal, err = bolt.Open(*dbPath, 0666, nil)
	}

	if err != nil {
		return nil, err
	}

	err = retVal.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("lastBridgedBucket"))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return retVal, nil
}

// newFileLogger returns logger instance that writes all logs to a specified file.
// If log file can't be created, it returns an error
func newFileLogger(options *options) (hclog.Logger, error) {
	logFileWriter, err := os.Create(options.logDir)
	if err != nil {
		return nil, fmt.Errorf("could not create log file, %w", err)
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       "bridge-relayer",
		Level:      options.logLevel,
		Output:     logFileWriter,
		JSONFormat: options.jsonLogFormat,
	}), nil
}

// newCLILogger returns minimal logger instance that sends all logs to standard output
func newCLILogger(options *options) hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:       "bridge-relayer",
		Level:      options.logLevel,
		JSONFormat: options.jsonLogFormat,
	})
}

// newLoggerFromConfig creates a new logger which logs to a specified file.
// If log file is not set it outputs to standard output ( console ).
// If log file is specified, and it can't be created the server command will error out
func newLoggerFromConfig(options *options) (hclog.Logger, error) {
	if options.logDir != "" {
		fileLoggerInstance, err := newFileLogger(options)
		if err != nil {
			return nil, err
		}

		return fileLoggerInstance, nil
	}

	return newCLILogger(options), nil
}

func (r *BridgeRelayer) sendSignedBridgeMessageBatch(batch *contractsapi.SignedBridgeMessageBatch) error {
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

		if batch.IsRollback {
			destinationGateway = r.externalGatewayAddr
			destinationRelayer = r.externalClient
		}
	} else {
		sourceGateway = r.internalGatewayAddr
		sourceRelayer = r.internalClient

		destinationGateway = r.externalGatewayAddr
		destinationRelayer = r.externalClient

		if batch.IsRollback {
			destinationGateway = r.internalGatewayAddr
			destinationRelayer = r.internalClient
		}
	}

	messages, err := GetBridgeMessagesInRange(batch.StartID, batch.EndID, sourceRelayer, sourceGateway)
	if err != nil {
		return fmt.Errorf("failed to get messages from source gateway contract, err: %w", err)
	}

	input, err := (&contractsapi.ReceiveBatchGatewayFn{
		BatchMessages:     messages,
		SignedBridgeBatch: batch,
	}).EncodeAbi()
	if err != nil {
		return fmt.Errorf("failed to encode abi, err: %w", err)
	}

	tx := types.NewTx(types.NewLegacyTx(
		types.WithFrom(r.privateKey.Address()),
		types.WithTo(&destinationGateway),
		types.WithInput(input),
	))

	receipt, err := destinationRelayer.SendTransaction(tx, r.privateKey)
	if err != nil {
		return fmt.Errorf("id-ed batch has already been processed or cannot be processed, err: %w", err)
	}

	r.logger.Debug("sent commit bridge message batch transaction to external chain",
		"gatewayAddr", r.externalGatewayAddr,
		"status", types.ReceiptStatus(receipt.Status),
		"txHash", receipt.TransactionHash,
		"blockNumber", receipt.BlockNumber,
	)

	return nil
}

func (r *BridgeRelayer) sendCommitValidatorSet(newValidatorSet *contractsapi.SignedValidatorSet) error {
	input, err := (&contractsapi.CommitValidatorSetBridgeStorageFn{
		NewValidatorSet: newValidatorSet.NewValidatorSet,
		Signature:       newValidatorSet.Signature,
		Bitmap:          newValidatorSet.Bitmap,
		BlockMetadata:   newValidatorSet.BlockMetadata,
	}).EncodeAbi()
	if err != nil {
		return err
	}

	txn := types.NewTx(types.NewLegacyTx(
		types.WithFrom(r.privateKey.Address()),
		types.WithTo(&r.externalGatewayAddr),
		types.WithInput(input),
	))

	receipt, err := r.externalClient.SendTransaction(txn, r.privateKey)
	if err != nil {
		return fmt.Errorf("failed to send commit validator set transaction to external chain, err: %w", err)
	}

	r.logger.Debug("sent commit validator set transaction to external chain",
		"gatewayAddr", r.externalGatewayAddr,
		"status", types.ReceiptStatus(receipt.Status),
		"txHash", receipt.TransactionHash,
		"blockNumber", receipt.BlockNumber,
	)

	txn = types.NewTx(types.NewLegacyTx(
		types.WithFrom(r.privateKey.Address()),
		types.WithTo(&r.internalGatewayAddr),
		types.WithInput(input),
	))

	receipt, err = r.internalClient.SendTransaction(txn, r.privateKey)
	if err != nil {
		return err
	}

	r.logger.Debug("sent commit validator set transaction to internal chain",
		"gatewayAddr", r.internalGatewayAddr,
		"status", types.ReceiptStatus(receipt.Status),
		"txHash", receipt.TransactionHash,
		"blockNumber", receipt.BlockNumber,
	)

	return nil
}
