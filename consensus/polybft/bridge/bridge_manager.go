package bridge

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"path"
	"strings"
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
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
)

var (
	errUnknownBridgeEvent = errors.New("unknown bridge event")
	errQuorumNotReached   = errors.New("quorum not reached for batch")

	// Bridge events signatures
	bridgeMessageEventSig         = new(contractsapi.BridgeMsgEvent).Sig()
	bridgeBatchProcessedEventSig  = new(contractsapi.BridgeBatchProcessedEvent).Sig()
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
	GetInternalGatewayAddr() types.Address
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

func (d *dummyBridgeEventManager) GetInternalGatewayAddr() types.Address { return [20]byte{} }

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
	lock                    sync.RWMutex
	pendingBridgeBatchesE2I []*PendingBridgeBatch
	pendingBridgeBatchesI2E []*PendingBridgeBatch
	unexecutedBatches       []*PendingBridgeBatch
	retryBatches            map[types.Hash]PendingBridgeBatch
	pendingRetryBatches     map[types.Hash][]*PendingBridgeBatch
	externalClient          jsonrpc.EthClient
	validatorSet            validator.ValidatorSet
	epoch                   uint64
	nextEventIDE2I          uint64
	nextEventIDI2E          uint64
	externalChainID         uint64
	internalChainID         uint64
	blockchain              polychain.Blockchain

	runtime                   Runtime
	tracker                   *tracker.EventTracker
	externalConfirmationDepth *big.Int
}

// newBridgeManager creates a new instance of bridge event manager
func newBridgeManager(
	logger hclog.Logger,
	state *BridgeManagerStore,
	config *bridgeEventManagerConfig,
	runtime Runtime,
	externalChainID, internalChainID uint64, blockchain polychain.Blockchain) *bridgeEventManager {
	return &bridgeEventManager{
		logger:              logger,
		state:               state,
		config:              config,
		retryBatches:        make(map[types.Hash]PendingBridgeBatch),
		pendingRetryBatches: make(map[types.Hash][]*PendingBridgeBatch),
		runtime:             runtime,
		externalChainID:     externalChainID,
		internalChainID:     internalChainID,
		blockchain:          blockchain,
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
	b.externalConfirmationDepth = big.NewInt(int64(runtimeConfig.EventTracker.NumBlockConfirmations))

	relayer, err := createBridgeTxRelayer(b.config.bridgeCfg.JSONRPCEndpoint, b.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize bridge external client. Error: %w", err)
	}

	b.externalClient = *relayer.Client()

	return nil
}

func (b *bridgeEventManager) GetInternalGatewayAddr() types.Address {
	return b.config.bridgeCfg.InternalGatewayAddr
}

// Close stops the bridge manager
func (b *bridgeEventManager) Close() {
	b.tracker.Close()
}

// initTracker starts a new event tracker (to receive bridge events from external chain)
func (b *bridgeEventManager) initTracker(runtimeCfg *config.Runtime) (*tracker.EventTracker, error) {
	store, err := store.NewBoltDBEventTrackerStore(path.Join(runtimeCfg.StateDataDir,
		fmt.Sprintf("/bridge-%d.db", b.externalChainID)))
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
					bridgeBatchProcessedEventSig, newBatchEventSig, bridgeMessageResultEventSig},
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

// BridgeBatch returns a batch to be submitted if there is a pending batch with quorum
func (b *bridgeEventManager) BridgeBatch(blockNumber uint64) ([]*BridgeBatchSigned, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	getLargestPendingBatchFn := func(pendingBatches []*PendingBridgeBatch, sourceChainId uint64) (*BridgeBatchSigned,
		error) {
		var (
			largestBridgeBatch *BridgeBatchSigned
			err                error
		)

		// We start from the end, since last pending batch is the most relevant one.
		for i := len(pendingBatches) - 1; i >= 0; i-- {
			if pendingBatches[i].SourceChainID.Uint64() == sourceChainId {
				var aggregatedSignature polytypes.Signature

				aggregatedSignature, err = b.getAggSignatureForBridgeBatchMessage(blockNumber, pendingBatches[i])
				if err != nil {
					if errors.Is(err, errQuorumNotReached) {
						err = nil

						// Consider potential logging.

						continue
					}

					break
				}

				largestBridgeBatch = &BridgeBatchSigned{
					BridgeMessageBatch: pendingBatches[i].BridgeMessageBatch,
					AggSignature:       aggregatedSignature,
					InternalChainID:    b.internalChainID,
				}

				break
			}
		}

		return largestBridgeBatch, err
	}

	largestI2EBatch, err := getLargestPendingBatchFn(b.pendingBridgeBatchesI2E, b.internalChainID)
	if err != nil {
		return nil, fmt.Errorf("could not get largest internal-to-external pending batch. Error: %w", err)
	}

	largestE2IBatch, err := getLargestPendingBatchFn(b.pendingBridgeBatchesE2I, b.externalChainID)
	if err != nil {
		return nil, fmt.Errorf("could not get largest external-to-internal pending batch. Error: %w", err)
	}

	signedBridgeBatches := make([]*BridgeBatchSigned, 0, 2)

	if largestI2EBatch != nil {
		signedBridgeBatches = append(signedBridgeBatches, largestI2EBatch)
	}

	if largestE2IBatch != nil {
		signedBridgeBatches = append(signedBridgeBatches, largestE2IBatch)
	}

	for _, pendingRetryBatches := range b.pendingRetryBatches {
		largestRetryBatch, err := getLargestPendingBatchFn(pendingRetryBatches, b.internalChainID)
		if err != nil {
			return nil, fmt.Errorf("could not get largest retry pending batch. Error: %w", err)
		}

		if largestRetryBatch != nil {
			signedBridgeBatches = append(signedBridgeBatches, largestRetryBatch)
		}
	}

	return signedBridgeBatches, nil
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
		pendingBridgeBatch.BridgeMessageBatch.SourceChainID.Uint64())
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

	b.pendingBridgeBatchesE2I = nil
	b.pendingBridgeBatchesI2E = nil
	b.validatorSet = req.ValidatorSet
	b.epoch = req.NewEpochID

	return nil
}

// PostBlock creates batch from internal events.
func (b *bridgeEventManager) PostBlock(req *oracle.PostBlockRequest) error {
	var systemState systemstate.SystemState

	if req.FullBlock.Block.Header.Number > 1 {
		provider, err := b.blockchain.GetStateProviderForBlock(req.FullBlock.Block.Header)
		if err != nil {
			return err
		}

		systemState = b.blockchain.GetSystemState(provider)

		b.nextEventIDI2E, err = systemState.GetNextCommittedIndex(b.externalChainID, systemstate.Internal)
		if err != nil {
			return err
		}

		b.nextEventIDE2I, err = systemState.GetNextCommittedIndex(b.externalChainID, systemstate.External)
		if err != nil {
			return err
		}
	}

	if err := b.buildI2EBridgeBatch(req.DBTx); err != nil {
		// we don't return an error here. If bridge message event is inserted in db,
		// we will just try to build a batch on next block or next event arrival
		b.logger.Error("could not build an internal chain originated batch on PostBlock",
			"err", err)
	}

	if err := b.buildE2IBridgeBatch(req.DBTx); err != nil {
		// we don't return an error here. If bridge message event is inserted in db,
		// we will just try to build a batch on next block or next event arrival
		b.logger.Error("could not build an external chain originated batch on PostBlock",
			"err", err)
	}

	b.handleRetry(req.DBTx, systemState)

	return nil
}

// buildE2IBridgeBatch builds a new external to internal bridge batch, signs it and gossips its vote for it
func (b *bridgeEventManager) buildE2IBridgeBatch(dbTx *bolt.Tx) error {
	return b.buildBridgeBatch(dbTx, b.externalChainID, b.internalChainID, b.nextEventIDE2I)
}

// buildI2EBridgeBatch builds a new internal to external bridge batch, signs it and gossips its vote for it
func (b *bridgeEventManager) buildI2EBridgeBatch(dbTx *bolt.Tx) error {
	return b.buildBridgeBatch(dbTx, b.internalChainID, b.externalChainID, b.nextEventIDI2E)
}

func (b *bridgeEventManager) buildBridgeBatch(
	dbTx *bolt.Tx,
	sourceChainID, destinationChainID uint64,
	nextBridgeEventIDIndex uint64) error {
	if !b.runtime.IsActiveValidator() {
		return nil
	}

	epoch := b.epoch
	messages, numOfOrdinaryMsgs, err := b.state.getBridgeMessages(
		nextBridgeEventIDIndex,
		b.config.maxNumberOfEvents,
		sourceChainID,
		destinationChainID,
		dbTx)

	if err != nil {
		return fmt.Errorf("could not get bridge messages to create batch. Error: %w", err)
	}

	if len(messages) == 0 {
		return nil
	}

	var blockNumber uint64

	if sourceChainID == b.internalChainID {
		block, err := b.externalClient.GetBlockByNumber(jsonrpc.BlockNumber(ethgo.Latest), false)
		if err != nil {
			return err
		}

		blockNumber = block.Number()
	} else {
		// We generally don't need to calculate a threshold for E2I batches, as they are executed
		// immediately, however it's left in for consistency.
		blockNumber = b.blockchain.CurrentHeader().Number
	}

	pendingBatch := &PendingBridgeBatch{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           messages,
			SourceChainID:      big.NewInt(int64(sourceChainID)),
			DestinationChainID: big.NewInt(int64(destinationChainID)),
			Threshold: new(big.Int).SetUint64(
				uint64((math.Ceil(float64(blockNumber)/10) * 10)) + b.config.bridgeCfg.BridgeBatchThreshold),
			NumberOfRegularEvents: big.NewInt(int64(numOfOrdinaryMsgs)),
			CommitCounter:         big.NewInt(1),
		},
		Epoch: epoch,
	}

	hash, err := pendingBatch.Hash()
	if err != nil {
		return fmt.Errorf("could not generate a hash for bridge batch. Error: %w", err)
	}

	hashBytes := hash.Bytes()

	signature, err := b.config.key.SignWithDomain(hashBytes, signer.DomainBridge)
	if err != nil {
		return fmt.Errorf("could not create a signature for bridge batch. Error: %w", err)
	}

	sig := &BridgeBatchVoteConsensusData{
		Sender:    b.config.key.String(),
		Signature: signature,
	}

	if _, err = b.state.insertConsensusData(epoch, hashBytes, sig, dbTx, sourceChainID); err != nil {
		return fmt.Errorf("could not insert signature for bridge batch. Error: %w", err)
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

	sid := pendingBatch.SourceChainID
	did := pendingBatch.DestinationChainID

	firstID := big.NewInt(0)
	lastID := big.NewInt(0)

	if numOfOrdinaryMsgs > 0 {
		firstID = pendingBatch.Messages[0].ID
		lastID = pendingBatch.Messages[-1+int(numOfOrdinaryMsgs)].ID
	}

	b.logger.Info(
		"New (pending) Bridge batch created and multicasted",
		"direction", fmt.Sprintf("%d -> %d", sid.Uint64(), did.Uint64()),
		"total number of messages", len(pendingBatch.Messages),
		"number of ordinary", fmt.Sprintf("%d (%s-%s)", numOfOrdinaryMsgs, firstID.String(), lastID.String()),
		"number of rollback", len(pendingBatch.Messages)-int(numOfOrdinaryMsgs),
		"threshold", pendingBatch.Threshold,
		"hash", hash.String(),
	)

	b.lock.Lock()

	if sourceChainID == b.internalChainID {
		b.pendingBridgeBatchesI2E = append(b.pendingBridgeBatchesI2E, pendingBatch)
	} else {
		b.pendingBridgeBatchesE2I = append(b.pendingBridgeBatchesE2I, pendingBatch)
	}

	b.lock.Unlock()

	return nil
}

func (b *bridgeEventManager) handleRetry(dbTx *bolt.Tx, systemState systemstate.SystemState) {
	block, err := b.externalClient.GetBlockByNumber(jsonrpc.BlockNumber(ethgo.Latest), false)
	if err != nil {
		// Log the error, but won't return because it might be just a temporary problem.
		b.logger.Error("could not poll the block from the external chain", "err", err)

		return
	}

	blockNumber := big.NewInt(int64(block.Header.Number))

	b.lock.Lock()
	defer b.lock.Unlock()

	// If an error occurs inside the loop, we simply log it and proceed to the next iteration.
	// We do not return immediately upon encountering an error because we want to process all
	// batches from the unexecuted list. The iterator `i` does not increment when a retry batch
	// is successfully created, as the unexecuted list shrinks by one, making the next batch (i+1)
	// the current (i).
	for i := 0; i < len(b.unexecutedBatches); {
		rt := big.NewInt(0).Add(b.unexecutedBatches[i].Threshold, b.externalConfirmationDepth)

		// We add a random small number (salt) for extra security. This is not required.
		rt.Add(rt, big.NewInt(2))

		if blockNumber.Cmp(rt) >= 0 {
			retryBatch := *b.unexecutedBatches[i]
			retryBatch.NumberOfRegularEvents = big.NewInt(0)
			retryBatch.Threshold = big.NewInt(0)
			retryBatch.CommitCounter = big.NewInt(0)

			hash, err := retryBatch.Hash()
			if err != nil {
				b.logger.Error("could not generate a hash for retry bridge batch", "err", err)

				i++

				continue
			}

			numOfTries, err := systemState.GetBatchCommitCounter(hash)
			if err != nil {
				b.logger.Error("could not get a number of retries for bridge batch", "err", err)

				i++

				continue
			}

			// Since the batch has exceeded its threshold and can no longer be executed on the
			// destination chain, we remove it from the unexecuted list and move it to the map
			// of batches for which a new (retry) batch needs to be created.

			b.unexecutedBatches = append(b.unexecutedBatches[:i], b.unexecutedBatches[i+1:]...)

			retryBatch.CommitCounter = numOfTries.Add(numOfTries, big.NewInt(1))
			b.retryBatches[hash] = retryBatch
			b.pendingRetryBatches[hash] = []*PendingBridgeBatch{}
		} else {
			i++
		}
	}

	if !b.runtime.IsActiveValidator() {
		return
	}

	for hash := range b.retryBatches {
		err = b.buildRetryBridgeBatch(hash, blockNumber.Uint64(), dbTx)
		if err != nil {
			b.logger.Error("could not create retry bridge batch", "err", err)
		}
	}
}

func (b *bridgeEventManager) buildRetryBridgeBatch(primaryHash types.Hash, blockNumber uint64, dbTx *bolt.Tx) error {
	rb := b.retryBatches[primaryHash]
	pendingBatch := PendingBridgeBatch{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           rb.Messages,
			SourceChainID:      rb.SourceChainID,
			DestinationChainID: rb.DestinationChainID,
			Threshold: new(big.Int).SetUint64(
				uint64((math.Ceil(float64(blockNumber)/10) * 10)) + b.config.bridgeCfg.BridgeBatchThreshold),
			NumberOfRegularEvents: rb.NumberOfRegularEvents,
			CommitCounter:         rb.CommitCounter,
		},
		Epoch: b.epoch,
	}

	hash, err := pendingBatch.Hash()
	if err != nil {
		return fmt.Errorf("could not generate a hash for retry bridge batch. Error: %w", err)
	}

	hashBytes := hash.Bytes()

	signature, err := b.config.key.SignWithDomain(hashBytes, signer.DomainBridge)
	if err != nil {
		return fmt.Errorf("could not create a signature for retry bridge batch. Error: %w", err)
	}

	sig := &BridgeBatchVoteConsensusData{
		Sender:    b.config.key.String(),
		Signature: signature,
	}

	if _, err = b.state.insertConsensusData(b.epoch, hashBytes, sig, dbTx, b.internalChainID); err != nil {
		return fmt.Errorf("could not insert signature for retry bridge batch. Error: %w", err)
	}

	// gossip message
	b.multicast(&BridgeBatchVote{
		Hash: hashBytes,
		BridgeBatchVoteConsensusData: &BridgeBatchVoteConsensusData{
			Signature: signature,
			Sender:    b.config.key.String(),
		},
		EpochNumber:        b.epoch,
		SourceChainID:      b.internalChainID,
		DestinationChainID: b.externalChainID,
	})

	pendingRetryBatches := b.pendingRetryBatches[primaryHash]
	b.pendingRetryBatches[primaryHash] = append(pendingRetryBatches, &pendingBatch)

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
			types.Hash(bridgeBatchProcessedEventSig),
		},
		contracts.BridgeStorageContract: {
			types.Hash(newBatchEventSig),
		},
	}
}

// ProcessLog method is responsible for processing bridge events originating from the internal (Blade)
// chain. An event provider is responsible for collecting these events.
func (b *bridgeEventManager) ProcessLog(header *types.Header, eventLog *ethgo.Log, dbTx *bolt.Tx) error {
	isEventMine := func(chainID *big.Int) bool {
		// If the event (bridge message) is related to an external chain that is not managed by the
		// current bridge manager, it should be immediately discarded.
		if b.externalChainID != chainID.Uint64() {
			return false
		}

		return true
	}

	// We process four types of events:
	// 1. bridgeMessageEventSig (`BridgeMsg`)
	//	  - This event is emitted by the `sendBridgeMsg` method of the Gateway smart contract on
	//	 	an internal chain. It is triggered when users submit a transaction to transfer tokens
	//  	from the internal to the external chain. The only required action in this case is to
	//		store the event (bridge message) in the local database.
	// 2. bridgeMessageResultEventSig (`BridgeMessageResult`)
	//	  - This event is emitted by the `_executeBridgeMessage` or `_executeRollbackBridgeMessage`
	// 		method of the Gateway smart contract on an internal chain. It is triggered when a normal
	//		(ordinary) or rollback message is transferred from the external to the internal chain
	//		and executed there. The processing of this event depends on whether it is a normal or
	//		rollback message, as well as whether the message was successfully executed or not.
	// 3. bridgeBatchProcessedEventSig (`BridgeBatchProcessed`)
	//	  - This event is emitted by the `receiveBatch` method of the Gateway smart contract on an
	// 		internal chain. It is triggered after a batch transferring messages from the external
	// 		to the internal chain has been successfully bridged and validated on the internal chain.
	// 		Success of executing individual messages from a batch do not affect this event in any
	// 		way. Since external-to-internal batches are executed immediately after the quorum number
	//		of signatures is obtained, and therefore these batches are not stored in the unexecuted
	// 		list, no additional steps need to be taken. Event is used for logging purposes only.
	// 4. newBatchEventSig (`NewBatch`)
	//	  - This event is emitted by the `commitBatch` method of the BridgeStorage smart contract on
	//		an internal chain. It is triggered after the quorum number of signatures for a batch is
	//		collected and the block proposer on the sprint block commits the given batch. The actions
	//		to be taken depend on the batch direction and whether the batch is being committed for the
	//		first time, i.e. whether it is a retry batch or not. We add a batch to the unexecuted list
	//		only if it is an outgoing batch, i.e. a batch that goes to and is executed on an external
	//		chain. Incoming batches (external-to-internal batches) are executed immediately upon receiving
	//		a quorum of votes (execution and commit to bridge storage occur within the same transaction),
	//		thus adding to unexecuted list is not required. Additionally, regardless of direction, all
	//		rollback messages that were committed in a given batch are deleted from the local database,
	//		but only if the batch is being committed for the first time (not a retry batch). Also, in
	//		case it is a retry batch, it is deleted from the retry map together with all its pending
	//		retry batches. Otherwise, if it is a regular batch, then, depending on the batch direction
	// 		(whether it goes from the internal to the external chain or vice-verse) the I2E/E2I pending
	//		list is restarted (cleared).
	switch eventLog.Topics[0] {
	case bridgeMessageEventSig:
		event := &contractsapi.BridgeMsgEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge message event", "err", err)

			return err
		}

		if !isEventMine(event.DestinationChainID) {
			return nil
		}

		if err := b.handleBridgeMessageEvent(event, dbTx); err != nil {
			return err
		}

	case bridgeMessageResultEventSig:
		event := &contractsapi.BridgeMessageResultEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge message result event", "err", err)

			return err
		}

		if !isEventMine(event.SourceChainID) {
			return nil
		}

		if err := b.handleBridgeMessageResultEvent(event, dbTx); err != nil {
			return err
		}

	case bridgeBatchProcessedEventSig:
		event := &contractsapi.BridgeBatchProcessedEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge batch processed event", "err", err)

			return err
		}

		if !isEventMine(event.SourceChainID) {
			return nil
		}

		sid := event.SourceChainID
		did := event.DestinationChainID

		b.logger.Info(
			"Caught Bridge batch processed event",
			"success", event.Success,
			"hash", fmt.Sprintf("%v", event.BatchHash),
			"direction", fmt.Sprintf("%d -> %d", sid.Uint64(), did.Uint64()),
		)

	case newBatchEventSig:
		event := &contractsapi.NewBatchEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge batch processed event", "err", err)

			return err
		}

		provider, err := b.blockchain.GetStateProviderForBlock(header)
		if err != nil {
			return err
		}

		ss := systemstate.NewSystemState(contracts.EpochManagerContract, contracts.BridgeStorageContract, provider)

		bridgeBatch, err := ss.GetBridgeBatchByNumber(event.ID)
		if err != nil {
			return err
		}

		if !isEventMine(bridgeBatch.Batch.SourceChainID) && !isEventMine(bridgeBatch.Batch.DestinationChainID) {
			return nil
		}

		b.logger.Info(
			"Caught New batch event",
			"ID", event.ID.String(),
		)

		sid := bridgeBatch.Batch.SourceChainID
		did := bridgeBatch.Batch.DestinationChainID

		if bridgeBatch.Batch.SourceChainID.Uint64() == b.internalChainID &&
			bridgeBatch.Batch.DestinationChainID.Uint64() == b.externalChainID {
			b.lock.Lock()

			unexecutedBatch := &PendingBridgeBatch{
				BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
					Messages:              bridgeBatch.Batch.Messages,
					SourceChainID:         bridgeBatch.Batch.SourceChainID,
					DestinationChainID:    bridgeBatch.Batch.DestinationChainID,
					Threshold:             big.NewInt(0),
					NumberOfRegularEvents: big.NewInt(0),
					CommitCounter:         big.NewInt(0),
				},
				Epoch: b.epoch,
			}

			if bridgeBatch.Batch.CommitCounter.Cmp(big.NewInt(1)) > 0 {
				hash, err := unexecutedBatch.Hash()
				if err != nil {
					b.logger.Error("could not calculate a hash for the bridge batch", "err", err)

					b.lock.Unlock()

					return err
				}

				delete(b.retryBatches, hash)
				delete(b.pendingRetryBatches, hash)

				b.logger.Info(fmt.Sprintf("Batch (%s, %s, %d, %d -> %d) has been successfully removed from the retry map",
					event.ID.String(), bridgeBatch.Batch.CommitCounter.String(),
					len(bridgeBatch.Batch.Messages), sid.Uint64(), did.Uint64()))
			} else {
				if unexecutedBatch.SourceChainID.Cmp(big.NewInt(int64(b.internalChainID))) == 0 {
					b.pendingBridgeBatchesI2E = nil
				} else {
					b.pendingBridgeBatchesE2I = nil
				}
			}

			unexecutedBatch.Threshold = bridgeBatch.Batch.Threshold
			unexecutedBatch.NumberOfRegularEvents = bridgeBatch.Batch.NumberOfRegularEvents
			unexecutedBatch.CommitCounter = bridgeBatch.Batch.CommitCounter

			b.unexecutedBatches = append(b.unexecutedBatches, unexecutedBatch)

			b.logger.Info(fmt.Sprintf("Batch (%s, %s, %d, %d -> %d) has been successfully added to the unexecuted list",
				event.ID.String(), bridgeBatch.Batch.CommitCounter.String(),
				len(bridgeBatch.Batch.Messages), sid.Uint64(), did.Uint64()))

			b.lock.Unlock()
		}

		if bridgeBatch.Batch.CommitCounter.Cmp(big.NewInt(1)) == 0 {
			for _, m := range bridgeBatch.Batch.Messages {
				if m.IsRollback {
					if err := b.state.removeBridgeMessageEvent(m.ID, m.SourceChainID, m.DestinationChainID, true, dbTx); err != nil {
						b.logger.Error("could not remove rollback bridge message", "err", err)

						return err
					}
				}
			}
		}

		b.logger.Info(fmt.Sprintf("Commitment of the batch (%s, %s, %d, %d -> %d) has been successfully processed",
			event.ID.String(), bridgeBatch.Batch.CommitCounter.String(),
			len(bridgeBatch.Batch.Messages), sid.Uint64(), did.Uint64()))

	default:
		b.logger.Error("unknown bridge event")

		return errUnknownBridgeEvent
	}

	return nil
}

// AddLog method is responsible for processing bridge events originating from the external (non-Blade)
// chain. An event tracker is responsible for collecting these events.
func (b *bridgeEventManager) AddLog(chainID *big.Int, eventLog *ethgo.Log) error {
	// If the event comes from an external chain that is not managed by the current
	// bridge manager, it should be immediately discarded.
	if b.externalChainID != chainID.Uint64() {
		return nil
	}

	// We process three types of events:
	// 1. bridgeMessageEventSig (`BridgeMsg`)
	//	  - This event is emitted by the `sendBridgeMsg` method of the Gateway smart contract on
	//	 	an external chain. It is triggered when users submit a transaction to transfer tokens
	//  	from the external to the internal chain. The only required action in this case is to
	//		store the event (bridge message) in the local database.
	// 2. bridgeMessageResultEventSig (`BridgeMessageResult`)
	//	  - This event is emitted by the `_executeBridgeMessage` or `_executeRollbackBridgeMessage`
	// 		method of the Gateway smart contract on an external chain. It is triggered when a normal
	//		(ordinary) or rollback message is transferred from the internal to the external chain
	//		and executed there. The processing of this event depends on whether it is a normal or
	//		rollback message, as well as whether the message was successfully executed or not.
	// 3. bridgeBatchProcessedEventSig (`BridgeBatchProcessed`)
	//	  - This event is emitted by the `receiveBatch` method of the Gateway smart contract on an
	// 		external chain. It is triggered after a batch transferring messages from the internal
	// 		to the external chain has been successfully bridged and validated on the external chain.
	// 		Success of executing individual messages from a batch do not affect this event in any
	// 		way. Event is used as a sign to remove the batch from the unexecuted list, indicating
	//		that waiting for the batch to process is no longer needed and that the retry mechanism
	//		does not need to be triggered.
	switch eventLog.Topics[0] {
	case bridgeMessageEventSig:
		event := &contractsapi.BridgeMsgEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge message event", "err", err)

			return err
		}

		if err := b.handleBridgeMessageEvent(event, nil); err != nil {
			return err
		}

	case bridgeMessageResultEventSig:
		event := &contractsapi.BridgeMessageResultEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge message result event", "err", err)

			return err
		}

		if err := b.handleBridgeMessageResultEvent(event, nil); err != nil {
			return err
		}

	case bridgeBatchProcessedEventSig:
		event := &contractsapi.BridgeBatchProcessedEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge batch processed event", "err", err)

			return err
		}

		sid := event.SourceChainID
		did := event.DestinationChainID

		b.logger.Info(
			"Caught Bridge batch processed event",
			"success", event.Success,
			"hash", fmt.Sprintf("%v", event.BatchHash),
			"direction", fmt.Sprintf("%d -> %d", sid.Uint64(), did.Uint64()),
		)

		b.lock.Lock()

		for i := 0; i < len(b.unexecutedBatches); i++ {
			hash, err := b.unexecutedBatches[i].Hash()
			if err != nil {
				b.logger.Error("could not calculate a hash for the bridge batch", "err", err)

				return err
			}

			if hash == types.Hash(event.BatchHash) {
				b.unexecutedBatches = append(b.unexecutedBatches[:i], b.unexecutedBatches[i+1:]...)

				break
			}
		}

		b.lock.Unlock()

		b.logger.Info(fmt.Sprintf("Batch (%v, %v, %d -> %d) has been successfully removed from the unexecuted list",
			event.Success, event.BatchHash, sid.Uint64(), did.Uint64()))

	default:
		b.logger.Error(fmt.Sprintf("unknown bridge event came from the external chain %d", chainID.Uint64()))

		return errUnknownBridgeEvent
	}

	return nil
}

func (b *bridgeEventManager) handleBridgeMessageEvent(event *contractsapi.BridgeMsgEvent, dbTx *bolt.Tx) error {
	id := event.ID
	sid := event.SourceChainID
	did := event.DestinationChainID

	b.logger.Info(
		"Caught Bridge message event",
		"ID", id.String(),
		"direction", fmt.Sprintf("%d -> %d", sid.Uint64(), did.Uint64()),
	)

	if err := b.state.insertBridgeMessageEvent(event, false, dbTx); err != nil {
		b.logger.Error("could not insert bridge message", "err", err)

		return err
	}

	b.logger.Info(fmt.Sprintf("Bridge message %s has been successfully stored to the ordinary bucket", id.String()))

	return nil
}

func (b *bridgeEventManager) handleBridgeMessageResultEvent(event *contractsapi.BridgeMessageResultEvent,
	dbTx *bolt.Tx) error {
	id := event.ID
	sid := event.SourceChainID
	did := event.DestinationChainID

	b.logger.Info(
		"Caught Bridge message result event",
		"ID", id.String(),
		"direction", fmt.Sprintf("%d -> %d", sid.Uint64(), did.Uint64()),
		"rollback", event.IsRollback,
		"status", event.Status,
	)

	switch event.IsRollback {
	case false:
		// If the (switch) event is the result of processing a normal (ordinary, non-rollback)
		// bridge message, there are two possible outcomes. The first possibility is that the
		// message was successfully executed. In this case, it should simply be deleted from the
		// local database, specifically from the bucket associated with normal (ordinary) messages.
		// The second possibility is that the message was not successfully executed. In this case,
		// the message must first be retrieved from the bucket associated with normal (ordinary)
		// messages. Then, the source and destination chain IDs must be swapped, as this message
		// now needs to be executed on the internal chain. Once this is done, the modified message
		// is inserted into the bucket associated with rollback messages. Finally, the message is
		// deleted from the bucket associated with normal (ordinary) messages.
		{
			if event.Status {
				if err := b.state.removeBridgeMessageEvent(id, sid, did, false, dbTx); err != nil {
					b.logger.Error("could not remove ordinary bridge message", "err", err)

					return err
				}

				b.logger.Info(fmt.Sprintf("Bridge message %s has been successfully processed", id.String()))

				return nil
			}

			message, err := b.state.getBridgeMessageEvent(id, sid, did, false, dbTx)
			if err != nil {
				b.logger.Error("could not get ordinary bridge message", "err", err)

				return err
			}

			// This can only happen (be nil) if the message was previously removed from the
			// bucket associated with normal (ordinary) messages. Since deletion from that
			// bucket occurs only after the message has been written to the rollback bucket,
			// no further action is required.
			if message == nil {
				b.logger.Info(fmt.Sprintf("Bridge message %s is already in the rollback bucket", id.String()))

				return nil
			}

			message.SourceChainID = did
			message.DestinationChainID = sid

			if err = b.state.insertBridgeMessageEvent(message, true, dbTx); err != nil {
				b.logger.Error("could not insert bridge message to rollback bucket", "err", err)

				return err
			}

			if err = b.state.removeBridgeMessageEvent(id, sid, did, false, dbTx); err != nil {
				b.logger.Error("could not remove ordinary bridge message", "err", err)

				return err
			}

			b.logger.Info(fmt.Sprintf("Bridge message %s has been successfully moved to the rollback bucket", id.String()))
		}
	case true:
		// If the (switch) event is the result of processing a rollback bridge message, there
		// are two possible outcomes: either the message is successfully executed or it is not.
		// In both cases, no action should be taken. We assume that a rollback message can never
		// fail to execute.
		{
			if event.Status {
				b.logger.Info(fmt.Sprintf("Bridge message %s has been successfully processed", id.String()))

				// The following code is currently commented out, because in the current implementation
				// we remove a message from the rollback bucket once a given rollback message is found
				// in the batch that has been committed to bridge storage.

				// if err := b.state.removeBridgeMessageEvent(id, sid, did, true, nil); err != nil {
				// 	b.logger.Error("could not remove rollback bridge message", "err", err)

				// 	return err
				// }

				return nil
			}

			// This should never happen.

			b.logger.Info(fmt.Sprintf("Bridge message %s has not been successfully processed", id.String()))
		}
	}

	return nil
}

// createBridgeTxRelayer creates a new instance of txrelayer.TxRelayer
// used for sending transactions to the external chain
func createBridgeTxRelayer(rpcEndpoint string, logger hclog.Logger) (txrelayer.TxRelayer, error) {
	if rpcEndpoint == "" || strings.Contains(rpcEndpoint, "0.0.0.0") {
		_, port, err := net.SplitHostPort(rpcEndpoint)
		if err == nil {
			rpcEndpoint = fmt.Sprintf("http://%s:%s", "127.0.0.1", port)
		} else {
			rpcEndpoint = txrelayer.DefaultRPCAddress
		}
	}

	return txrelayer.NewTxRelayer(
		txrelayer.WithIPAddress(rpcEndpoint),
		txrelayer.WithWriter(logger.StandardWriter(&hclog.StandardLoggerOptions{})))
}
