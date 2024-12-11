package bridge

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"path"
	"sync"

	"github.com/Ethernal-Tech/blockchain-event-tracker/store"
	"github.com/Ethernal-Tech/blockchain-event-tracker/tracker"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/hashicorp/go-hclog"
	"github.com/libp2p/go-libp2p/core/peer"
	bolt "go.etcd.io/bbolt"

	"github.com/0xPolygon/polygon-edge/bls"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/bitmap"
	polychain "github.com/0xPolygon/polygon-edge/consensus/polybft/blockchain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/oracle"
	polybftProto "github.com/0xPolygon/polygon-edge/consensus/polybft/proto"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/signer"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/state"
	systemstate "github.com/0xPolygon/polygon-edge/consensus/polybft/system_state"
	polytypes "github.com/0xPolygon/polygon-edge/consensus/polybft/types"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/validator"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/wallet"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/types"
)

var (
	errUnknownBridgeEvent = errors.New("unknown bridge event")
	errQuorumNotReached   = errors.New("quorum not reached for batch")

	// Bridge events signatures
	bridgeMessageEventSig         = new(contractsapi.BridgeMsgEvent).Sig()
	bridgeBatchResultEventSig     = new(contractsapi.BridgeBatchResultEvent).Sig()
	bridgeMessageResultEventSig   = new(contractsapi.BridgeMessageResultEvent).Sig()
	newBatchEventSig              = new(contractsapi.NewBatchEvent).Sig()
	newValidatorSetStoredEventSig = new(contractsapi.NewValidatorSetStoredEvent).Sig()
)

const maxNumberOfBatchEvents = 10

type Runtime interface {
	IsActiveValidator() bool
}

// BridgeManager is an interface that defines functions for bridge workflow
type BridgeManager interface {
	state.EventSubscriber
	Start(runtimeCfg *config.Runtime) error
	AddLog(chainID *big.Int, eventLog *ethgo.Log) error
	BridgeBatch(blockNumber uint64) ([]*BridgeBatchSigned, error)
	PostBlock(req *oracle.PostBlockRequest) error
	PostEpoch(req *oracle.PostEpochRequest) error
	Close()
}

var _ BridgeManager = (*dummyBridgeEventManager)(nil)

// dummyBridgeEventManager is used when bridge is not enabled
type dummyBridgeEventManager struct{}

func (d *dummyBridgeEventManager) Start(runtimeCfg *config.Runtime) error             { return nil }
func (d *dummyBridgeEventManager) AddLog(chainID *big.Int, eventLog *ethgo.Log) error { return nil }
func (d *dummyBridgeEventManager) BridgeBatch(blockNumber uint64) ([]*BridgeBatchSigned, error) {
	return nil, nil
}
func (d *dummyBridgeEventManager) PostBlock(req *oracle.PostBlockRequest) error { return nil }
func (d *dummyBridgeEventManager) PostEpoch(req *oracle.PostEpochRequest) error {
	return nil
}

// EventSubscriber implementation
func (d *dummyBridgeEventManager) GetLogFilters() map[types.Address][]types.Hash {
	return make(map[types.Address][]types.Hash)
}
func (d *dummyBridgeEventManager) ProcessLog(header *types.Header,
	log *ethgo.Log, dbTx *bolt.Tx) error {
	return nil
}
func (d *dummyBridgeEventManager) Close() {}

// bridgeEventManagerConfig holds the configuration data of bridge event manager
type bridgeEventManagerConfig struct {
	bridgeCfg         *config.Bridge
	topic             Topic
	key               *wallet.Key
	maxNumberOfEvents uint64
}

var _ BridgeManager = (*bridgeEventManager)(nil)

// bridgeEventManager is a struct that manages the workflow of
// saving and querying bridge message events, and creating, and submitting new batches
type bridgeEventManager struct {
	logger hclog.Logger
	state  *BridgeManagerStore

	config *bridgeEventManagerConfig

	// per epoch fields
	lock                         sync.RWMutex
	pendingBridgeBatchesExternal []*PendingBridgeBatch
	pendingBridgeBatchesInternal []*PendingBridgeBatch
	unexecutedBatches            []*PendingBridgeBatch
	rollbackBatches              []*PendingBridgeBatch
	externalClient               jsonrpc.EthClient
	validatorSet                 validator.ValidatorSet
	epoch                        uint64
	nextEventIDExternal          uint64
	nextEventIDInternal          uint64
	externalChainID              uint64
	internalChainID              uint64
	blockchain                   polychain.Blockchain

	runtime Runtime
	tracker *tracker.EventTracker
}

// newBridgeManager creates a new instance of bridge event manager
func newBridgeManager(
	logger hclog.Logger,
	state *BridgeManagerStore,
	config *bridgeEventManagerConfig,
	runtime Runtime,
	externalChainID, internalChainID uint64, blockchain polychain.Blockchain) *bridgeEventManager {
	return &bridgeEventManager{
		logger:          logger,
		state:           state,
		config:          config,
		runtime:         runtime,
		externalChainID: externalChainID,
		internalChainID: internalChainID,
		blockchain:      blockchain,
	}
}

// Start starts the bridge event manager
func (b *bridgeEventManager) Start(runtimeConfig *config.Runtime) error {
	if err := b.initTransport(); err != nil {
		return fmt.Errorf("failed to initialize bridge event transport layer. Error: %w", err)
	}

	tracker, err := b.initTracker(runtimeConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize bridge event tracker. Error: %w", err)
	}

	b.tracker = tracker

	relayer, err := createBridgeTxRelayer(b.config.bridgeCfg.JSONRPCEndpoint, b.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize bridge external client. Error: %w", err)
	}

	b.externalClient = *relayer.Client()

	return nil
}

// internalChainRollbackHandler manages rollback logic for batches that have not been
// successfully executed in the internal chain
func (b *bridgeEventManager) internalChainRollbackHandler(blockNumber *big.Int, dbTx *bolt.Tx) error {
	if err := b.createRollbackBatches(blockNumber, b.externalChainID, b.internalChainID, dbTx); err != nil {
		b.logger.Error("could not create a rollback batches", "err", err)

		return err
	}

	return nil
}

// externalChainRollbackHandler manages rollback logic for batches that have not been
// successfully executed in the internal chain
func (b *bridgeEventManager) externalChainRollbackHandler(dbTx *bolt.Tx) error {
	block, err := b.externalClient.GetBlockByNumber(jsonrpc.BlockNumber(ethgo.Latest), false)
	if err != nil {
		// log the error, but won't return because it might be just a temporary problem
		b.logger.Error("could not poll the block from the external chain", "err", err)
	}

	blockNumber := big.NewInt(int64(block.Header.Number))
	if err := b.createRollbackBatches(blockNumber, b.internalChainID, b.externalChainID, dbTx); err != nil {
		b.logger.Error("could not create a rollback batches", "err", err)

		return nil
	}

	return nil
}

// createRollbackBatches goes through unexecuted batches, checks if any are ready to rollback,
// and if so, initiates the rollback process
func (b *bridgeEventManager) createRollbackBatches(blockNumber *big.Int,
	sourceChainID uint64, destinationChainID uint64, dbTx *bolt.Tx) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i := 0; i < len(b.unexecutedBatches); i++ {
		if b.unexecutedBatches[i].SourceChainID.Uint64() == sourceChainID &&
			b.unexecutedBatches[i].DestinationChainID.Uint64() == destinationChainID &&
			blockNumber.Cmp(b.unexecutedBatches[i].Threshold) >= 0 {
			b.unexecutedBatches[i].IsRollback = true
			b.unexecutedBatches[i].Epoch = b.epoch

			hash, err := b.unexecutedBatches[i].Hash()
			if err != nil {
				return fmt.Errorf("failed to generate hash for (rollback) BridgeBatch. Error: %w", err)
			}

			hashBytes := hash.Bytes()

			signature, err := b.config.key.SignWithDomain(hashBytes, signer.DomainBridge)
			if err != nil {
				return fmt.Errorf("failed to sign (rollback) batch message. Error: %w", err)
			}

			sig := &BridgeBatchVoteConsensusData{
				Sender:    b.config.key.String(),
				Signature: signature,
			}

			if _, err = b.state.insertConsensusData(
				b.epoch,
				hashBytes,
				sig,
				dbTx,
				sourceChainID); err != nil {
				return fmt.Errorf(
					"failed to insert signature for message (rollback) batch to the state. Error: %w",
					err,
				)
			}

			// gossip message
			b.multicast(&BridgeBatchVote{
				Hash: hashBytes,
				BridgeBatchVoteConsensusData: &BridgeBatchVoteConsensusData{
					Signature: signature,
					Sender:    b.config.key.String(),
				},
				EpochNumber:        b.epoch,
				SourceChainID:      sourceChainID,
				DestinationChainID: destinationChainID,
			})

			b.rollbackBatches = append(b.rollbackBatches, b.unexecutedBatches[i])
		}
	}

	return nil
}

// Close stops the bridge manager
func (b *bridgeEventManager) Close() {
	b.tracker.Close()
}

// initTracker starts a new event tracker (to receive bridge events from external chain)
func (b *bridgeEventManager) initTracker(runtimeCfg *config.Runtime) (*tracker.EventTracker, error) {
	store, err := store.NewBoltDBEventTrackerStore(path.Join(runtimeCfg.StateDataDir, "/bridge.db"))
	if err != nil {
		return nil, err
	}

	eventTracker, err := tracker.NewEventTracker(
		&tracker.EventTrackerConfig{
			EventSubscriber:        b,
			Logger:                 b.logger,
			RPCEndpoint:            b.config.bridgeCfg.JSONRPCEndpoint,
			SyncBatchSize:          runtimeCfg.EventTracker.SyncBatchSize,
			NumBlockConfirmations:  runtimeCfg.EventTracker.NumBlockConfirmations,
			NumOfBlocksToReconcile: runtimeCfg.EventTracker.NumOfBlocksToReconcile,
			PollInterval:           runtimeCfg.GenesisConfig.BlockTrackerPollInterval.Duration,
			LogFilter: map[ethgo.Address][]ethgo.Hash{
				ethgo.Address(b.config.bridgeCfg.ExternalGatewayAddr): {bridgeMessageEventSig,
					bridgeBatchResultEventSig, newBatchEventSig},
			},
		},
		store, b.config.bridgeCfg.EventTrackerStartBlocks[b.config.bridgeCfg.ExternalGatewayAddr],
	)

	if err != nil {
		return nil, err
	}

	return eventTracker, eventTracker.Start()
}

// initTransport subscribes to bridge topics (getting votes for batches)
func (b *bridgeEventManager) initTransport() error {
	return b.config.topic.Subscribe(func(obj interface{}, _ peer.ID) {
		if !b.runtime.IsActiveValidator() {
			// don't save votes if not a validator
			return
		}

		msg, ok := obj.(*polybftProto.TransportMessage)
		if !ok {
			b.logger.Warn("failed to deliver vote, invalid msg", "obj", obj)

			return
		}

		var transportMsg *BridgeBatchVote
		if err := json.Unmarshal(msg.Data, &transportMsg); err != nil {
			b.logger.Warn("failed to deliver vote", "error", err)

			return
		}

		if err := b.saveVote(transportMsg); err != nil {
			b.logger.Warn("failed to deliver vote", "error", err)
		}
	})
}

// saveVote saves the gotten vote to boltDb for later quorum check and signature aggregation
func (b *bridgeEventManager) saveVote(vote *BridgeBatchVote) error {
	b.lock.RLock()
	epoch := b.epoch
	valSet := b.validatorSet
	b.lock.RUnlock()

	if valSet == nil || vote.EpochNumber < epoch || vote.EpochNumber > epoch+1 {
		// Epoch metadata is undefined or received a vote for the irrelevant epoch
		return nil
	}

	if !b.isRelevantChainID(vote.SourceChainID) || !b.isRelevantChainID(vote.DestinationChainID) {
		// Vote is for irrelevant chain, skip it
		return nil
	}

	if vote.EpochNumber == epoch+1 {
		if err := b.state.insertEpoch(epoch+1, nil, vote.SourceChainID); err != nil {
			return fmt.Errorf("error saving msg vote from a future epoch: %d. Error: %w", epoch+1, err)
		}
	}

	if err := b.verifyVoteSignature(valSet, types.StringToAddress(vote.Sender), vote.Signature, vote.Hash); err != nil {
		return fmt.Errorf("error verifying vote signature: %w", err)
	}

	msgVote := &BridgeBatchVoteConsensusData{
		Sender:    vote.Sender,
		Signature: vote.Signature,
	}

	numSignatures, err := b.state.insertConsensusData(
		vote.EpochNumber,
		vote.Hash,
		msgVote,
		nil,
		vote.SourceChainID)
	if err != nil {
		return fmt.Errorf("error inserting message vote: %w", err)
	}

	b.logger.Info(
		"deliver message",
		"hash", hex.EncodeToString(vote.Hash),
		"sender", vote.Sender,
		"signatures", numSignatures,
	)

	return nil
}

// isRelevantChainID checks whether internal or external chain id corresponds to the given chain id
func (b *bridgeEventManager) isRelevantChainID(chainID uint64) bool {
	return b.internalChainID == chainID || b.externalChainID == chainID
}

// Verifies signature of the message against the public key of the signer and checks if the signer is a validator
func (b *bridgeEventManager) verifyVoteSignature(valSet validator.ValidatorSet, signerAddr types.Address,
	signature []byte, hash []byte) error {
	validator := valSet.Accounts().GetValidatorMetadata(signerAddr)
	if validator == nil {
		return fmt.Errorf("unable to resolve validator %s", signerAddr)
	}

	unmarshaledSignature, err := bls.UnmarshalSignature(signature)
	if err != nil {
		return fmt.Errorf("failed to unmarshal signature from signer %s, %w", signerAddr.String(), err)
	}

	if !unmarshaledSignature.Verify(validator.BlsKey, hash, signer.DomainBridge) {
		return fmt.Errorf("incorrect signature from %s", signerAddr)
	}

	return nil
}

// AddLog saves the received log from event tracker if it matches a bridge message event ABI
func (b *bridgeEventManager) AddLog(chainID *big.Int, eventLog *ethgo.Log) error {
	switch eventLog.Topics[0] {
	case bridgeMessageEventSig:
		if b.externalChainID != chainID.Uint64() {
			return nil
		}

		event := &contractsapi.BridgeMsgEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch {
			return nil
		}

		b.logger.Info(
			"Add Bridge message event",
			"block", eventLog.BlockNumber,
			"hash", eventLog.TransactionHash,
			"index", eventLog.LogIndex,
		)

		if err != nil {
			b.logger.Error("could not decode bridge message event", "err", err)

			return err
		}

		if err := b.state.insertBridgeMessageEvent(event, nil); err != nil {
			b.logger.Error("could not save bridge message event to boltDb", "err", err)

			return err
		}

		return nil

	case bridgeBatchResultEventSig:
		event := &contractsapi.BridgeBatchResultEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch {
			return nil
		}

		b.logger.Info(
			"Add Bridge batch result event",
			"block", eventLog.BlockNumber,
			"hash", eventLog.TransactionHash,
			"index", eventLog.LogIndex,
		)

		if err != nil {
			b.logger.Error("could not decode bridge batch result event", "err", err)

			return err
		}

		b.lock.Lock()

		for i := 0; i < len(b.unexecutedBatches); {
			if b.unexecutedBatches[i].SourceChainID.Cmp(event.SourceChainID) == 0 &&
				b.unexecutedBatches[i].DestinationChainID.Cmp(event.DestinationChainID) == 0 &&
				b.unexecutedBatches[i].StartID.Cmp(event.StartID) == 0 &&
				b.unexecutedBatches[i].EndID.Cmp(event.EndID) == 0 {
				b.unexecutedBatches = append(b.unexecutedBatches[:i], b.unexecutedBatches[i+1:]...)
			} else {
				i++
			}
		}

		b.lock.Unlock()

		return nil

	default:
		b.logger.Error("unknown bridge event")

		return errUnknownBridgeEvent
	}
}

// BridgeBatch returns a batch to be submitted if there is a pending batch with quorum
func (b *bridgeEventManager) BridgeBatch(blockNumber uint64) ([]*BridgeBatchSigned, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	getLargestPendingBatchFn := func(pendingBatches []*PendingBridgeBatch) (*BridgeBatchSigned, error) {
		var largestBridgeBatch *BridgeBatchSigned

		// we start from the end, since last pending batch is the largest one
		for i := len(pendingBatches) - 1; i >= 0; i-- {
			pendingBatch := pendingBatches[i]
			if (pendingBatch.StartID.Uint64() == b.nextEventIDInternal &&
				pendingBatch.SourceChainID.Uint64() == b.internalChainID) ||
				(pendingBatch.StartID.Uint64() == b.nextEventIDExternal &&
					pendingBatch.SourceChainID.Uint64() == b.externalChainID) {
				aggregatedSignature, err := b.getAggSignatureForBridgeBatchMessage(blockNumber, pendingBatch)
				if err != nil {
					if errors.Is(err, errQuorumNotReached) {
						// a valid case, batch has no quorum, we should not return an error
						if pendingBatch.BridgeBatch.EndID.Uint64()-pendingBatch.BridgeBatch.StartID.Uint64() > 0 {
							b.logger.Debug("can not submit a batch, quorum not reached",
								"from", pendingBatch.BridgeBatch.StartID.Uint64(),
								"to", pendingBatch.BridgeBatch.EndID.Uint64())
						}

						continue
					}

					return nil, err
				}

				largestBridgeBatch = &BridgeBatchSigned{
					BridgeBatch:  pendingBatch.BridgeBatch,
					AggSignature: aggregatedSignature,
				}

				break
			}
		}

		return largestBridgeBatch, nil
	}

	largestExternalBatch, err := getLargestPendingBatchFn(b.pendingBridgeBatchesExternal)
	if err != nil {
		return nil, fmt.Errorf("failed to get largest pending external batch: %w", err)
	}

	largestInternalBatch, err := getLargestPendingBatchFn(b.pendingBridgeBatchesInternal)
	if err != nil {
		return nil, fmt.Errorf("failed to get largest pending internal batch: %w", err)
	}

	signedBridgeBatches := make([]*BridgeBatchSigned, 0)
	if largestExternalBatch != nil {
		signedBridgeBatches = append(signedBridgeBatches, largestExternalBatch)
	}

	if largestInternalBatch != nil {
		signedBridgeBatches = append(signedBridgeBatches, largestInternalBatch)
	}

	rollbackbatches, err := b.getRollbackBatch(blockNumber)
	if err != nil {
		return nil, err
	}

	signedBridgeBatches = append(signedBridgeBatches, rollbackbatches...)

	return signedBridgeBatches, nil
}

func (b *bridgeEventManager) getRollbackBatch(blockNumber uint64) ([]*BridgeBatchSigned, error) {
	seen := make(map[types.Hash]bool)
	result := make([]*BridgeBatchSigned, 0)

	for _, p := range b.rollbackBatches {
		hash, err := p.Hash()
		if err != nil {
			return nil, err
		}

		if !seen[hash] {
			seen[hash] = true

			aggregatedSignature, err := b.getAggSignatureForBridgeBatchMessage(blockNumber, p)
			if err != nil {
				if errors.Is(err, errQuorumNotReached) {
					// a valid case, batch has no quorum, we should not return an error
					if p.BridgeBatch.EndID.Uint64()-p.BridgeBatch.StartID.Uint64() > 0 {
						b.logger.Debug("can not submit a rollback batch, quorum not reached",
							"from", p.BridgeBatch.StartID.Uint64(),
							"to", p.BridgeBatch.EndID.Uint64())
					}

					continue
				}

				return nil, err
			}

			result = append(result, &BridgeBatchSigned{BridgeBatch: p.BridgeBatch, AggSignature: aggregatedSignature})
		}
	}

	return result, nil
}

// getAggSignatureForBridgeBatchMessage checks if pending batch has quorum,
// and if it does, aggregates the signatures
func (b *bridgeEventManager) getAggSignatureForBridgeBatchMessage(blockNumber uint64,
	pendingBridgeBatch *PendingBridgeBatch) (polytypes.Signature, error) {
	validatorSet := b.validatorSet

	validatorAddrToIndex := make(map[string]int, validatorSet.Len())
	validatorsMetadata := validatorSet.Accounts()

	for i, validator := range validatorsMetadata {
		validatorAddrToIndex[validator.Address.String()] = i
	}

	bridgeBatchHash, err := pendingBridgeBatch.Hash()
	if err != nil {
		return polytypes.Signature{}, err
	}

	// get all the votes from the database for batch
	votes, err := b.state.getMessageVotes(
		pendingBridgeBatch.Epoch,
		bridgeBatchHash.Bytes(),
		pendingBridgeBatch.BridgeBatch.SourceChainID.Uint64())
	if err != nil {
		return polytypes.Signature{}, err
	}

	var (
		signatures = make(bls.Signatures, 0, len(votes))
		bmap       = bitmap.Bitmap{}
		signers    = make(map[types.Address]struct{}, 0)
	)

	for _, vote := range votes {
		index, exists := validatorAddrToIndex[vote.Sender]
		if !exists {
			continue // don't count this vote, because it does not belong to validator
		}

		signature, err := bls.UnmarshalSignature(vote.Signature)
		if err != nil {
			return polytypes.Signature{}, err
		}

		bmap.Set(uint64(index))

		signatures = append(signatures, signature)
		signers[types.StringToAddress(vote.Sender)] = struct{}{}
	}

	if !validatorSet.HasQuorum(blockNumber, signers) {
		return polytypes.Signature{}, errQuorumNotReached
	}

	aggregatedSignature, err := signatures.Aggregate().Marshal()
	if err != nil {
		return polytypes.Signature{}, err
	}

	result := polytypes.Signature{
		AggregatedSignature: aggregatedSignature,
		Bitmap:              bmap,
	}

	return result, nil
}

// PostEpoch notifies the bridge event manager that an epoch has changed,
// so that it can discard any previous epoch bridge batch, and build a new one (since validator set changed)
func (b *bridgeEventManager) PostEpoch(req *oracle.PostEpochRequest) error {
	if err := b.state.insertEpoch(req.NewEpochID, req.DBTx, b.externalChainID); err != nil {
		return fmt.Errorf("an error occurred while inserting new epoch in db, chainID: %d. Reason: %w",
			b.externalChainID, err)
	}

	if err := b.state.insertEpoch(req.NewEpochID, req.DBTx, b.internalChainID); err != nil {
		return fmt.Errorf("an error occurred while inserting new epoch in db, chainID: %d. Reason: %w",
			b.internalChainID, err)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.pendingBridgeBatchesExternal = nil
	b.pendingBridgeBatchesInternal = nil
	b.rollbackBatches = nil
	b.validatorSet = req.ValidatorSet
	b.epoch = req.NewEpochID

	return nil
}

// PostBlock creates batch from internal events.
func (b *bridgeEventManager) PostBlock(req *oracle.PostBlockRequest) error {
	if req.FullBlock.Block.Header.Number > 1 {
		provider, err := b.blockchain.GetStateProviderForBlock(req.FullBlock.Block.Header)
		if err != nil {
			return err
		}

		systemState := b.blockchain.GetSystemState(provider)

		b.nextEventIDInternal, err = systemState.GetNextCommittedIndex(b.externalChainID, systemstate.Internal)
		if err != nil {
			return err
		}

		b.nextEventIDExternal, err = systemState.GetNextCommittedIndex(b.externalChainID, systemstate.External)
		if err != nil {
			return err
		}
	}

	if err := b.buildInternalBridgeBatch(req.DBTx); err != nil {
		// we don't return an error here. If bridge message event is inserted in db,
		// we will just try to build a batch on next block or next event arrival
		b.logger.Error("could not build an internal chain originated batch on PostBlock",
			"err", err)
	}

	if err := b.buildExternalBridgeBatch(req.DBTx); err != nil {
		// we don't return an error here. If bridge message event is inserted in db,
		// we will just try to build a batch on next block or next event arrival
		b.logger.Error("could not build an external chain originated batch on PostBlock",
			"err", err)
	}

	if err := b.internalChainRollbackHandler(big.NewInt(int64(req.FullBlock.Block.Number())), req.DBTx); err != nil {
		// we don't return an error here. If threshold is less than block number,
		// we will just try to build a batch on next block or next event arrival
		b.logger.Error("could not build an external chain originated batch on PostBlock",
			"err", err)
	}

	if err := b.externalChainRollbackHandler(req.DBTx); err != nil {
		// we don't return an error here. If threshold is less than block number,
		// we will just try to build a batch on next block or next event arrival
		b.logger.Error("could not build an external chain originated batch on PostBlock",
			"err", err)
	}

	return nil
}

// buildExternalBridgeBatch builds a new external bridge batch, signs it and gossips its vote for it
func (b *bridgeEventManager) buildExternalBridgeBatch(dbTx *bolt.Tx) error {
	return b.buildBridgeBatch(dbTx, b.externalChainID, b.internalChainID, b.nextEventIDExternal)
}

// buildInternalBridgeBatch builds a new internal bridge batch, signs it and gossips its vote for it
func (b *bridgeEventManager) buildInternalBridgeBatch(dbTx *bolt.Tx) error {
	return b.buildBridgeBatch(dbTx, b.internalChainID, b.externalChainID, b.nextEventIDInternal)
}

func (b *bridgeEventManager) buildBridgeBatch(
	dbTx *bolt.Tx,
	sourceChainID, destinationChainID uint64,
	nextBridgeEventIDIndex uint64) error {
	if !b.runtime.IsActiveValidator() {
		// don't build batch if not a validator
		return nil
	}

	var pendingBridgeBatches []*PendingBridgeBatch

	b.lock.RLock()

	externalThreshold := false

	if sourceChainID == b.externalChainID {
		pendingBridgeBatches = b.pendingBridgeBatchesExternal
	} else if sourceChainID == b.internalChainID {
		externalThreshold = true
		pendingBridgeBatches = b.pendingBridgeBatchesInternal
	}

	// Since lock is reduced grab original values into local variables in order to keep them
	epoch := b.epoch
	bridgeMessageEvents, err := b.state.getBridgeMessageEventsForBridgeBatch(
		nextBridgeEventIDIndex,
		nextBridgeEventIDIndex+b.config.maxNumberOfEvents-1,
		dbTx,
		sourceChainID, destinationChainID)

	if err != nil && !errors.Is(err, errNotEnoughBridgeEvents) {
		b.lock.RUnlock()

		return fmt.Errorf("failed to get bridge message event for batch. Error: %w", err)
	}

	if len(bridgeMessageEvents) == 0 {
		// there are no bridge message events
		b.lock.RUnlock()

		return nil
	}

	if len(pendingBridgeBatches) > 0 &&
		pendingBridgeBatches[len(pendingBridgeBatches)-1].
			BridgeBatch.EndID.
			Cmp(bridgeMessageEvents[len(bridgeMessageEvents)-1].ID) >= 0 {
		// already built a bridge batch of this size which is pending to be submitted
		b.lock.RUnlock()

		return nil
	}

	b.lock.RUnlock()

	var blockNumber uint64

	if externalThreshold {
		block, err := b.externalClient.GetBlockByNumber(jsonrpc.BlockNumber(ethgo.Latest), false)
		if err != nil {
			return err
		}

		blockNumber = block.Number()
	} else {
		blockNumber = b.blockchain.CurrentHeader().Number
	}

	pendingBridgeBatch, err := NewPendingBridgeBatch(epoch, bridgeMessageEvents)
	if err != nil {
		return err
	}

	pendingBridgeBatch.Threshold = new(big.Int).SetUint64(
		uint64((math.Ceil(float64(blockNumber)/10) * 10)) + b.config.bridgeCfg.BridgeBatchThreshold)

	hash, err := pendingBridgeBatch.Hash()
	if err != nil {
		return fmt.Errorf("failed to generate hash for BridgeBatch. Error: %w", err)
	}

	hashBytes := hash.Bytes()

	signature, err := b.config.key.SignWithDomain(hashBytes, signer.DomainBridge)
	if err != nil {
		return fmt.Errorf("failed to sign batch message. Error: %w", err)
	}

	sig := &BridgeBatchVoteConsensusData{
		Sender:    b.config.key.String(),
		Signature: signature,
	}

	if _, err = b.state.insertConsensusData(
		epoch,
		hashBytes,
		sig,
		dbTx,
		sourceChainID); err != nil {
		return fmt.Errorf(
			"failed to insert signature for message batch to the state. Error: %w",
			err,
		)
	}

	// gossip message
	b.multicast(&BridgeBatchVote{
		Hash: hashBytes,
		BridgeBatchVoteConsensusData: &BridgeBatchVoteConsensusData{
			Signature: signature,
			Sender:    b.config.key.String(),
		},
		EpochNumber:        epoch,
		SourceChainID:      sourceChainID,
		DestinationChainID: destinationChainID,
	})

	if pendingBridgeBatch.BridgeBatch.EndID.Uint64()-pendingBridgeBatch.BridgeBatch.StartID.Uint64() > 0 {
		b.logger.Debug(
			"[buildBridgeBatch] build batch",
			"from", pendingBridgeBatch.BridgeBatch.StartID.Uint64(),
			"to", pendingBridgeBatch.BridgeBatch.EndID.Uint64(),
		)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	pendingBridgeBatches = append(pendingBridgeBatches, pendingBridgeBatch)

	if sourceChainID == b.externalChainID {
		b.pendingBridgeBatchesExternal = pendingBridgeBatches
	} else if sourceChainID == b.internalChainID {
		b.pendingBridgeBatchesInternal = pendingBridgeBatches
	}

	return nil
}

// multicast publishes given message to the rest of the network
func (b *bridgeEventManager) multicast(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		b.logger.Warn("failed to marshal bridge message", "err", err)

		return
	}

	err = b.config.topic.Publish(&polybftProto.TransportMessage{Data: data})
	if err != nil {
		b.logger.Warn("failed to gossip bridge message", "err", err)
	}
}

// EventSubscriber implementation

// GetLogFilters returns a map of log filters for getting desired events,
// where the key is the address of contract that emits desired events,
// and the value is a slice of signatures of events we want to get.
// This function is the implementation of EventSubscriber interface
func (b *bridgeEventManager) GetLogFilters() map[types.Address][]types.Hash {
	return map[types.Address][]types.Hash{
		b.config.bridgeCfg.InternalGatewayAddr: {
			types.Hash(bridgeMessageEventSig),
			types.Hash(bridgeMessageResultEventSig),
			types.Hash(bridgeBatchResultEventSig),
		},
		contracts.BridgeStorageContract: {
			types.Hash(newBatchEventSig),
		},
	}
}

// ProcessLog is the implementation of EventSubscriber interface,
// used to handle a log defined in GetLogFilters, provided by event provider
func (b *bridgeEventManager) ProcessLog(header *types.Header, log *ethgo.Log, dbTx *bolt.Tx) error {
	switch log.Topics[0] {
	case bridgeMessageResultEventSig:
		var bridgeMessageResultEvent contractsapi.BridgeMessageResultEvent

		doesMatch, err := bridgeMessageResultEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if !doesMatch || b.externalChainID != bridgeMessageResultEvent.SourceChainID.Uint64() {
			return nil
		}

		if bridgeMessageResultEvent.Status {
			return b.state.removeBridgeEvents(bridgeMessageResultEvent, dbTx)
		}

		return nil
	case bridgeMessageEventSig:
		event := &contractsapi.BridgeMsgEvent{}

		doesMatch, err := event.ParseLog(log)
		if !doesMatch || b.externalChainID != event.DestinationChainID.Uint64() {
			return nil
		}

		if err != nil {
			b.logger.Error("could not decode bridge message event", "err", err)

			return err
		}

		if err := b.state.insertBridgeMessageEvent(event, dbTx); err != nil {
			b.logger.Error("could not save bridge message event to boltDb", "err", err)

			return err
		}

		return nil

	case bridgeBatchResultEventSig:
		event := &contractsapi.BridgeBatchResultEvent{}

		doesMatch, err := event.ParseLog(log)
		if !doesMatch || event.SourceChainID.Uint64() != b.externalChainID {
			return nil
		}

		b.logger.Info(
			"Add Bridge batch result event",
			"block", log.BlockNumber,
			"hash", log.TransactionHash,
			"index", log.LogIndex,
		)

		if err != nil {
			b.logger.Error("could not decode bridge batch result event", "err", err)

			return err
		}

		b.lock.Lock()

		for i := 0; i < len(b.unexecutedBatches); {
			if b.unexecutedBatches[i].SourceChainID.Cmp(event.SourceChainID) == 0 &&
				b.unexecutedBatches[i].DestinationChainID.Cmp(event.DestinationChainID) == 0 &&
				b.unexecutedBatches[i].StartID.Cmp(event.StartID) == 0 &&
				b.unexecutedBatches[i].EndID.Cmp(event.EndID) == 0 {
				b.unexecutedBatches = append(b.unexecutedBatches[:i], b.unexecutedBatches[i+1:]...)
			} else {
				i++
			}
		}

		b.lock.Unlock()

		return nil

	case newBatchEventSig:
		var newBatchEvent contractsapi.NewBatchEvent

		doesMatch, err := newBatchEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if !doesMatch {
			return nil
		}

		provider, err := b.blockchain.GetStateProviderForBlock(header)
		if err != nil {
			return err
		}

		ss := systemstate.NewSystemState(contracts.EpochManagerContract, contracts.BridgeStorageContract, provider)

		bridgeBatch, err := ss.GetBridgeBatchByNumber(newBatchEvent.ID)
		if err != nil {
			return err
		}

		b.lock.Lock()

		if !bridgeBatch.IsRollback {
			b.unexecutedBatches = append(b.unexecutedBatches, &PendingBridgeBatch{
				BridgeBatch: &contractsapi.BridgeBatch{
					RootHash:           bridgeBatch.RootHash,
					StartID:            bridgeBatch.StartID,
					EndID:              bridgeBatch.EndID,
					SourceChainID:      bridgeBatch.SourceChainID,
					DestinationChainID: bridgeBatch.DestinationChainID,
					Threshold:          bridgeBatch.Threshold,
					IsRollback:         bridgeBatch.IsRollback,
				},
				Epoch: b.epoch,
			})
		}

		b.lock.Unlock()

		return nil

	default:
		b.logger.Error("unknown bridge event")

		return errUnknownBridgeEvent
	}
}
