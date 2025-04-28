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
	"strconv"
	"strings"
	"sync"

	"github.com/Ethernal-Tech/blockchain-event-tracker/store"
	"github.com/Ethernal-Tech/blockchain-event-tracker/tracker"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/armon/go-metrics"
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
	bridgeMessageResultEventSig   = new(contractsapi.BridgeMessageResultEvent).Sig()
	newBatchEventSig              = new(contractsapi.NewBatchEvent).Sig()
	newValidatorSetStoredEventSig = new(contractsapi.NewValidatorSetStoredEvent).Sig()
)

const maxNumberOfBatchEvents = 10

// BridgeBatchVoteConsensusData encapsulates sender identifier and its signature.
type BridgeBatchVoteConsensusData struct {
	// Signer of the vote.
	Sender string
	// Signature of the message.
	Signature []byte
}

// BridgeBatchVote represents the payload which is gossiped across the network.
type BridgeBatchVote struct {
	*BridgeBatchVoteConsensusData
	// Hash represents the full hash of the bridge batch. This is the subject of the signing.
	Hash []byte
	// EpochNumber denotes the epoch in which the vote was produced.
	EpochNumber uint64
	// SourceChainID represents the ID of the originating chain.
	SourceChainID uint64
	// DestinationChainID represents the ID of the destination chain.
	DestinationChainID uint64
}

type Runtime interface {
	IsActiveValidator() bool
}

type JSONRPCClient interface {
	GetBlockByNumber(jsonrpc.BlockNumber, bool) (*types.Block, error)
}

type ChainTipProvider interface {
	GetLastProcessedBlock() (uint64, error)
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

var _ BridgeManager = (*dummyBridgeManager)(nil)

// dummyBridgeManager is used when bridge is not enabled
type dummyBridgeManager struct{}

func (d *dummyBridgeManager) Start(runtimeCfg *config.Runtime) error             { return nil }
func (d *dummyBridgeManager) AddLog(chainID *big.Int, eventLog *ethgo.Log) error { return nil }
func (d *dummyBridgeManager) BridgeBatch(blockNumber uint64) ([]*BridgeBatchSigned, error) {
	return nil, nil
}
func (d *dummyBridgeManager) PostBlock(req *oracle.PostBlockRequest) error { return nil }
func (d *dummyBridgeManager) PostEpoch(req *oracle.PostEpochRequest) error {
	return nil
}

func (d *dummyBridgeManager) GetInternalGatewayAddr() types.Address { return [20]byte{} }

// EventSubscriber implementation
func (d *dummyBridgeManager) GetLogFilters() map[types.Address][]types.Hash {
	return make(map[types.Address][]types.Hash)
}
func (d *dummyBridgeManager) ProcessLog(header *types.Header,
	log *ethgo.Log, dbTx *bolt.Tx) error {
	return nil
}
func (d *dummyBridgeManager) Close() {}

// bridgeManagerConfig holds the configuration data of bridge event manager
type bridgeManagerConfig struct {
	bridgeCfg         *config.Bridge
	topic             Topic
	key               *wallet.Key
	maxNumberOfEvents uint64
}

var _ BridgeManager = (*bridgeManager)(nil)

// bridgeManager is a struct that manages the workflow of
// saving and querying bridge message events, and creating, and submitting new batches
type bridgeManager struct {
	logger hclog.Logger
	state  *BridgeManagerStore

	config *bridgeManagerConfig

	// per epoch fields
	lock                    sync.RWMutex
	pendingBridgeBatchesE2I []*PendingBridgeBatch
	pendingBridgeBatchesI2E []*PendingBridgeBatch
	unexecutedBatches       []*PendingBridgeBatch
	retryBatches            map[types.Hash]PendingBridgeBatch
	pendingRetryBatches     map[types.Hash][]*PendingBridgeBatch
	externalClient          JSONRPCClient
	validatorSet            validator.ValidatorSet
	epoch                   uint64
	nextEventIDE2I          uint64
	nextEventIDI2E          uint64
	externalChainID         uint64
	internalChainID         uint64
	blockchain              polychain.Blockchain
	votes                   map[types.Hash]map[string][]byte

	runtime             Runtime
	tracker             *tracker.EventTracker
	externalTipProvider ChainTipProvider
}

// newBridgeManager creates a new instance of bridge event manager
func newBridgeManager(
	logger hclog.Logger,
	state *BridgeManagerStore,
	config *bridgeManagerConfig,
	runtime Runtime,
	externalClient JSONRPCClient,
	externalChainID, internalChainID uint64, blockchain polychain.Blockchain, dbTx *bolt.Tx) *bridgeManager {
	bm := &bridgeManager{
		logger:              logger,
		state:               state,
		config:              config,
		retryBatches:        make(map[types.Hash]PendingBridgeBatch),
		pendingRetryBatches: make(map[types.Hash][]*PendingBridgeBatch),
		externalClient:      externalClient,
		runtime:             runtime,
		externalChainID:     externalChainID,
		internalChainID:     internalChainID,
		blockchain:          blockchain,
		votes:               make(map[types.Hash]map[string][]byte),
	}

	bm.restoreUnexecutedBatches(dbTx)

	return bm
}

// restoreUnexecutedBatches restores unexecuted batches (and indirectly retry batches).
func (b *bridgeManager) restoreUnexecutedBatches(dbTx *bolt.Tx) {
	ids, err := b.state.getUnexecutedBatches(b.externalChainID, dbTx)
	if err != nil {
		b.logger.Error("could not get unexecuted batches", "err", err)

		return
	}

	if len(ids) == 0 {
		return
	}

	provider, err := b.blockchain.GetStateProviderForBlock(b.blockchain.CurrentHeader())
	if err != nil {
		b.logger.Error("could not get state provider", "err", err)

		return
	}

	ss := b.blockchain.GetSystemState(provider)
	for _, ID := range ids {
		batch, err := ss.GetBridgeBatchByNumber(big.NewInt(int64(ID)))
		if err != nil {
			b.logger.Error("could not get bridge batch", "err", err)

			continue
		}

		// Since the batch has already been voted on, the specific epoch doesn't matter.
		b.unexecutedBatches = append(b.unexecutedBatches, &PendingBridgeBatch{
			BridgeMessageBatch: batch.Batch,
		})
	}
}

// Start starts the bridge event manager.
func (b *bridgeManager) Start(runtimeConfig *config.Runtime) error {
	if err := b.initTransport(); err != nil {
		return fmt.Errorf("failed to initialize bridge event transport layer. Error: %w", err)
	}

	tracker, err := b.initTracker(runtimeConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize bridge event tracker. Error: %w", err)
	}

	b.tracker = tracker

	return nil
}

func (b *bridgeManager) GetInternalGatewayAddr() types.Address {
	return b.config.bridgeCfg.InternalGatewayAddr
}

// Close stops the bridge event manager.
func (b *bridgeManager) Close() {
	b.tracker.Close()
}

// initTracker starts a new event tracker (to receive bridge events the from external chain.
func (b *bridgeManager) initTracker(runtimeCfg *config.Runtime) (*tracker.EventTracker, error) {
	store, err := store.NewBoltDBEventTrackerStore(path.Join(runtimeCfg.StateDataDir,
		fmt.Sprintf("/bridge-%d.db", b.externalChainID)))
	if err != nil {
		return nil, err
	}

	b.externalTipProvider = store

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
					bridgeMessageResultEventSig},
			},
		},
		store, b.config.bridgeCfg.EventTrackerStartBlocks[b.config.bridgeCfg.ExternalGatewayAddr],
	)

	if err != nil {
		return nil, err
	}

	return eventTracker, eventTracker.Start()
}

// initTransport subscribes to bridge topics to receive votes for batches.
func (b *bridgeManager) initTransport() error {
	return b.config.topic.Subscribe(func(obj interface{}, _ peer.ID) {
		if !b.runtime.IsActiveValidator() {
			// Don't save votes if not a validator.
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

// saveVotes checks the correctness of the vote, and if everything is fine, saves it.
func (b *bridgeManager) saveVote(vote *BridgeBatchVote) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Check if the validator set for the current epoch is known. If not, we cannot process the
	// newly arrived vote.
	if b.validatorSet == nil {
		return nil
	}

	// We are only interested in votes from the epoch we are currently in.
	if vote.EpochNumber != b.epoch {
		return nil
	}

	// If the vote arrives for the bridge path (internal chain <-> external chain) that is not
	// managed by this bridge manager, it should be immediately discarded (it will be processed
	// by the appropriate manager).
	if b.externalChainID != vote.SourceChainID &&
		b.externalChainID != vote.DestinationChainID {
		return nil
	}

	// Check if the signature is valid and if the signer is a validator in the current epoch.
	if err := b.verifyVoteSignature(b.validatorSet, vote); err != nil {
		return fmt.Errorf("error verifying vote signature: %w", err)
	}

	if b.votes[types.Hash(vote.Hash)] == nil {
		b.votes[types.Hash(vote.Hash)] = make(map[string][]byte)
	}

	b.votes[types.Hash(vote.Hash)][vote.Sender] = vote.Signature

	b.logger.Info(
		"New vote for bridge batch saved",
		"batch hash", fmt.Sprintf("0x%v", hex.EncodeToString(vote.Hash)),
		"sender", vote.Sender,
		"total number of votes for the batch", len(b.votes[types.Hash(vote.Hash)]),
	)

	return nil
}

// verifyVoteSignature verifies signature of the message against the public key of the signer and
// checks if the signer is a validator.
func (b *bridgeManager) verifyVoteSignature(
	valSet validator.ValidatorSet,
	vote *BridgeBatchVote) error {
	signerAddr := types.StringToAddress(vote.Sender)
	validator := valSet.Accounts().GetValidatorMetadata(signerAddr)

	if validator == nil {
		return fmt.Errorf("unable to resolve validator %s", signerAddr)
	}

	unmarshaledSignature, err := bls.UnmarshalSignature(vote.Signature)
	if err != nil {
		return fmt.Errorf("failed to unmarshal signature from signer %s, %w",
			signerAddr.String(),
			err)
	}

	if !unmarshaledSignature.Verify(validator.BlsKey, vote.Hash, signer.DomainBridge) {
		return fmt.Errorf("incorrect signature from %s", signerAddr)
	}

	return nil
}

// BridgeBatch returns a list of batches to be submitted.
func (b *bridgeManager) BridgeBatch(blockNumber uint64) ([]*BridgeBatchSigned, error) {
	// BBB - just for "easy-to-search" purposes
	getLargestPendingBatchFn := func(
		pendingBatches []*PendingBridgeBatch,
		sourceChainId uint64) (*BridgeBatchSigned, error) {
		var (
			largestBridgeBatch *BridgeBatchSigned
			err                error
		)

		// We start from the end, since the last pending batch is the most relevant one.
		for i := len(pendingBatches) - 1; i >= 0; i-- {
			if pendingBatches[i].SourceChainID.Uint64() == sourceChainId {
				var sig polytypes.Signature

				sig, err = b.getAggSignatureForBridgeBatch(blockNumber, pendingBatches[i])
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
					AggSignature:       sig,
					InternalChainID:    b.internalChainID,
				}

				break
			}
		}

		return largestBridgeBatch, err
	}

	largestI2EBatch, err := getLargestPendingBatchFn(b.pendingBridgeBatchesI2E, b.internalChainID)
	if err != nil {
		return nil, fmt.Errorf("could not get largest I2E pending batch. Error: %w", err)
	}

	largestE2IBatch, err := getLargestPendingBatchFn(b.pendingBridgeBatchesE2I, b.externalChainID)
	if err != nil {
		return nil, fmt.Errorf("could not get largest E2I pending batch. Error: %w", err)
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

// getAggSignatureForBridgeBatch checks if the given pending batch has a quorum and, if it does,
// aggregates and returns the signature.
func (b *bridgeManager) getAggSignatureForBridgeBatch(
	blockNumber uint64,
	pendingBridgeBatch *PendingBridgeBatch) (polytypes.Signature, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	validatorAddrToIndex := make(map[string]int, b.validatorSet.Len())
	validatorsMetadata := b.validatorSet.Accounts()

	for i, validator := range validatorsMetadata {
		validatorAddrToIndex[validator.Address.String()] = i
	}

	bridgeBatchHash, err := pendingBridgeBatch.Hash()
	if err != nil {
		return polytypes.Signature{}, err
	}

	votes := []*BridgeBatchVoteConsensusData{}

	for sender, signature := range b.votes[bridgeBatchHash] {
		votes = append(votes, &BridgeBatchVoteConsensusData{
			Sender:    sender,
			Signature: signature,
		})
	}

	var (
		signatures = make(bls.Signatures, 0, len(votes))
		bmap       = bitmap.Bitmap{}
		signers    = make(map[types.Address]struct{}, 0)
	)

	for _, vote := range votes {
		index, exists := validatorAddrToIndex[vote.Sender]
		if !exists {
			// Don't count this vote, because it does not belong to validator.
			continue
		}

		signature, err := bls.UnmarshalSignature(vote.Signature)
		if err != nil {
			return polytypes.Signature{}, err
		}

		bmap.Set(uint64(index))

		signatures = append(signatures, signature)
		signers[types.StringToAddress(vote.Sender)] = struct{}{}
	}

	if !b.validatorSet.HasQuorum(blockNumber, signers) {
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

// PostEpoch notifies the bridge event manager that an epoch has changed, so that it can discard
// any previous epoch bridge batch, and build a new one (since validator set changed)
func (b *bridgeManager) PostEpoch(req *oracle.PostEpochRequest) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.pendingBridgeBatchesE2I = nil
	b.pendingBridgeBatchesI2E = nil
	b.validatorSet = req.ValidatorSet
	b.epoch = req.NewEpochID
	b.votes = make(map[types.Hash]map[string][]byte)

	return nil
}

// PostBlock creates I2E, E2I and retry batches.
func (b *bridgeManager) PostBlock(req *oracle.PostBlockRequest) error {
	var sysState systemstate.SystemState

	if req.FullBlock.Block.Header.Number > 1 {
		provider, err := b.blockchain.GetStateProviderForBlock(req.FullBlock.Block.Header)
		if err != nil {
			return err
		}

		sysState = b.blockchain.GetSystemState(provider)

		b.nextEventIDI2E, err = sysState.GetNextCommittedIndex(b.externalChainID, systemstate.I2E)
		if err != nil {
			return err
		}

		b.nextEventIDE2I, err = sysState.GetNextCommittedIndex(b.externalChainID, systemstate.E2I)
		if err != nil {
			return err
		}

		if err := b.buildI2EBridgeBatch(sysState, req.DBTx); err != nil {
			b.logger.Error("could not build an internal chain originated batch on PostBlock",
				"err", err)
		}

		if err := b.buildE2IBridgeBatch(sysState, req.DBTx); err != nil {
			b.logger.Error("could not build an external chain originated batch on PostBlock",
				"err", err)
		}

		b.handleRetry(sysState)
	}

	return nil
}

// buildE2IBridgeBatch builds, signs, and multicasts the E2I batch.
func (b *bridgeManager) buildE2IBridgeBatch(
	sysState systemstate.SystemState,
	dbTx *bolt.Tx) error {
	return b.buildBridgeBatch(b.externalChainID, b.internalChainID, b.nextEventIDE2I, sysState, dbTx)
}

// buildI2EBridgeBatch builds, signs, and multicasts the I2E batch.
func (b *bridgeManager) buildI2EBridgeBatch(
	sysState systemstate.SystemState,
	dbTx *bolt.Tx) error {
	return b.buildBridgeBatch(b.internalChainID, b.externalChainID, b.nextEventIDI2E, sysState, dbTx)
}

// buildBridgeBatch builds, signs, and multicasts the batch.
func (b *bridgeManager) buildBridgeBatch(
	sourceChainID,
	destinationChainID uint64,
	nextBridgeEventIDIndex uint64,
	sysState systemstate.SystemState,
	dbTx *bolt.Tx) error {
	if !b.runtime.IsActiveValidator() {
		return nil
	}

	epoch := b.epoch
	messages, numOfOrdinaryMsgs, err := b.state.getBridgeMessages(
		nextBridgeEventIDIndex,
		b.config.maxNumberOfEvents,
		sourceChainID,
		destinationChainID,
		sysState,
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
			Threshold:          big.NewInt(0),
			CommitCounter:      big.NewInt(0),
		},
		Epoch: epoch,
	}

	baseHash, err := pendingBatch.Hash()
	if err != nil {
		return fmt.Errorf("could not generate a hash for bridge batch. Error: %w", err)
	}

	pendingBatch.Threshold = new(big.Int).SetUint64(uint64((math.Ceil(float64(blockNumber)/10) * 10)) +
		b.config.bridgeCfg.BridgeBatchThreshold)
	pendingBatch.CommitCounter = big.NewInt(1)

	fullHash, err := pendingBatch.Hash()
	if err != nil {
		return fmt.Errorf("could not generate a hash for bridge batch. Error: %w", err)
	}

	hashBytes := fullHash.Bytes()

	signature, err := b.config.key.SignWithDomain(hashBytes, signer.DomainBridge)
	if err != nil {
		return fmt.Errorf("could not create a signature for bridge batch. Error: %w", err)
	}

	sig := &BridgeBatchVoteConsensusData{
		Sender:    b.config.key.String(),
		Signature: signature,
	}

	b.lock.Lock()
	if b.votes[fullHash] == nil {
		b.votes[fullHash] = make(map[string][]byte)
	}

	b.votes[fullHash][sig.Sender] = sig.Signature
	b.lock.Unlock()

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
		lastID = pendingBatch.Messages[int(numOfOrdinaryMsgs)-1].ID
	}

	b.logger.Info(
		"New (pending) Bridge batch created and multicasted",
		"direction", fmt.Sprintf("%d -> %d", sid.Uint64(), did.Uint64()),
		"total number of messages", len(pendingBatch.Messages),
		"number of ordinary", fmt.Sprintf("%d (%s-%s)", numOfOrdinaryMsgs, firstID.String(), lastID.String()),
		"number of rollback", len(pendingBatch.Messages)-int(numOfOrdinaryMsgs),
		"threshold", pendingBatch.Threshold,
		"base hash", baseHash.String(),
		"full hash", fullHash.String(),
	)

	if sourceChainID == b.internalChainID {
		b.pendingBridgeBatchesI2E = append(b.pendingBridgeBatchesI2E, pendingBatch)
	} else {
		b.pendingBridgeBatchesE2I = append(b.pendingBridgeBatchesE2I, pendingBatch)
	}

	return nil
}

// handleRetry handles the complete logic related to checking whether a batch is ready for retry,
// as well as building and broadcasting retry candidates.
func (b *bridgeManager) handleRetry(
	sysState systemstate.SystemState) {
	blockNumber, err := b.externalTipProvider.GetLastProcessedBlock()
	if err != nil {
		// Log the error, but won't return because it might be just a temporary problem.
		b.logger.Error("could not poll the block from the external chain", "err", err)

		return
	}

	b.lock.Lock()

	// If an error occurs inside the loop, we simply log it and proceed to the next iteration.
	// We do not return immediately upon encountering an error because we want to process all
	// batches from the unexecuted list. The iterator `i` does not increment when a retry batch
	// is successfully created, as the unexecuted list shrinks by one, making the next batch (i+1)
	// the current (i).
	for i := 0; i < len(b.unexecutedBatches); {
		// For each batch in the list, we check whether the number of the last processed block
		// on the external chain is greater than the batch threshold. If so, batch is ready for
		// retry.
		if blockNumber > b.unexecutedBatches[i].Threshold.Uint64() {
			retryBatch := *b.unexecutedBatches[i]
			retryBatch.Threshold = big.NewInt(0)
			retryBatch.CommitCounter = big.NewInt(0)

			hash, err := retryBatch.Hash()
			if err != nil {
				b.logger.Error("could not generate a hash for retry bridge batch", "err", err)

				i++

				continue
			}

			numOfTries, err := sysState.GetBatchCommitCounter(hash)
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
			// Storing the retry template for the given batch.
			b.retryBatches[hash] = retryBatch
			// Storing the list that will contain retry candidates for the given batch.
			b.pendingRetryBatches[hash] = []*PendingBridgeBatch{}

			b.logger.Info(
				fmt.Sprintf("Retry mechanism has been successfully started for the batch (%s, %d -> %d)",
					hash.String(),
					retryBatch.SourceChainID.Uint64(),
					retryBatch.DestinationChainID.Uint64()))
		} else {
			i++
		}
	}

	b.lock.Unlock()

	if !b.runtime.IsActiveValidator() {
		return
	}

	// Creating a new retry candidate for each batch that is ready for the retry.
	for hash := range b.retryBatches {
		err = b.buildRetryBridgeBatch(hash, blockNumber)
		if err != nil {
			b.logger.Error("could not create retry bridge batch", "err", err)
		}
	}
}

// buildRetryBridgeBatch builds, signs, and multicasts a retry version of the batch.
func (b *bridgeManager) buildRetryBridgeBatch(
	baseHash types.Hash,
	blockNumber uint64) error {
	//
	// Bulding a retry batch is actually based just on taking a retry template for the given
	// batch and calculating a new threshold.
	//
	// Taking a retry template.
	rb := b.retryBatches[baseHash]
	pendingBatch := PendingBridgeBatch{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           rb.Messages,
			SourceChainID:      rb.SourceChainID,
			DestinationChainID: rb.DestinationChainID,
			Threshold: new(big.Int).SetUint64(
				uint64((math.Ceil(float64(blockNumber)/10) * 10)) +
					b.config.bridgeCfg.BridgeBatchThreshold),
			CommitCounter: rb.CommitCounter,
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

	b.lock.Lock()
	if b.votes[hash] == nil {
		b.votes[hash] = make(map[string][]byte)
	}

	b.votes[hash][b.config.key.String()] = signature
	b.lock.Unlock()

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

	pendingRetryBatches := b.pendingRetryBatches[baseHash]
	b.pendingRetryBatches[baseHash] = append(pendingRetryBatches, &pendingBatch)

	numOfOrdinaryMsgs := 0

	for _, message := range pendingBatch.Messages {
		if !message.IsRollback {
			numOfOrdinaryMsgs++
		}
	}

	firstID := big.NewInt(0)
	lastID := big.NewInt(0)

	if numOfOrdinaryMsgs > 0 {
		firstID = pendingBatch.Messages[0].ID
		lastID = pendingBatch.Messages[numOfOrdinaryMsgs-1].ID
	}

	b.logger.Info(
		"New (retry) Bridge batch created and multicasted",
		"direction", fmt.Sprintf("%d -> %d", rb.SourceChainID.Uint64(), rb.DestinationChainID.Uint64()),
		"total number of messages", len(pendingBatch.Messages),
		"number of ordinary", fmt.Sprintf("%d (%s-%s)", numOfOrdinaryMsgs, firstID.String(), lastID.String()),
		"number of rollback", len(pendingBatch.Messages)-numOfOrdinaryMsgs,
		"threshold", pendingBatch.Threshold,
		"base hash", baseHash.String(),
		"full hash", hash.String(),
	)

	return nil
}

// multicast publishes (broadcasts) a given message to the rest of the network.
func (b *bridgeManager) multicast(msg any) {
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

// GetLogFilters returns a map of log filters for getting desired events from the internal chain.
func (b *bridgeManager) GetLogFilters() map[types.Address][]types.Hash {
	return map[types.Address][]types.Hash{
		b.config.bridgeCfg.InternalGatewayAddr: {
			types.Hash(bridgeMessageEventSig),
			types.Hash(bridgeMessageResultEventSig),
		},
		contracts.BridgeStorageContract: {
			types.Hash(newBatchEventSig),
		},
	}
}

// ProcessLog method is responsible for processing bridge events originating from the internal
// (Blade) chain. An event provider is responsible for collecting these events.
func (b *bridgeManager) ProcessLog(
	header *types.Header,
	eventLog *ethgo.Log,
	dbTx *bolt.Tx) error {
	isEventMine := func(chainID *big.Int) bool {
		// If the event is related to an external (non-Blade) chain that is not managed by the
		// current bridge manager, it should be immediately discarded.
		if b.externalChainID != chainID.Uint64() {
			return false
		}

		return true
	}

	// We process three types of events:
	// 1. bridgeMessageEventSig (`BridgeMsg`)
	//	  - This event is emitted by the `sendBridgeMsg` method of the Gateway smart contract on
	//	 	an internal chain. It is triggered when user submit a transaction to transfer tokens
	//  	from the internal to the external chain.
	// 2. bridgeMessageResultEventSig (`BridgeMessageResult`)
	//	  - This event is emitted by the `_executeBridgeMessage`/`_executeRollbackBridgeMessage`
	// 		method of the Gateway smart contract on an internal chain. It is triggered when the
	//		ordinary or rollback message is transferred from the external to the internal chain
	//		and executed there.
	// 3. newBatchEventSig (`NewBatch`)
	//	  - This event is emitted by the `commitBatch` method of the BridgeStorage smart contract
	// 		on an internal chain. It is triggered after the quorum number of signatures for the
	//		given batch is collected and the block proposer commits it on the sprint block.
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

		b.lock.Lock()
		defer b.lock.Unlock()

		if err := b.handleBridgeMessageEvent(header, event, dbTx); err != nil {
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

		b.lock.Lock()
		defer b.lock.Unlock()

		if err := b.handleBridgeMessageResultEvent(header, event, dbTx); err != nil {
			return err
		}

		// Unlike handling in AddLog, in this case there is no check related to the unexecuted
		// list. The reason lies in the fact that E2I batches are executed immediately, so they
		// are not even written to the unexecuted list.

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

		ss := b.blockchain.GetSystemState(provider)

		bridgeBatch, err := ss.GetBridgeBatchByNumber(event.ID)
		if err != nil {
			return err
		}

		if !isEventMine(bridgeBatch.Batch.SourceChainID) &&
			!isEventMine(bridgeBatch.Batch.DestinationChainID) {
			return nil
		}

		b.lock.Lock()
		defer b.lock.Unlock()

		b.logger.Info(
			"Caught New batch event",
			"ID", event.ID.String(),
		)

		sid := bridgeBatch.Batch.SourceChainID
		did := bridgeBatch.Batch.DestinationChainID

		baseUnexecutedBatch := &PendingBridgeBatch{
			BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
				Messages:           bridgeBatch.Batch.Messages,
				SourceChainID:      bridgeBatch.Batch.SourceChainID,
				DestinationChainID: bridgeBatch.Batch.DestinationChainID,
				Threshold:          big.NewInt(0),
				CommitCounter:      big.NewInt(0),
			},
		}

		baseHash, err := baseUnexecutedBatch.Hash()
		if err != nil {
			b.logger.Error("could not calculate a hash for the bridge batch", "err", err)

			return err
		}

		fullUnexecutedBatch := &PendingBridgeBatch{
			BridgeMessageBatch: bridgeBatch.Batch,
			Epoch:              b.epoch,
		}

		fullHash, err := fullUnexecutedBatch.Hash()
		if err != nil {
			b.logger.Error("could not calculate a hash for the bridge batch", "err", err)

			return err
		}

		// First, we need to update the state of the bridge manager, represented through the
		// structures such as the unexecuted list, the retry batch map, etc., to reflect the
		// new state resulting from the commit of the latest batch.

		b.updateStateOnBatchCommit(event.ID, baseHash, fullHash, fullUnexecutedBatch, dbTx)

		// Next step is to delete committed rollback messages from the bolt bucket related
		// to rollback messages. (this should be part of the handleBridgeMessageCommitment)

		if bridgeBatch.Batch.CommitCounter.Cmp(big.NewInt(1)) == 0 {
			for _, m := range bridgeBatch.Batch.Messages {
				if m.IsRollback {
					if err := b.state.removeBridgeMessageEvent(
						m.ID,
						m.SourceChainID,
						m.DestinationChainID,
						true,
						dbTx); err != nil {
						b.logger.Error("could not remove rollback bridge message", "err", err)

						return err
					}
				}
			}
		}

		// Finally, for each committed ordinary message, we need to check if it's known to us, as
		// well as if it has already been executed (this disorder can happen, for example, during
		// synchronization). For an ordinary message for which the previous two conditions are met,
		// we proceed to the message finalization phase (see finalizeOrdinaryBridgeMessage for more
		// information).

		for _, m := range bridgeBatch.Batch.Messages {
			if err := b.handleBridgeMessageCommitment(m, dbTx); err != nil {
				b.logger.Error("could not handle bridge message commitment", "err", err)
			}
		}

		b.logger.Info(
			fmt.Sprintf("Commitment of the batch (%s, %s, %s, %s, %d -> %d)"+
				"has been successfully processed",
				event.ID.String(),
				baseHash.String(),
				bridgeBatch.Batch.CommitCounter.String(),
				fullHash.String(),
				sid.Uint64(),
				did.Uint64()))

		if sid.Uint64() == b.internalChainID {
			err = b.state.insertUnexecutedBatch(b.externalChainID, baseHash, event.ID, dbTx)
			if err != nil {
				b.logger.Error("could not insert unexecuted batch", "err", err)
			}
		}

	default:
		b.logger.Error("unknown bridge event")

		return errUnknownBridgeEvent
	}

	return nil
}

// updateStateOnBatchCommit updates the bridge manager's state structures to reflect the latest
// state resulting from the commit of the new batch.
func (b *bridgeManager) updateStateOnBatchCommit(
	eventID *big.Int,
	baseHash types.Hash,
	fullHash types.Hash,
	batch *PendingBridgeBatch,
	dbTx *bolt.Tx) {
	//
	// Updating the state is a bit tricky, but the behavior is as follows. First, we check
	// whether it's I2E batch or not. If it's not, we simply restart pendingBridgeBatchesE2I
	// because a new E2I batch has been found and committed. If it is an I2E batch, we check
	// whether it's a retry batch or not. For E2I, this check doesn't exist because batches
	// are executed immediately, therefore they are not written to the unexecuted list, thus
	// cannot be retries. Checking whether it's a retry batch is done by looking at whether
	// the batch's commit counter is greater than 1. If it's a retry batch, we delete it from
	// the map of batches that are ready for retry (retryBatches), because a new version of
	// the batch has just been committed. Additionally, we also delete the retry candidates
	// for the given batch (pendingRetryBatches). The previous two deletions are implemented
	// only if the node had previously initiated a retry for the given batch (during syncing,
	// it can happen that the retry mechanism for a batch hasn't been initiated at all, even
	// though it was initiated on synchronized nodes). If it's not an I2E retry batch, we
	// simply restart (set to nil) pendingBridgebatchesI2E because a new (regular) I2E batch
	// has been found and committed. Since the above has been done, the I2E batch should be
	// inserted into the unexecuted list. However, it is inserted into this list only if at
	// least one of its messages hasn't been executed. If all messages have been executed,
	// then the batch is also executed, so adding it to the given list would be incorrect.
	//
	if batch.SourceChainID.Uint64() == b.internalChainID &&
		batch.DestinationChainID.Uint64() == b.externalChainID {
		if batch.CommitCounter.Cmp(big.NewInt(1)) > 0 {
			if _, ok := b.retryBatches[baseHash]; ok {
				delete(b.retryBatches, baseHash)
				delete(b.pendingRetryBatches, baseHash)

				b.logger.Info(
					fmt.Sprintf(
						"Batch (%s, %s, %s, %s, %d -> %d)"+
							"has been successfully removed from the retry map",
						eventID.String(),
						baseHash.String(),
						batch.CommitCounter.String(),
						fullHash.String(),
						batch.SourceChainID.Uint64(),
						batch.DestinationChainID.Uint64()))
			}
		} else {
			b.pendingBridgeBatchesI2E = nil
		}

		alreadyExecuted := true

		for _, msg := range batch.Messages {
			if !b.state.isBridgeMessageExecuted(msg, dbTx) {
				alreadyExecuted = false

				break
			}
		}

		if !alreadyExecuted {
			b.unexecutedBatches = append(b.unexecutedBatches, batch)

			b.logger.Info(
				fmt.Sprintf("Batch (%s, %s, %s, %s, %d -> %d)"+
					"has been successfully added to the unexecuted list",
					eventID.String(),
					baseHash.String(),
					batch.CommitCounter.String(),
					fullHash.String(),
					batch.SourceChainID.Uint64(),
					batch.DestinationChainID.Uint64()))
		}
	} else {
		b.pendingBridgeBatchesE2I = nil
	}
}

// handleBridgeMessageCommitment handles the commitment of the new message on the BridgeStorage.
func (b *bridgeManager) handleBridgeMessageCommitment(
	message *contractsapi.BridgeMessage,
	dbTx *bolt.Tx) error {
	if message.IsRollback {
		return nil
	}

	if !b.state.isBridgeMessageKnown(message, dbTx) {
		return nil
	}

	if !b.state.isBridgeMessageExecuted(message, dbTx) {
		return nil
	}

	result, err := b.state.getBridgeMessageResult(message, dbTx)
	if err != nil {
		return fmt.Errorf("could not get bridge message result, err: %w", err)
	}

	if err := b.finalizeOrdinaryBridgeMessage(message, result.Status, dbTx); err != nil {
		return fmt.Errorf("could not finalize bridge message, err: %w", err)
	}

	return nil
}

// AddLog is responsible for processing bridge events originating from the external (non-Blade)
// chain. An event tracker is responsible for collecting these events.
func (b *bridgeManager) AddLog(
	chainID *big.Int,
	eventLog *ethgo.Log) error {
	// If the event comes from an external chain that is not managed by this bridge manager, it
	// should be immediately discarded.
	if b.externalChainID != chainID.Uint64() {
		return nil
	}

	// It's important to first start (open) the bolt DB write transaction and only then lock the
	// mutex. Otherwise, there is a possibility of a deadlock occurring.
	dbTx, err := b.state.beginDBTransaction(true)
	if err != nil {
		return err
	}
	defer dbTx.Commit() //nolint:errcheck

	b.lock.Lock()
	defer b.lock.Unlock()

	currentBlock := b.blockchain.CurrentHeader()

	// We process two types of events:
	// 1. bridgeMessageEventSig (`BridgeMsg`)
	//	  - This event is emitted by the `sendBridgeMsg` method of the Gateway smart contract on
	//	 	an external chain. It is triggered when user submit a transaction to transfer tokens
	//  	from the external to the internal chain.
	// 2. bridgeMessageResultEventSig (`BridgeMessageResult`)
	//	  - This event is emitted by the `_executeBridgeMessage`/`_executeRollbackBridgeMessage`
	// 		method of the Gateway smart contract on an internal chain. It is triggered when the
	//		ordinary or rollback message is transferred from the external to the internal chain
	//		and executed there.
	switch eventLog.Topics[0] {
	case bridgeMessageEventSig:
		event := &contractsapi.BridgeMsgEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge message event", "err", err)

			return err
		}

		if err := b.handleBridgeMessageEvent(currentBlock, event, dbTx); err != nil {
			return err
		}

	case bridgeMessageResultEventSig:
		event := &contractsapi.BridgeMessageResultEvent{}

		doesMatch, err := event.ParseLog(eventLog)
		if !doesMatch || err != nil {
			b.logger.Error("could not decode bridge message result event", "err", err)

			return err
		}

		if err := b.handleBridgeMessageResultEvent(currentBlock, event, dbTx); err != nil {
			return err
		}

		// Since a result has arrived for some newly executed message, we need to go through all
		// unexecuted batches and check if all messages have been executed for any batch. If so,
		// that batch is deleted from the list of unexecuted batches. In this way, a given batch
		// won't be considered for retry anymore.

	ub_loop:
		for i := 0; i < len(b.unexecutedBatches); {
			for _, message := range b.unexecutedBatches[i].Messages {
				if !b.state.isBridgeMessageExecuted(message, dbTx) {
					i++

					continue ub_loop
				}
			}

			hash, err := b.unexecutedBatches[i].Hash()
			if err != nil {
				b.logger.Error("could not calculate a hash for the bridge batch", "err", err)

				i++

				continue
			}

			b.unexecutedBatches[i].Threshold = big.NewInt(0)
			b.unexecutedBatches[i].CommitCounter = big.NewInt(0)

			baseHash, err := b.unexecutedBatches[i].Hash()
			if err != nil {
				b.logger.Error("could not calculate a base hash for the bridge batch", "err", err)

				i++

				continue
			}

			b.logger.Info(
				fmt.Sprintf("Batch (0x%s, %d -> %d)"+
					"has been successfully removed from the unexecuted list",
					hex.EncodeToString(hash.Bytes()),
					b.unexecutedBatches[i].SourceChainID.Uint64(),
					b.unexecutedBatches[i].DestinationChainID.Uint64()))

			b.unexecutedBatches = append(b.unexecutedBatches[:i], b.unexecutedBatches[i+1:]...)

			err = b.state.removeUnexecutedBatch(b.externalChainID, baseHash, dbTx)
			if err != nil {
				b.logger.Error("could not insert unexecuted batch", "err", err)
			}
		}

	default:
		b.logger.Error(fmt.Sprintf("unknown bridge event came from the external chain %d",
			chainID.Uint64()))

		return errUnknownBridgeEvent
	}

	return nil
}

// handleBridgeMessageEvent handles the BridgeMsg event emitted by the Gateway SC on the internal
// (Blade) and all external chains.
func (b *bridgeManager) handleBridgeMessageEvent(
	header *types.Header,
	event *contractsapi.BridgeMsgEvent,
	dbTx *bolt.Tx) error {
	//
	// Handling the arrival of a new message is basically only focused on writing the message to
	// the bucket related to ordinary messages. However, in certain situations (e.g. syncing) it
	// may happen that the given message has already been previously committed or even executed.
	// If the previous two conditions are met, we proceed to the message finalization phase (see
	// finalizeOrdinaryBridgeMessage for more information).
	//
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

	metrics.IncrCounterWithLabels([]string{"num_of_pending_bridge_messages"}, 1,
		[]metrics.Label{{
			Name:  "ID",
			Value: strconv.Itoa(int(b.externalChainID)),
		}})

	msg := &contractsapi.BridgeMessage{
		ID:                 id,
		SourceChainID:      sid,
		DestinationChainID: did,
		IsRollback:         false,
	}

	committed, err := b.isBridgeMessageCommitted(header, msg)
	if err != nil {
		return err
	}

	if !committed {
		return nil
	}

	if !b.state.isBridgeMessageExecuted(msg, dbTx) {
		return nil
	}

	result, err := b.state.getBridgeMessageResult(msg, dbTx)
	if err != nil {
		b.logger.Error("could not get bridge message result", "err", err)

		return err
	}

	return b.finalizeOrdinaryBridgeMessage(msg, result.Status, dbTx)
}

// handleBridgeMessageResultEvent handles the BridgeMessageResult event emitted by the Gateway SC
// on the internal (Blade) and all external chains.
func (b *bridgeManager) handleBridgeMessageResultEvent(
	header *types.Header,
	event *contractsapi.BridgeMessageResultEvent,
	dbTx *bolt.Tx) error {
	//
	// Handling the execution result of a message depends on whether it was an ordinary message
	// or not (rollback message), as well as whether the message was successfully executed or
	// not. If it's a rollback message, regardless of whether it was successfully executed or
	// not, only logging is performed (note that we consider that a rollback message will never
	// fail to execute). If it's a result for an ordinary message, we check whether the given
	// ordinary message is known (we previously received a BridgeMsg event from the Gateway) and
	// whether it has been committed (through a batch on BridgeStorage). If either of these two
	// conditions is not met, the process stops there. Otherwise, we proceed to the finalization
	// phase (see finalizeOrdinaryBridgeMessage for more information).
	//
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

	if err := b.state.insertBridgeMessageResultEvent(event, dbTx); err != nil {
		b.logger.Error("could not insert bridge message result", "err", err)

		return err
	}

	switch event.IsRollback {
	case false:
		msg := &contractsapi.BridgeMessage{
			ID:                 id,
			SourceChainID:      sid,
			DestinationChainID: did,
			IsRollback:         false,
		}

		if !b.state.isBridgeMessageKnown(msg, dbTx) {
			return nil
		}

		committed, err := b.isBridgeMessageCommitted(header, msg)
		if err != nil {
			return err
		}

		if !committed {
			return nil
		}

		return b.finalizeOrdinaryBridgeMessage(msg, event.Status, dbTx)
	case true:
		if event.Status {
			b.logger.Info(fmt.Sprintf("Rollback bridge message %s has been successfully processed",
				id.String()))

			metrics.IncrCounterWithLabels([]string{"num_of_successful_bridge_messages"}, 1,
				[]metrics.Label{{
					Name:  "ID",
					Value: strconv.Itoa(int(b.externalChainID)),
				}})

			metrics.IncrCounterWithLabels([]string{"num_of_pending_bridge_messages"}, -1,
				[]metrics.Label{{
					Name:  "ID",
					Value: strconv.Itoa(int(b.externalChainID)),
				}})

			return nil
		}

		// This should never happen.

		b.logger.Info(fmt.Sprintf("Rollback bridge message %s has not been successfully processed",
			id.String()))
	}

	return nil
}

// finalizeOrdinaryBridgeMessage represents the final phase of processing an ordinary message.
func (b *bridgeManager) finalizeOrdinaryBridgeMessage(
	msg *contractsapi.BridgeMessage,
	successful bool,
	dbTx *bolt.Tx) error {
	//
	// The way the message is processed depends on whether it was successfully executed or not.
	// If it was successfully executed, it is simply deleted from the bucket related to ordinary
	// messages. Otherwise, we check if its rollback version is already committed. If not, the
	// message is deleted from the bucket related to ordinary messages and written to the bucket
	// related to rollback messages. In other words, it is transferred from the ordinary to the
	// rollback bucket. Note: It's possible that the rollback version of the message has already
	// been committed. However, this is not a concern, as we handle this scenario when creating
	// a batch (rollback messages that are committed and still present in the DB are deleted at
	// that time).
	//
	id := msg.ID
	sid := msg.SourceChainID
	did := msg.DestinationChainID

	if successful {
		if err := b.state.removeBridgeMessageEvent(id, sid, did, false, dbTx); err != nil {
			b.logger.Error("could not remove ordinary bridge message", "err", err)

			return err
		}

		metrics.IncrCounterWithLabels([]string{"num_of_successful_bridge_messages"}, 1,
			[]metrics.Label{{
				Name:  "ID",
				Value: strconv.Itoa(int(b.externalChainID)),
			}})

		metrics.IncrCounterWithLabels([]string{"num_of_pending_bridge_messages"}, -1,
			[]metrics.Label{{
				Name:  "ID",
				Value: strconv.Itoa(int(b.externalChainID)),
			}})

		b.logger.Info(fmt.Sprintf("Bridge message %s has been successfully processed", id.String()))

		return nil
	}

	metrics.IncrCounterWithLabels([]string{"num_of_unsuccessful_bridge_messages"}, 1,
		[]metrics.Label{{
			Name:  "ID",
			Value: strconv.Itoa(int(b.externalChainID)),
		}})

	// In this case, we don't decrease the number of pending messages (in the context of metrics),
	// because even though the ordinary message is executed (and the number of pending messages
	// should decreases by one), a new rollback message is created, thus the number of pending
	// messages remains the same.

	rollbackMsg, err := b.state.getBridgeMessageEvent(id, sid, did, false, dbTx)
	if err != nil {
		b.logger.Error("could not get ordinary bridge message", "err", err)

		return err
	}

	rollbackMsg.SourceChainID = did
	rollbackMsg.DestinationChainID = sid

	if err := b.state.insertBridgeMessageEvent(rollbackMsg, true, dbTx); err != nil {
		b.logger.Error("could not insert bridge message to rollback bucket", "err", err)

		return err
	}

	if err := b.state.removeBridgeMessageEvent(id, sid, did, false, dbTx); err != nil {
		b.logger.Error("could not remove ordinary bridge message", "err", err)

		return err
	}

	b.logger.Info(fmt.Sprintf("Bridge message %s has been successfully moved to the rollback bucket",
		id.String()))

	return nil
}

// isBridgeMessageCommitted checks whether the bridge message has been committed up to the given
// block (including the given block).
func (b *bridgeManager) isBridgeMessageCommitted(
	blockHeader *types.Header,
	msg *contractsapi.BridgeMessage) (bool, error) {
	//
	// The verification process for determining whether a message has been committed depends on
	// whether it is an ordinary or a rollback message.
	//
	// For an ordinary message, the verification is done by checking whether the message ID is
	// smaller than the ID of the next message to be committed on the BridgeStorage.
	//
	// For a rollback message, the verification is done by checking whether the message ID is
	// present in the corresponding list of committed rollback messages on the BridgeStorage.
	//
	provider, err := b.blockchain.GetStateProviderForBlock(blockHeader)
	if err != nil {
		return false, err
	}

	sysState := b.blockchain.GetSystemState(provider)

	if msg.IsRollback {
		if b.internalChainID == msg.SourceChainID.Uint64() {
			return sysState.GetCommittedRollbackedI2E(b.externalChainID, msg.ID)
		} else {
			return sysState.GetCommittedRollbackedE2I(b.externalChainID, msg.ID)
		}
	}

	// If it is not a rollback message, the following applies.

	var nextToCommit uint64

	if b.internalChainID == msg.SourceChainID.Uint64() {
		nextToCommit, err = sysState.GetNextCommittedIndex(b.externalChainID, systemstate.I2E)
	} else {
		nextToCommit, err = sysState.GetNextCommittedIndex(b.externalChainID, systemstate.E2I)
	}

	if err != nil {
		return false, err
	}

	if msg.ID.Uint64() < nextToCommit {
		return true, nil
	}

	return false, nil
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
