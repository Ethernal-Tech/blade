package bridge

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/abi"
	"github.com/hashicorp/go-hclog"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/blockchain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/oracle"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/signer"
	systemstate "github.com/0xPolygon/polygon-edge/consensus/polybft/system_state"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/validator"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/types"
)

var bigZero = big.NewInt(0)

func newTestState(t *testing.T) *BridgeManagerStore {
	t.Helper()

	dir := fmt.Sprintf("/tmp/consensus-temp_%v", time.Now().UTC().Format(time.RFC3339Nano))
	err := os.Mkdir(dir, 0775)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

	db, err := bolt.Open(path.Join(dir, "my.db"), 0666, nil)
	if err != nil {
		t.Fatal(err)
	}

	numOfBridges := uint64(10)
	chainIds := make([]uint64, 0, numOfBridges)

	for i := uint64(1); i <= numOfBridges; i++ {
		chainIds = append(chainIds, i)
	}

	store, err := newBridgeManagerStore(db, nil, chainIds, 100)
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func newTestBridgeManager(t *testing.T, key *validator.TestValidator, runtime Runtime, jsonClient JSONRPCClient, blockchain blockchain.Blockchain) *bridgeEventManager {
	t.Helper()

	state := newTestState(t)
	require.NoError(t, state.insertEpoch(0, nil, 1))

	topic := &mockTopic{}

	s := newBridgeManager(
		hclog.NewNullLogger(),
		state,
		&bridgeEventManagerConfig{
			bridgeCfg:         &config.Bridge{BridgeBatchThreshold: 100},
			topic:             topic,
			key:               key.Key(),
			maxNumberOfEvents: maxNumberOfBatchEvents,
		}, runtime, jsonClient, 1, 100, blockchain)

	s.nextEventIDE2I = 1
	s.nextEventIDI2E = 1
	s.externalConfirmationDepth = big.NewInt(5)

	return s
}

func TestBridgeEventManager_PostEpoch_BuildBridgeBatch(t *testing.T) {
	t.Parallel()

	vals := validator.NewTestValidators(t, 5)

	blockchain := new(blockchain.BlockchainMock)
	blockchain.On("CurrentHeader").Return(&types.Header{Number: 10})

	t.Run("When node is validator", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, blockchain)

		require.NoError(t, s.buildE2IBridgeBatch(nil, nil))
		require.Nil(t, s.pendingBridgeBatchesE2I)

		bridgeMessages10 := generateBridgeMessageEvents(t, 10, 1)

		for i := 0; i < 5; i++ {
			require.NoError(t, s.state.insertBridgeMessageEvent(bridgeMessages10[i], false, nil))
		}

		require.NoError(t, s.buildE2IBridgeBatch(nil, nil))

		length := len(s.pendingBridgeBatchesE2I[0].Messages)
		require.Len(t, s.pendingBridgeBatchesE2I, 1)
		require.Equal(t, uint64(1), s.pendingBridgeBatchesE2I[0].Messages[0].ID.Uint64())
		require.Equal(t, uint64(5), s.pendingBridgeBatchesE2I[0].Messages[length-1].ID.Uint64())
		require.Equal(t, uint64(0), s.pendingBridgeBatchesE2I[0].Epoch)

		for i := 5; i < 10; i++ {
			require.NoError(t, s.state.insertBridgeMessageEvent(bridgeMessages10[i], false, nil))
		}

		require.NoError(t, s.buildE2IBridgeBatch(nil, nil))

		length = len(s.pendingBridgeBatchesE2I[1].Messages)
		require.Len(t, s.pendingBridgeBatchesE2I, 2)
		require.Equal(t, uint64(1), s.pendingBridgeBatchesE2I[1].Messages[0].ID.Uint64())
		require.Equal(t, uint64(10), s.pendingBridgeBatchesE2I[1].Messages[length-1].ID.Uint64())
		require.Equal(t, uint64(0), s.pendingBridgeBatchesE2I[1].Epoch)

		require.NotNil(t, s.config.topic.(*mockTopic).consume())
	})

	t.Run("When node is not validator", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: false}, nil, blockchain)

		bridgeMessages10 := generateBridgeMessageEvents(t, 10, 1)

		for i := 0; i < 5; i++ {
			require.NoError(t, s.state.insertBridgeMessageEvent(bridgeMessages10[i], false, nil))
		}

		require.NoError(t, s.buildE2IBridgeBatch(nil, nil))
		require.Len(t, s.pendingBridgeBatchesE2I, 0)
	})
}

func TestBridgeEventManager_MessagePool(t *testing.T) {
	t.Parallel()

	vals := validator.NewTestValidators(t, 5)

	blockchain := new(blockchain.BlockchainMock)
	blockchain.On("CurrentHeader").Return(&types.Header{Number: 10})

	t.Run("Old epoch", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, nil)

		s.epoch = 1
		msg := &BridgeBatchVote{
			EpochNumber: 0,
		}

		err := s.saveVote(msg)
		require.NoError(t, err)
	})

	t.Run("Sender is not a validator", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, nil)
		s.validatorSet = vals.ToValidatorSet()

		badVal := validator.NewTestValidator(t, "a", 0)
		msg, err := newMockMsg().sign(badVal, signer.DomainBridge)
		require.NoError(t, err)

		msg.SourceChainID = 1
		msg.DestinationChainID = 100

		require.Error(t, s.saveVote(msg))
	})

	t.Run("Invalid epoch", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, nil)
		s.validatorSet = vals.ToValidatorSet()

		val := newMockMsg()
		msg, err := val.sign(vals.GetValidator("0"), signer.DomainBridge)
		require.NoError(t, err)

		// invalid epoch +2
		msg.EpochNumber = 2

		require.NoError(t, s.saveVote(msg))

		// no votes for the current epoch
		votes, err := s.state.getMessageVotes(0, msg.Hash, 1)
		require.NoError(t, err)
		require.Len(t, votes, 0)

		// returns an error for the invalid epoch
		_, err = s.state.getMessageVotes(1, msg.Hash, 1)
		require.Error(t, err)
	})

	t.Run("Sender and signature mismatch", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, nil)
		s.validatorSet = vals.ToValidatorSet()

		// validator signs the msg in behalf of another validator
		val := newMockMsg()
		msg, err := val.sign(vals.GetValidator("0"), signer.DomainBridge)
		require.NoError(t, err)

		msg.SourceChainID = 1
		msg.DestinationChainID = 100

		msg.Sender = vals.GetValidator("1").Address().String()
		require.Error(t, s.saveVote(msg))

		// non validator signs the msg in behalf of a validator
		badVal := validator.NewTestValidator(t, "a", 0)
		msg, err = newMockMsg().sign(badVal, signer.DomainBridge)
		require.NoError(t, err)

		msg.SourceChainID = 1
		msg.DestinationChainID = 100

		msg.Sender = vals.GetValidator("1").Address().String()
		require.Error(t, s.saveVote(msg))
	})

	t.Run("Sender votes", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, nil)
		s.validatorSet = vals.ToValidatorSet()

		msg := newMockMsg()
		val1signed, err := msg.sign(vals.GetValidator("1"), signer.DomainBridge)
		require.NoError(t, err)

		val1signed.SourceChainID = 1
		val1signed.DestinationChainID = 100

		val2signed, err := msg.sign(vals.GetValidator("2"), signer.DomainBridge)
		require.NoError(t, err)

		val2signed.SourceChainID = 1
		val2signed.DestinationChainID = 100

		// vote with validator 1
		require.NoError(t, s.saveVote(val1signed))

		votes, err := s.state.getMessageVotes(0, msg.hash, 1)
		require.NoError(t, err)
		require.Len(t, votes, 1)

		// vote with validator 1 again (the votes do not increase)
		require.NoError(t, s.saveVote(val1signed))
		votes, _ = s.state.getMessageVotes(0, msg.hash, 1)
		require.Len(t, votes, 1)

		// vote with validator 2
		require.NoError(t, s.saveVote(val2signed))
		votes, _ = s.state.getMessageVotes(0, msg.hash, 1)
		require.Len(t, votes, 2)
	})
}

func TestBridgeEventManager_BuildBridgeBatch(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, nil)
	s.validatorSet = vals.ToValidatorSet()

	// batches are empty
	batches, err := s.BridgeBatch(1)
	require.NoError(t, err)
	require.Len(t, batches, 0)

	bridgeMessage := contractsapi.BridgeMessage{
		ID:                 big.NewInt(1),
		SourceChainID:      big.NewInt(1),
		DestinationChainID: big.NewInt(2),
		IsRollback:         false,
	}

	bridgeMessageBatch := contractsapi.BridgeMessageBatch{
		Messages:           []*contractsapi.BridgeMessage{&bridgeMessage},
		SourceChainID:      big.NewInt(1),
		DestinationChainID: big.NewInt(2),
		Threshold:          big.NewInt(10000),
		CommitCounter:      big.NewInt(1),
	}

	s.pendingBridgeBatchesE2I = []*PendingBridgeBatch{
		{&bridgeMessageBatch, 0},
	}

	hash, err := s.pendingBridgeBatchesE2I[0].Hash()
	require.NoError(t, err)

	msg := newMockMsg().WithHash(hash.Bytes())

	// validators 0 and 1 vote for the proposal, there is not enough
	// voting power for the proposal
	signedMsg1, err := msg.sign(vals.GetValidator("0"), signer.DomainBridge)
	require.NoError(t, err)

	signedMsg1.SourceChainID = 1
	signedMsg1.DestinationChainID = 100

	signedMsg2, err := msg.sign(vals.GetValidator("1"), signer.DomainBridge)
	require.NoError(t, err)

	signedMsg2.SourceChainID = 1
	signedMsg2.DestinationChainID = 100

	require.NoError(t, s.saveVote(signedMsg1))
	require.NoError(t, s.saveVote(signedMsg2))

	batches, err = s.BridgeBatch(0)
	require.NoError(t, err) // there is no error if quorum is not met, since its a valid case
	require.Len(t, batches, 0)

	// validator 2 and 3 vote for the proposal, there is enough voting power now

	signedMsg1, err = msg.sign(vals.GetValidator("2"), signer.DomainBridge)
	require.NoError(t, err)

	signedMsg1.SourceChainID = 1
	signedMsg1.DestinationChainID = 100

	signedMsg2, err = msg.sign(vals.GetValidator("3"), signer.DomainBridge)
	require.NoError(t, err)

	signedMsg2.SourceChainID = 1
	signedMsg2.DestinationChainID = 100

	require.NoError(t, s.saveVote(signedMsg1))
	require.NoError(t, s.saveVote(signedMsg2))

	batches, err = s.BridgeBatch(1)
	require.NoError(t, err)
	require.NotNil(t, batches)
}

func TestBridgeEventManager_RemoveProcessedEvents(t *testing.T) {
	const bridgeMessageEventsCount = 5

	sysState := new(systemstate.SystemStateMock)
	sysState.On("GetNextCommittedIndex").Return(uint64(1000))

	blockchain := new(blockchain.BlockchainMock)
	blockchain.On("GetStateProviderForBlock", mock.Anything).Return(nil)
	blockchain.On("GetSystemState", mock.Anything).Return(sysState)
	blockchain.On("CurrentHeader", mock.Anything).Return(&types.Header{Number: 10})

	vals := validator.NewTestValidators(t, 5)

	s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, blockchain)
	bridgeMessageEvents := generateBridgeMessageEvents(t, bridgeMessageEventsCount, 1)

	for _, event := range bridgeMessageEvents {
		require.NoError(t, s.state.insertBridgeMessageEvent(event, false, nil))
	}

	bridgeMessageEventsBefore, err := s.state.list(s.internalChainID)
	require.NoError(t, err)
	require.Equal(t, bridgeMessageEventsCount, len(bridgeMessageEventsBefore))

	for _, event := range bridgeMessageEvents {
		eventLog := createTestLogForBridgeMessageResultEvent(t, event.ID.Uint64())
		require.NoError(t, s.ProcessLog(&types.Header{Number: 10}, eventLog, nil))
	}

	// all bridge message events and their proofs should be removed from the store
	stateSyncEventsAfter, err := s.state.list(s.internalChainID)
	require.NoError(t, err)
	require.Equal(t, 0, len(stateSyncEventsAfter))
}

func Test_handleBridgeMessageEvent(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	// The following sequence of tests effectively covers all cases related to the ProcessLog
	// and AddLog methods when a `BridgeMsg` event occurs. The only additional functionality
	// these methods have, besides calling "handleBridgeMessageEvent", is checking whether the
	// event belongs to the current bridge manager and parsing the event itself (which should
	// be tested separately outside the bridge context). All tests handle the I2E message, but
	// the approach is exactly the same for the E2I messages.

	bc := &blockchain.BlockchainMock{}
	ss := &systemstate.SystemStateMock{}

	bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
	bc.On("GetSystemState", mock.Anything).Return(ss)
	ss.On("GetNextCommittedIndex", mock.Anything).Return(uint64(10))

	// Function to check whether a message has been processed correctly. The types of possible
	// checks are as follows (typeOfCheck argument):
	//	1 - the message should not even go into the finalization (fin.) phase at all
	//	2 - the message should go into the fin. phase and it has been successfully executed
	//	3 - the message should go into the fin. phase and it has been unsuccessfully executed
	checkFn := func(
		msgEvent *contractsapi.BridgeMsgEvent,
		bm *bridgeEventManager,
		typeOfCheck int) {
		// Function to check whether the message is part of the ordinary bucket.
		getOrdinaryMsg := func() (*contractsapi.BridgeMsgEvent, error) {
			return bm.state.getBridgeMessageEvent(
				msgEvent.ID,
				msgEvent.SourceChainID,
				msgEvent.DestinationChainID,
				false,
				nil)
		}

		// Function to check whether the message is part of the rollback bucket.
		getRollbackMsg := func() (*contractsapi.BridgeMsgEvent, error) {
			return bm.state.getBridgeMessageEvent(
				msgEvent.ID,
				msgEvent.DestinationChainID,
				msgEvent.SourceChainID,
				true,
				nil)
		}

		switch typeOfCheck {
		case 1:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.NotNil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)
		case 2:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)
		case 3:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.NotNil(t, msgEvent)
		}
	}

	// This test illustrates a scenario where the new bridge message arrives in the system, and
	// it has neither been committed nor executed before.
	//
	// Expected: the message should only be written to the database, specifically to the bucket
	// reserver for ordinary messages.
	t.Run("1", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(10),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err := bm.handleBridgeMessageEvent(nil, msgEvent, nil)
		require.NoError(t, err)

		checkFn(msgEvent, bm, 1)
	})

	// This test illustrates a scenario where the new bridge message arrives in the system, and
	// it has been previously committed but not executed.
	//
	// Expected: the message should only be written to the database, specifically to the bucket
	// reserved for ordinary messages.
	t.Run("2", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err := bm.handleBridgeMessageEvent(nil, msgEvent, nil)
		require.NoError(t, err)

		checkFn(msgEvent, bm, 1)
	})

	// This test illustrates a scenario where the new bridge message arrives in the system, and
	// it has been previously executed but not committed.
	//
	// Expected: the message should only be written to the database, specifically to the bucket
	// reserved for ordinary messages.
	t.Run("3", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(10),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(10),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err := bm.state.insertBridgeMessageResultEvent(msgResultEvent, nil)
		require.NoError(t, err)

		err = bm.handleBridgeMessageEvent(nil, msgEvent, nil)
		require.NoError(t, err)

		checkFn(msgEvent, bm, 1)
	})

	// This test illustrates a scenario where the new bridge message arrives in the system, and
	// it has been previously committed and executed. The execution result is successful.
	//
	// Expected: nothing should be done, i.e., the message should be first written to the bucket
	// reserved for ordinary messages and then, since it has already been successfully executed,
	// removed from it.
	t.Run("4", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Status:             true,
		}

		err := bm.state.insertBridgeMessageResultEvent(msgResultEvent, nil)
		require.NoError(t, err)

		err = bm.handleBridgeMessageEvent(nil, msgEvent, nil)
		require.NoError(t, err)

		checkFn(msgEvent, bm, 2)
	})

	// This test illustrates a scenario where the new bridge message arrives in the system, and
	// it has been previously committed and executed. The execution result is unsuccessful.
	//
	// Expected: the message should only be written to the bucket reserved for rollback messages.
	t.Run("5", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Status:             false,
		}

		err := bm.state.insertBridgeMessageResultEvent(msgResultEvent, nil)
		require.NoError(t, err)

		err = bm.handleBridgeMessageEvent(nil, msgEvent, nil)
		require.NoError(t, err)

		checkFn(msgEvent, bm, 3)
	})
}

func Test_handleBridgeMessageResultEvent(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	// The following sequence of tests effectively covers all cases related to the ProcessLog
	// and (~) AddLog methods when a `BridgeMessageResult` event occurs. Note, there are some
	// extra, separated tests for verifying whether the unexecuted list is correctly handled
	// when the given event occurs in the context of AddLog. The only additional functionality
	// these methods have, besides calling "handleBridgeMessageResultEvent", is checking whether
	// the event belongs to the current bridge manager and parsing the event itself (which should
	// be tested separately outside the bridge context). All tests handle the I2E message, but
	// the approach is exactly the same for the E2I messages.

	bc := &blockchain.BlockchainMock{}
	ss := &systemstate.SystemStateMock{}

	bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
	bc.On("GetSystemState", mock.Anything).Return(ss)
	ss.On("GetNextCommittedIndex", mock.Anything).Return(uint64(10))

	// Function to check whether the execution result of an ordinary message has been processed
	// correctly. The types of possible checks are as follows (typeOfCheck argument):
	//	1 - the message is not known and it should not go into the finalization (fin.) phase
	//	2 - the message in known, but it should not go into the finalization (fin.) phase
	//	3 - the message should go into the fin. phase and it has been successfully executed
	//	4 - the message should go into the fin. phase and it has been unsuccessfully executed
	checkFn := func(
		msgResultEvent *contractsapi.BridgeMessageResultEvent,
		bm *bridgeEventManager,
		typeOfCheck int) {
		// Function to check whether the execution result of a message is part of the ordinary
		// bucket.
		getMsgResult := func() (*contractsapi.BridgeMessageResultEvent, error) {
			return bm.state.getBridgeMessageResult(
				&contractsapi.BridgeMessage{
					ID:                 msgResultEvent.ID,
					SourceChainID:      msgResultEvent.SourceChainID,
					DestinationChainID: msgResultEvent.DestinationChainID,
					IsRollback:         false,
				},
				nil)
		}

		// Function to check whether the message is part of the ordinary bucket.
		getOrdinaryMsg := func() (*contractsapi.BridgeMsgEvent, error) {
			return bm.state.getBridgeMessageEvent(
				msgResultEvent.ID,
				msgResultEvent.SourceChainID,
				msgResultEvent.DestinationChainID,
				false,
				nil)
		}

		// Function to check whether the message is part of the rollback bucket.
		getRollbackMsg := func() (*contractsapi.BridgeMsgEvent, error) {
			return bm.state.getBridgeMessageEvent(
				msgResultEvent.ID,
				msgResultEvent.DestinationChainID,
				msgResultEvent.SourceChainID,
				true,
				nil)
		}

		msgEventResult, err := getMsgResult()

		require.NoError(t, err)
		require.NotNil(t, msgEventResult)

		switch typeOfCheck {
		case 1, 3:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)
		case 2:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.NotNil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)
		case 4:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.NotNil(t, msgEvent)
		}
	}

	// This test illustrates a scenario where the new execution result for an ordinary message
	// arrives in the system, and the ordinary message itself is neither known nor committed.
	//
	// Expected: the execution result should only be written to the database, specifically to
	// the bucket reserved for executed ordinary messages.
	t.Run("1", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(10),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Status:             true,
		}

		err := bm.handleBridgeMessageResultEvent(nil, msgResultEvent, nil)
		require.NoError(t, err)

		checkFn(msgResultEvent, bm, 1)
	})

	// This test illustrates a scenario where the new execution result for an ordinary message
	// arrives in the system, and the ordinary message is committed, but not known.
	//
	// Expected: the execution result should only be written to the database, specifically to
	// the bucket reserved for executed ordinary messages.
	t.Run("2", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Status:             true,
		}

		err := bm.handleBridgeMessageResultEvent(nil, msgResultEvent, nil)
		require.NoError(t, err)

		checkFn(msgResultEvent, bm, 1)
	})

	// This test illustrates a scenario where the new execution result for an ordinary message
	// arrives in the system, and the ordinary message is known, but not committed.
	//
	// Expected: the execution result should only be written to the database, specifically to
	// the bucket reserved for executed ordinary messages.
	t.Run("3", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(10),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err := bm.state.insertBridgeMessageEvent(msgEvent, false, nil)
		require.NoError(t, err)

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(10),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Status:             true,
		}

		err = bm.handleBridgeMessageResultEvent(nil, msgResultEvent, nil)
		require.NoError(t, err)

		checkFn(msgResultEvent, bm, 2)
	})

	// This test illustrates a scenario where the new execution result for an ordinary message
	// arrives in the system, and the ordinary message is known and committed. The execution
	// result is successful.
	//
	// Expected: the execution result should be written to the database, more precisely, to the
	// bucket reserved for executed ordinary messages, while the message should be removed from
	// the ordinary bucket.
	t.Run("4", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err := bm.state.insertBridgeMessageEvent(msgEvent, false, nil)
		require.NoError(t, err)

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Status:             true,
		}

		err = bm.handleBridgeMessageResultEvent(nil, msgResultEvent, nil)
		require.NoError(t, err)

		checkFn(msgResultEvent, bm, 3)
	})

	// This test illustrates a scenario where the new execution result for an ordinary message
	// arrives in the system, and the ordinary message is known and committed. The execution
	// result is unsuccessful.
	//
	// Expected: the execution result should be written to the database, more precisely, to the
	// bucket reserved for executed rollback messages, while the message should be removed from
	// the ordinary bucket and added to the rollback bucket.
	t.Run("5", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err := bm.state.insertBridgeMessageEvent(msgEvent, false, nil)
		require.NoError(t, err)

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Status:             false,
		}

		err = bm.handleBridgeMessageResultEvent(nil, msgResultEvent, nil)
		require.NoError(t, err)

		checkFn(msgResultEvent, bm, 4)
	})

	// This test illustrates a scenario where the new execution result for a rollback message
	// arrives in the system. Whether the message is committed or known, as well as whether it
	// was successfully or unsuccessfully executed, has no impact, since the execution result
	// of a rollback message is always handled in the same way.
	//
	// Expected: the execution result should only be written to the database, specifically to
	// the bucket reserved for executed rollback messages.
	t.Run("6", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		msgResultEvent := &contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(5),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Status:             false,
		}

		err := bm.handleBridgeMessageResultEvent(nil, msgResultEvent, nil)
		require.NoError(t, err)

		msgResultEvent, err = bm.state.getBridgeMessageResult(msg, nil)
		require.NoError(t, err)
		require.NotNil(t, msgResultEvent)
	})
}

func Test_AddLog_Unexecuted_list(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	bc := &blockchain.BlockchainMock{}
	bc.On("CurrentHeader", mock.Anything).Return(&types.Header{})

	// Function to create dummy `BridgeMessageResult` log for an I2E message.
	createLogFn := func(id uint64, isRollback bool) *ethgo.Log {
		data, err := abi.MustNewType("tuple(uint256 a, uint256 b, bool c, bytes d)").
			Encode([]interface{}{big.NewInt(100), big.NewInt(1), isRollback, []byte("")})
		require.NoError(t, err)

		return &ethgo.Log{
			Topics: []ethgo.Hash{
				bridgeMessageResultEventSig,
				ethgo.BytesToHash(common.EncodeUint64ToBytes(id)),
				ethgo.BytesToHash(common.EncodeUint64ToBytes(1)),
			},
			Data: data,
		}
	}

	// Function to create a batch with the given ordinary and rollback messages.
	createBatchFn := func(numOfOrdMsgs int64, rollbackMsgIds ...int64) *PendingBridgeBatch {
		totalNum := numOfOrdMsgs + int64(len(rollbackMsgIds))

		msgs := make([]*contractsapi.BridgeMessage, totalNum)

		for i := int64(0); i < numOfOrdMsgs; i++ {
			msgs[i] = &contractsapi.BridgeMessage{
				ID:                 big.NewInt(i + 1),
				SourceChainID:      big.NewInt(100),
				DestinationChainID: big.NewInt(1),
			}
		}

		for i, id := range rollbackMsgIds {
			msgs[numOfOrdMsgs+int64(i)] = &contractsapi.BridgeMessage{
				ID:                 big.NewInt(id),
				SourceChainID:      big.NewInt(100),
				DestinationChainID: big.NewInt(1),
				IsRollback:         true,
			}
		}

		return &PendingBridgeBatch{
			BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
				Messages:           msgs,
				SourceChainID:      big.NewInt(100),
				DestinationChainID: big.NewInt(1),
				Threshold:          big.NewInt(0),
				CommitCounter:      big.NewInt(0),
			},
		}
	}

	// Function to insert an execution results of the messages.
	insertFn := func(bm *bridgeEventManager, isRollback bool, ids ...int64) {
		for _, id := range ids {
			err := bm.state.insertBridgeMessageResultEvent(
				&contractsapi.BridgeMessageResultEvent{
					ID:                 big.NewInt(id),
					SourceChainID:      big.NewInt(100),
					DestinationChainID: big.NewInt(1),
					IsRollback:         isRollback,
				}, nil)

			require.NoError(t, err)
		}
	}

	// This test illustrates a scenario where a batch containing two ordinary and two rollback
	// messages (IDs for both types are 1 and 2) exists in the unexecuted list, with only the
	// ordinary message with ID 2 remaining unexecuted, where execution result for an ordinary
	// message that is not part of the batch (ID 3) arrives.
	//
	// Expected: the unexecuted list should remain unchanged.
	t.Run("1", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		batch := createBatchFn(2, 1, 2)
		bm.unexecutedBatches = append(bm.unexecutedBatches, batch)

		insertFn(bm, false, 1)
		insertFn(bm, true, 1, 2)

		log := createLogFn(3, false)

		err := bm.AddLog(big.NewInt(1), log)

		require.NoError(t, err)

		require.EqualValues(t, 1, len(bm.unexecutedBatches))
	})

	// This test illustrates a scenario where a batch containing two ordinary and two rollback
	// messages (IDs for both types are 1 and 2) exists in the unexecuted list, with only the
	// rollback message with ID 1 remaining unexecuted, where execution result for a rollback
	// message that is not part of the batch (ID 3) arrives.
	//
	// Expected: the unexecuted list should remain unchanged.
	t.Run("2", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		batch := createBatchFn(2, 1, 2)
		bm.unexecutedBatches = append(bm.unexecutedBatches, batch)

		insertFn(bm, false, 1, 2)
		insertFn(bm, true, 2)

		log := createLogFn(3, true)

		err := bm.AddLog(big.NewInt(1), log)

		require.NoError(t, err)

		require.EqualValues(t, 1, len(bm.unexecutedBatches))
	})

	// This test illustrates a scenario where a batch containing two ordinary and two rollback
	// messages (IDs for both types are 1 and 2) exists in the unexecuted list, with multiple
	// messages remaining unexecuted, where an execution result arrives for an ordinary message
	// that is part of the batch.
	//
	// Expected: the unexecuted list should remain unchanged.
	t.Run("3", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		batch := createBatchFn(2, 1, 2)
		bm.unexecutedBatches = append(bm.unexecutedBatches, batch)

		insertFn(bm, false, 1)
		insertFn(bm, true, 2)

		log := createLogFn(2, false)

		err := bm.AddLog(big.NewInt(1), log)

		require.NoError(t, err)

		require.EqualValues(t, 1, len(bm.unexecutedBatches))
	})

	// This test illustrates a scenario where a batch containing two ordinary and two rollback
	// messages (IDs for both types are 1 and 2) exists in the unexecuted list, with multiple
	// messages remaining unexecuted, where an execution result arrives for a rollback message
	// that is part of the batch.
	//
	// Expected: the unexecuted list should remain unchanged.
	t.Run("4", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		batch := createBatchFn(2, 1, 2)
		bm.unexecutedBatches = append(bm.unexecutedBatches, batch)

		insertFn(bm, false, 1)
		insertFn(bm, true, 2)

		log := createLogFn(1, true)

		err := bm.AddLog(big.NewInt(1), log)

		require.NoError(t, err)

		require.EqualValues(t, 1, len(bm.unexecutedBatches))
	})

	// This test illustrates a scenario where a batch containing two ordinary and two rollback
	// messages (IDs for both types are 1 and 2) exists in the unexecuted list, with only the
	// ordinary message with ID 2 remaining unexecuted, where the execution result exactly for
	// that message arrives.
	//
	// Expected: the batch should be removed from the unexecuted list, making it empty.
	t.Run("5", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		batch := createBatchFn(2, 1, 2)
		bm.unexecutedBatches = append(bm.unexecutedBatches, batch)

		insertFn(bm, false, 1)
		insertFn(bm, true, 1, 2)

		log := createLogFn(2, false)

		err := bm.AddLog(big.NewInt(1), log)

		require.NoError(t, err)

		require.EqualValues(t, 0, len(bm.unexecutedBatches))
	})

	// This test illustrates a scenario where a batch containing two ordinary and two rollback
	// messages (IDs for both types are 1 and 2) exists in the unexecuted list, with only the
	// rollback message with ID 1 remaining unexecuted, where the execution result exactly for
	// that message arrives.
	//
	// Expected: the batch should be removed from the unexecuted list, making it empty.
	t.Run("6", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		batch := createBatchFn(2, 1, 2)
		bm.unexecutedBatches = append(bm.unexecutedBatches, batch)

		insertFn(bm, false, 1, 2)
		insertFn(bm, true, 2)

		log := createLogFn(1, true)

		err := bm.AddLog(big.NewInt(1), log)

		require.NoError(t, err)

		require.EqualValues(t, 0, len(bm.unexecutedBatches))
	})

	// This test illustrates a scenario where two batches exist in the unexecuted list, and the
	// execution result arrives for the only remaining unexecuted message in one of the batches.
	//
	// Expected: the corresponding batch should be removed from the unexecuted list, leaving only
	// one batch within list.
	t.Run("7", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		batch1 := createBatchFn(2, 1, 2)
		batch2 := createBatchFn(0, 3, 4)
		bm.unexecutedBatches = append(bm.unexecutedBatches, batch1, batch2)

		insertFn(bm, false, 1)
		insertFn(bm, true, 1, 2)

		log := createLogFn(2, false)

		err := bm.AddLog(big.NewInt(1), log)

		require.NoError(t, err)

		require.EqualValues(t, 1, len(bm.unexecutedBatches))
	})
}

func Test_updateStateOnBatchCommit(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	// All tests share a common initialization (initFn), that is, initial state, which consists
	// of the following:
	//	1. one batch exists in pendingBridgeBatchesI2E,
	// 	2. one batch exists in pendingBridgeBatchesE2I,
	//	3. one batch exists in unexecutedBatches,
	//	4. one batch exists in retryBatches,
	//	5. one batch has a corresponding list in pendingRetryBatches.
	//
	// The descriptions of all tests assume the aforementioned setup. Additionally, for tests
	// verifying the arrival of a non-retry batch, it is understood that this batch is part of
	// the corresponding pendingBridgeBatches list. Similarly, for tests examining the arrival
	// of a retry batch, it is assumed that the batch is already included in retryBatches and
	// pendingRetryBatches.

	// Function to create initial state used in all the following tests.
	initFn := func(bm *bridgeEventManager) (*PendingBridgeBatch, types.Hash) {
		msg1 := &contractsapi.BridgeMessage{
			ID:                 big.NewInt(1),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		msg2 := &contractsapi.BridgeMessage{
			ID:                 big.NewInt(2),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		i2eBatch := &PendingBridgeBatch{
			BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
				Messages:           []*contractsapi.BridgeMessage{msg1, msg2},
				SourceChainID:      big.NewInt(100),
				DestinationChainID: big.NewInt(1),
				Threshold:          big.NewInt(0),
				CommitCounter:      big.NewInt(0),
			},
		}

		baseHash, _ := i2eBatch.Hash()

		i2eBatch.CommitCounter = big.NewInt(1)

		bm.retryBatches[[32]byte{}] = PendingBridgeBatch{}
		bm.pendingRetryBatches[[32]byte{}] = []*PendingBridgeBatch{}
		bm.unexecutedBatches = append(bm.unexecutedBatches, &PendingBridgeBatch{})
		bm.pendingBridgeBatchesI2E = append(bm.pendingBridgeBatchesI2E, &PendingBridgeBatch{})
		bm.pendingBridgeBatchesE2I = append(bm.pendingBridgeBatchesE2I, &PendingBridgeBatch{})

		return i2eBatch, baseHash
	}

	// Function to check whether the state structures reflect the correct state.
	checkFn := func(bm *bridgeEventManager, unexe, penI2E, penE2I, ret, penRet int) {
		require.EqualValues(t, unexe, len(bm.unexecutedBatches))
		require.EqualValues(t, penI2E, len(bm.pendingBridgeBatchesI2E))
		require.EqualValues(t, penE2I, len(bm.pendingBridgeBatchesE2I))
		require.EqualValues(t, ret, len(bm.retryBatches))
		require.EqualValues(t, penRet, len(bm.pendingRetryBatches))
	}

	// This test illustrates a scenario where an E2I batch comes, that is, being committed.
	//
	// Expected: the E2I pending bridge batch list (pendingBridgeBatchesE2I) should be cleared.
	t.Run("1", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		initFn(bm)

		checkFn(bm, 1, 1, 1, 1, 1)

		bm.updateStateOnBatchCommit(
			big.NewInt(1),
			[32]byte{},
			[32]byte{},
			&PendingBridgeBatch{
				BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
					DestinationChainID: big.NewInt(100),
					SourceChainID:      big.NewInt(1),
				},
			},
			nil)

		checkFn(bm, 1, 1, 0, 1, 1)
	})

	// This test illustrates a scenario where an I2E non-retry batch arrives, with some of its
	// messages not yet executed.
	//
	// Expected: the I2E pending bridge batch list (pendingBridgeBatchesI2E) should be cleared,
	// while the number of batches within the unexecuted list (unexecutedBatches) should increase
	// by one.
	t.Run("2", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		batch, hash := initFn(bm)
		bm.pendingBridgeBatchesI2E = append(bm.pendingBridgeBatchesI2E, batch)

		err := bm.state.insertBridgeMessageResultEvent(&contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(1),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}, nil)

		require.NoError(t, err)

		checkFn(bm, 1, 2, 1, 1, 1)

		bm.updateStateOnBatchCommit(big.NewInt(1), hash, [32]byte{}, batch, nil)

		checkFn(bm, 2, 0, 1, 1, 1)
	})

	// This test illustrates a scenario where an I2E non-retry batch arrives, and all of its
	// messages have already been previously executed.
	//
	// Expected: the I2E pending bridge batch list (pendingBridgeBatchesI2E) should be cleared.
	t.Run("3", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		batch, hash := initFn(bm)
		bm.pendingBridgeBatchesI2E = append(bm.pendingBridgeBatchesI2E, batch)

		err := bm.state.insertBridgeMessageResultEvent(&contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(1),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}, nil)

		require.NoError(t, err)

		err = bm.state.insertBridgeMessageResultEvent(&contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(2),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}, nil)

		require.NoError(t, err)

		checkFn(bm, 1, 2, 1, 1, 1)

		bm.updateStateOnBatchCommit(big.NewInt(1), hash, [32]byte{}, batch, nil)

		checkFn(bm, 1, 0, 1, 1, 1)
	})

	// This test illustrates a scenario where an I2E retry batch arrives, with some of its messages
	// not yet executed.
	//
	// Expected: the batch should be removed from the retryBatches and its list of pending retry
	// batches should be removed from pendingRetryBatches. Also, the number of batches within the
	// unexecuted list (unexecutedBatches) should increase by 1.
	t.Run("4", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		batch, hash := initFn(bm)
		bm.retryBatches[hash] = PendingBridgeBatch{}
		bm.pendingRetryBatches[hash] = []*PendingBridgeBatch{}
		batch.CommitCounter = big.NewInt(2)

		err := bm.state.insertBridgeMessageResultEvent(&contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(1),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}, nil)

		require.NoError(t, err)

		checkFn(bm, 1, 1, 1, 2, 2)

		bm.updateStateOnBatchCommit(big.NewInt(1), hash, [32]byte{}, batch, nil)

		checkFn(bm, 2, 1, 1, 1, 1)
	})

	// This test illustrates a scenario where an I2E retry batch arrives, and all of its messages
	// have already been previously executed.
	//
	// Expected: the batch should be removed from the retryBatches and its list of pending retry
	// batches should be removed from pendingRetryBatches.
	t.Run("5", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		batch, hash := initFn(bm)
		bm.retryBatches[hash] = PendingBridgeBatch{}
		bm.pendingRetryBatches[hash] = []*PendingBridgeBatch{}
		batch.CommitCounter = big.NewInt(2)

		err := bm.state.insertBridgeMessageResultEvent(&contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(1),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}, nil)

		require.NoError(t, err)

		err = bm.state.insertBridgeMessageResultEvent(&contractsapi.BridgeMessageResultEvent{
			ID:                 big.NewInt(2),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}, nil)

		require.NoError(t, err)

		checkFn(bm, 1, 1, 1, 2, 2)

		bm.updateStateOnBatchCommit(big.NewInt(1), hash, [32]byte{}, batch, nil)

		checkFn(bm, 1, 1, 1, 1, 1)
	})
}

func Test_ProcessLog_Remove_Rollback_messages(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	bc := &blockchain.BlockchainMock{}
	ss := &systemstate.SystemStateMock{}

	msg1 := &contractsapi.BridgeMessage{
		ID:                 big.NewInt(1),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
		IsRollback:         true,
	}

	msg2 := &contractsapi.BridgeMessage{
		ID:                 big.NewInt(2),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
		IsRollback:         true,
	}

	sigBatch := &contractsapi.SignedBridgeMessageBatch{
		Batch: &contractsapi.BridgeMessageBatch{
			Messages:           []*contractsapi.BridgeMessage{msg1, msg2},
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Threshold:          big.NewInt(0),
			CommitCounter:      big.NewInt(1),
		},
	}

	bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
	bc.On("GetSystemState", mock.Anything).Return(ss)

	// Function to insert a rollback messages with the given IDs.
	insertFn := func(bm *bridgeEventManager, ids ...int64) {
		for _, id := range ids {
			err := bm.state.insertBridgeMessageEvent(
				&contractsapi.BridgeMsgEvent{
					ID:                 big.NewInt(id),
					SourceChainID:      big.NewInt(100),
					DestinationChainID: big.NewInt(1),
				}, true, nil)

			require.NoError(t, err)
		}
	}

	// Function to check whether the rollback message is removed or not.
	checkFn := func(bm *bridgeEventManager, id int64, shouldBeRemoved bool) {
		msg, err := bm.state.getBridgeMessageEvent(
			big.NewInt(id),
			big.NewInt(100),
			big.NewInt(1),
			true,
			nil)
		require.NoError(t, err)

		if shouldBeRemoved {
			require.Nil(t, msg)

			return
		}

		require.NotNil(t, msg)
	}

	// This test illustrates a scenario where the retry batch with two rollback messages is
	// committed (IDs 1 and 2), while the database contains 4 rollback messages with IDs 1,
	// 2, 3, and 4.
	//
	// Expected: messages with IDs 1 and 2 should be removed from the database.
	t.Run("1", func(t *testing.T) {
		ss.On("GetBridgeBatchByNumber", mock.Anything).Return(sigBatch)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		log := &ethgo.Log{
			Topics: []ethgo.Hash{
				newBatchEventSig,
				ethgo.BytesToHash(common.EncodeUint64ToBytes(1)),
			},
		}

		insertFn(bm, 1, 2, 3, 4)

		err := bm.ProcessLog(nil, log, nil)

		require.NoError(t, err)

		checkFn(bm, 1, true)
		checkFn(bm, 2, true)
		checkFn(bm, 3, false)
		checkFn(bm, 4, false)
	})

	// This test illustrates a scenario where the non-retry batch with two rollback messages is
	// committed (IDs 1 and 2), while the database contains 4 rollback messages with IDs 1, 2,
	// 3, and 4.
	//
	// Expected: no message should be removed.
	t.Run("2", func(t *testing.T) {
		sigBatch.Batch.CommitCounter = big.NewInt(2)
		ss.On("GetBridgeBatchByNumber", mock.Anything).Return(sigBatch)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		log := &ethgo.Log{
			Topics: []ethgo.Hash{
				newBatchEventSig,
				ethgo.BytesToHash(common.EncodeUint64ToBytes(1)),
			},
		}

		insertFn(bm, 1, 2, 3, 4)

		err := bm.ProcessLog(nil, log, nil)

		require.NoError(t, err)

		for i := range 4 {
			checkFn(bm, int64(i+1), false)
		}
	})
}

func Test_handleBridgeMessageCommitment(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	// All tests handle the I2E message, but the approach is exactly the same for the E2I messages.

	msg := &contractsapi.BridgeMessage{
		ID:                 big.NewInt(10),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
	}

	// Function to insert message and its execution result. The types of possible inserts are as
	// follows (typeOfInsert argument):
	//	1 - only the message should be inserted
	//	2 - only the execution result of the message should be inserted
	//	3 - both should be inserted
	//
	// Status argument denotes whether the message was executed successfully or not.
	insertFn := func(bm *bridgeEventManager, typeOfInsert int, status bool) {
		if typeOfInsert == 1 || typeOfInsert == 3 {
			err := bm.state.insertBridgeMessageEvent(
				&contractsapi.BridgeMsgEvent{
					ID:                 big.NewInt(10),
					SourceChainID:      big.NewInt(100),
					DestinationChainID: big.NewInt(1),
				},
				false,
				nil,
			)

			require.NoError(t, err)
		}

		if typeOfInsert == 2 || typeOfInsert == 3 {
			err := bm.state.insertBridgeMessageResultEvent(
				&contractsapi.BridgeMessageResultEvent{
					ID:                 big.NewInt(10),
					SourceChainID:      big.NewInt(100),
					DestinationChainID: big.NewInt(1),
					Status:             status,
				},
				nil,
			)

			require.NoError(t, err)
		}
	}

	// Function to check whether a message has been processed correctly. The types of possible
	// checks are as follows (typeOfCheck argument):
	//	1 - the message is not known and it should not go into the finalization (fin.) phase
	//	2 - the message in known, but it should not go into the finalization (fin.) phase
	//	3 - the message should go into the fin. phase and it has been successfully executed
	//	4 - the message should go into the fin. phase and it has been unsuccessfully executed
	checkFn := func(bm *bridgeEventManager, typeOfCheck int) {
		// Function to check whether the message is part of the ordinary bucket.
		getOrdinaryMsg := func() (*contractsapi.BridgeMsgEvent, error) {
			return bm.state.getBridgeMessageEvent(
				msg.ID,
				msg.SourceChainID,
				msg.DestinationChainID,
				false,
				nil)
		}

		// Function to check whether the message is part of the rollback bucket.
		getRollbackMsg := func() (*contractsapi.BridgeMsgEvent, error) {
			return bm.state.getBridgeMessageEvent(
				msg.ID,
				msg.DestinationChainID,
				msg.SourceChainID,
				true,
				nil)
		}

		switch typeOfCheck {
		case 1, 3:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)
		case 2:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.NotNil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)
		case 4:
			msgEvent, err := getOrdinaryMsg()

			require.NoError(t, err)
			require.Nil(t, msgEvent)

			msgEvent, err = getRollbackMsg()

			require.NoError(t, err)
			require.NotNil(t, msgEvent)
		}
	}

	t.Run("1", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		err := bm.handleBridgeMessageCommitment(msg, nil)
		require.NoError(t, err)

		checkFn(bm, 1)
	})

	t.Run("2", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		insertFn(bm, 1, true)

		err := bm.handleBridgeMessageCommitment(msg, nil)
		require.NoError(t, err)

		checkFn(bm, 2)
	})

	t.Run("3", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		insertFn(bm, 2, true)

		err := bm.handleBridgeMessageCommitment(msg, nil)
		require.NoError(t, err)

		checkFn(bm, 1)
	})

	t.Run("4", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		insertFn(bm, 3, true)

		err := bm.handleBridgeMessageCommitment(msg, nil)
		require.NoError(t, err)

		checkFn(bm, 3)
	})

	t.Run("5", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		insertFn(bm, 3, false)

		err := bm.handleBridgeMessageCommitment(msg, nil)
		require.NoError(t, err)

		checkFn(bm, 4)
	})
}

func Test_isBridgeMessageCommitted(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	// This test illustrates a scenario where the commitment of an I2E ordinary message with ID
	// 4 is checked, while the last committed I2E ordinary message has ID 6, that is, the next
	// one to be committed should have ID 7.
	//
	// Expected: the message should be reported as committed (true).
	t.Run("1", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetNextCommittedIndex", mock.Anything).Return(uint64(7))

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(4),
			SourceChainID: big.NewInt(100),
			IsRollback:    false,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, true, committed)
	})

	// This test illustrates a scenario where the commitment of an E2I ordinary message with ID
	// 4 is checked, while the last committed E2I ordinary message has ID 6, that is, the next
	// one to be committed should have ID 7.
	//
	// Expected: the message should be reported as committed (true).
	t.Run("2", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetNextCommittedIndex", mock.Anything).Return(uint64(7))

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(4),
			SourceChainID: big.NewInt(1),
			IsRollback:    false,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, true, committed)
	})

	// This test illustrates a scenario where the commitment of an I2E ordinary message with ID
	// 10 is checked, while the last committed I2E ordinary message has ID 6, that is, the next
	// one to be committed should have ID 7.
	//
	// Expected: the message should be reported as uncommitted (false).
	t.Run("3", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetNextCommittedIndex", mock.Anything).Return(uint64(7))

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(10),
			SourceChainID: big.NewInt(100),
			IsRollback:    false,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, false, committed)
	})

	// This test illustrates a scenario where the commitment of an E2I ordinary message with ID
	// 10 is checked, while the last committed E2I ordinary message has ID 6, that is, the next
	// one to be committed should have ID 7.
	//
	// Expected: the message should be reported as uncommitted (false).
	t.Run("4", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetNextCommittedIndex", mock.Anything).Return(uint64(7))

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(10),
			SourceChainID: big.NewInt(1),
			IsRollback:    false,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, false, committed)
	})

	// This test illustrates a scenario where the commitment of an I2E ordinary message with ID
	// 7 is checked, while the last committed I2E ordinary message has ID 6, that is, the next
	// one to be committed should have ID 7. This is a "borderline" case.
	//
	// Expected: the message should be reported as uncommitted (false).
	t.Run("5", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetNextCommittedIndex", mock.Anything).Return(uint64(7))

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(7),
			SourceChainID: big.NewInt(100),
			IsRollback:    false,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, false, committed)
	})

	// This test illustrates a scenario where the commitment of an E2I ordinary message with ID
	// 7 is checked, while the last committed E2I ordinary message has ID 6, that is, the next
	// one to be committed should have ID 7. This is a "borderline" case.
	//
	// Expected: the message should be reported as uncommitted (false).
	t.Run("6", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetNextCommittedIndex", mock.Anything).Return(uint64(7))

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(7),
			SourceChainID: big.NewInt(1),
			IsRollback:    false,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, false, committed)
	})

	// This test illustrates a scenario where the commitment of an I2E rollback message with ID
	// 4 is checked, while the given ID is already present in the list of committed I2E rollback
	// messages on the BridgeStorage smart contract.
	//
	// Expected: the message should be reported as committed (true).
	t.Run("7", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetCommittedRollbackedI2E", mock.Anything).Return(true)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(4),
			SourceChainID: big.NewInt(100),
			IsRollback:    true,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, true, committed)
	})

	// This test illustrates a scenario where the commitment of an E2I rollback message with ID
	// 4 is checked, while the given ID is already present in the list of committed E2I rollback
	// messages on the BridgeStorage smart contract.
	//
	// Expected: the message should be reported as committed (true).
	t.Run("8", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetCommittedRollbackedE2I", mock.Anything).Return(true)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(4),
			SourceChainID: big.NewInt(1),
			IsRollback:    true,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, true, committed)
	})

	// This test illustrates a scenario where the commitment of an I2E rollback message with ID
	// 4 is checked, while the given ID is not in the list of committed I2E rollback message on
	// the BridgeStorage smart contract.
	//
	// Expected: the message should be reported as uncommitted (false).
	t.Run("9", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetCommittedRollbackedI2E", mock.Anything).Return(false)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(4),
			SourceChainID: big.NewInt(100),
			IsRollback:    true,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, false, committed)
	})

	// This test illustrates a scenario where the commitment of an E2I rollback message with ID
	// 4 is checked, while the given ID is not in the list of committed E2I rollback message on
	// the BridgeStorage smart contract.
	//
	// Expected: the message should be reported as uncommitted (false).
	t.Run("10", func(t *testing.T) {
		bc := &blockchain.BlockchainMock{}
		ss := &systemstate.SystemStateMock{}

		bc.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		bc.On("GetSystemState", mock.Anything).Return(ss)
		ss.On("GetCommittedRollbackedE2I", mock.Anything).Return(false)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			bc,
		)

		msg := &contractsapi.BridgeMessage{
			ID:            big.NewInt(4),
			SourceChainID: big.NewInt(1),
			IsRollback:    true,
		}

		committed, err := bm.isBridgeMessageCommitted(nil, msg)
		require.NoError(t, err)
		require.EqualValues(t, false, committed)
	})
}

func Test_finalizeOrdinaryBridgeMessage(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)

	// This test illustrates a scenario of finalizing a successfully executed ordinary message.
	//
	// Expected: the message should be deleted from the database.
	t.Run("1", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(4),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err := bm.state.insertBridgeMessageEvent(msgEvent, false, nil)
		require.NoError(t, err)

		msg := &contractsapi.BridgeMessage{
			ID:                 big.NewInt(4),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err = bm.finalizeOrdinaryBridgeMessage(msg, true, nil)
		require.NoError(t, err)

		msgEvent, err = bm.state.getBridgeMessageEvent(
			msg.ID,
			msg.SourceChainID,
			msg.DestinationChainID,
			false,
			nil)

		require.NoError(t, err)
		require.Nil(t, msgEvent)

		msgEvent, err = bm.state.getBridgeMessageEvent(
			msg.ID,
			msg.DestinationChainID,
			msg.SourceChainID,
			true,
			nil)

		require.NoError(t, err)
		require.Nil(t, msgEvent)
	})

	// This test illustrates a scenario of finalizing an unsuccessfully executed ordinary message.
	//
	// Expected: the ordinary message should become a rollback one (this is done by removing the
	// message from the bucket reserved for ordinary messages and adding it to the bucket reserved
	// for rollback messages). Note, rollback message has the opposite direction.
	t.Run("2", func(t *testing.T) {
		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
			nil,
			nil,
		)

		msgEvent := &contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(4),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}

		err := bm.state.insertBridgeMessageEvent(msgEvent, false, nil)
		require.NoError(t, err)

		msg := &contractsapi.BridgeMessage{
			ID:                 big.NewInt(4),
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			IsRollback:         true,
		}

		err = bm.finalizeOrdinaryBridgeMessage(msg, false, nil)
		require.NoError(t, err)

		msgEvent, err = bm.state.getBridgeMessageEvent(
			msg.ID,
			msg.SourceChainID,
			msg.DestinationChainID,
			false,
			nil)

		require.NoError(t, err)
		require.Nil(t, msgEvent)

		msgEvent, err = bm.state.getBridgeMessageEvent(
			msg.ID,
			msg.DestinationChainID,
			msg.SourceChainID,
			true,
			nil)

		require.NoError(t, err)
		require.NotNil(t, msgEvent)
	})
}

func Test_handleRetry(t *testing.T) {
	vals := validator.NewTestValidators(t, 5)
	ss := &systemstate.SystemStateMock{}
	ss.On("GetBatchCommitCounter", mock.Anything).Return(big.NewInt(5))

	batch := &PendingBridgeBatch{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           []*contractsapi.BridgeMessage{},
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Threshold:          big.NewInt(0),
			CommitCounter:      big.NewInt(0),
		},
	}

	hash, err := batch.Hash()

	require.NoError(t, err)

	batch.Threshold = big.NewInt(120)
	batch.CommitCounter = big.NewInt(1)

	// Function to check whether the state structures reflect the correct state.
	checkFn := func(bm *bridgeEventManager, retNum, unNum int, removed bool) {
		require.EqualValues(t, retNum, len(bm.retryBatches))

		require.EqualValues(t, retNum, len(bm.pendingRetryBatches))

		require.EqualValues(t, unNum, len(bm.unexecutedBatches))

		_, ok := bm.retryBatches[hash]
		require.EqualValues(t, removed, ok)

		_, ok = bm.pendingRetryBatches[hash]
		require.EqualValues(t, removed, ok)
	}

	// This test illustrates a scenario where the batch being checked has its threshold set at
	// block 120, while the current block on the external chain is at 100.

	// Expected: the batch should not go into retry, as its threshold has not yet "expired".
	t.Run("1", func(t *testing.T) {
		client := &mockJSONRPCClient{}

		client.On("GetBlockByNumber", mock.Anything).Return(&types.Block{
			Header: &types.Header{
				Number: 100,
			},
		})

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: false},
			client,
			nil,
		)

		bm.retryBatches[[32]byte{}] = PendingBridgeBatch{}
		bm.pendingRetryBatches[[32]byte{}] = []*PendingBridgeBatch{}

		bm.unexecutedBatches = append(bm.unexecutedBatches, batch, &PendingBridgeBatch{
			BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
				Threshold: big.NewInt(500),
			},
		})

		require.NoError(t, err)

		bm.handleRetry(ss, nil)

		checkFn(bm, 1, 2, false)
	})

	// This test illustrates a scenario where the batch being checked has its threshold set at
	// block 120, while the current block on the external chain is at 128.

	// Expected: the batch should go into retry, as its threshold has "expired" (120 + 5 + 2 is
	// smaller than 128).
	t.Run("2", func(t *testing.T) {
		client := &mockJSONRPCClient{}

		client.On("GetBlockByNumber", mock.Anything).Return(&types.Block{
			Header: &types.Header{
				Number: 128,
			},
		})

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: false},
			client,
			nil,
		)

		bm.retryBatches[[32]byte{}] = PendingBridgeBatch{}
		bm.pendingRetryBatches[[32]byte{}] = []*PendingBridgeBatch{}

		bm.unexecutedBatches = append(bm.unexecutedBatches, batch, &PendingBridgeBatch{
			BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
				Threshold: big.NewInt(500),
			},
		})

		require.NoError(t, err)

		bm.handleRetry(ss, nil)

		checkFn(bm, 2, 1, true)
	})
}

func TestBridgeEventManager_AddLog_BuildBridgeBatches(t *testing.T) {
	t.Parallel()

	vals := validator.NewTestValidators(t, 5)

	t.Run("Node is a validator", func(t *testing.T) {
		t.Skip()
		t.Parallel()

		blockchain := new(blockchain.BlockchainMock)

		blockchain.On("CurrentHeader").Return(&types.Header{Number: 10})
		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil, blockchain)

		postBlockRequest := &oracle.PostBlockRequest{
			FullBlock: &types.FullBlock{
				Block: &types.Block{
					Header: &types.Header{Number: 0}}}}

		bridgeMsg := &contractsapi.BridgeMsgEvent{ID: bigZero, SourceChainID: big.NewInt(1), DestinationChainID: bigZero}
		bridgeMsgData, err := bridgeMsg.Encode()
		require.NoError(t, err)

		// log with the bridge message topic but incorrect content
		require.Error(t, s.AddLog(big.NewInt(1), &ethgo.Log{Topics: []ethgo.Hash{bridgeMessageEventSig}, Data: bridgeMsgData}))
		bridgeEvents, err := s.state.list(100)

		require.NoError(t, err)
		require.Len(t, bridgeEvents, 0)

		// correct event log
		data, err := abi.MustNewType("tuple(uint256 a, uint256 b, string c)").Encode([]string{"1", "100", "data"})
		require.NoError(t, err)

		goodLog := &ethgo.Log{
			Topics: []ethgo.Hash{
				bridgeMessageEventSig,
				ethgo.BytesToHash([]byte{0x1}), // bridge message index 1
				ethgo.ZeroHash,
				ethgo.ZeroHash,
			},
			Data: data,
		}

		require.NoError(t, s.AddLog(big.NewInt(1), goodLog))

		require.NoError(t, s.PostBlock(postBlockRequest))

		length := len(s.pendingBridgeBatchesE2I[0].Messages)

		bridgeMessages, _, err := s.state.getBridgeMessages(1, 1, 1, 100, nil, nil)
		require.NoError(t, err)
		require.Len(t, bridgeMessages, 1)
		require.Len(t, s.pendingBridgeBatchesE2I, 1)
		require.Equal(t, uint64(1), s.pendingBridgeBatchesE2I[0].Messages[0].ID.Uint64())
		require.Equal(t, uint64(1), s.pendingBridgeBatchesE2I[0].Messages[length-1].ID.Uint64())

		// add one more log to have a minimum batch
		goodLog2 := goodLog.Copy()
		goodLog2.Topics[1] = ethgo.BytesToHash([]byte{0x2}) // bridgeMsg event index 1
		require.NoError(t, s.AddLog(big.NewInt(1), goodLog2))

		require.NoError(t, s.PostBlock(postBlockRequest))

		length = len(s.pendingBridgeBatchesE2I[1].Messages)

		require.Len(t, s.pendingBridgeBatchesE2I, 2)
		require.Equal(t, uint64(1), s.pendingBridgeBatchesE2I[1].Messages[0].ID.Uint64())
		require.Equal(t, uint64(2), s.pendingBridgeBatchesE2I[1].Messages[length-1].ID.Uint64())

		// add two more logs to have larger batch
		goodLog3 := goodLog.Copy()
		goodLog3.Topics[1] = ethgo.BytesToHash([]byte{0x3}) // bridgeMsg event index 2
		require.NoError(t, s.AddLog(big.NewInt(1), goodLog3))

		require.NoError(t, s.PostBlock(postBlockRequest))

		goodLog4 := goodLog.Copy()
		goodLog4.Topics[1] = ethgo.BytesToHash([]byte{0x4}) // bridgeMsg event index 3
		require.NoError(t, s.AddLog(big.NewInt(1), goodLog4))

		require.NoError(t, s.PostBlock(postBlockRequest))

		length = len(s.pendingBridgeBatchesE2I[3].Messages)

		require.Len(t, s.pendingBridgeBatchesE2I, 4)
		require.Equal(t, uint64(1), s.pendingBridgeBatchesE2I[3].Messages[0].ID.Uint64())
		require.Equal(t, uint64(4), s.pendingBridgeBatchesE2I[3].Messages[length-1].ID.Uint64())
	})

	t.Run("Node is not a validator", func(t *testing.T) {
		t.Parallel()

		sysState := new(systemstate.SystemStateMock)
		sysState.On("GetNextCommittedIndex").Return(uint64(1000))

		blockchain := new(blockchain.BlockchainMock)
		blockchain.On("GetStateProviderForBlock", mock.Anything).Return(nil)
		blockchain.On("GetSystemState", mock.Anything).Return(sysState)
		blockchain.On("CurrentHeader", mock.Anything).Return(&types.Header{Number: 10})

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: false}, nil, blockchain)

		// correct event log
		data, err := abi.MustNewType("tuple(uint256 a, uint256 b, string c)").Encode([]string{"1", "100", "data"})
		require.NoError(t, err)

		var bridgeMessageEvent contractsapi.BridgeMsgEvent

		goodLog := &ethgo.Log{
			Topics: []ethgo.Hash{
				bridgeMessageEvent.Sig(),
				ethgo.BytesToHash([]byte{0x1}), // bridge message index 0
				ethgo.ZeroHash,
				ethgo.ZeroHash,
			},
			Data: data,
		}

		require.NoError(t, s.AddLog(big.NewInt(1), goodLog))

		// node should have inserted given bridgeMsg event, but it shouldn't build any batch
		bridgeMessages, _, err := s.state.getBridgeMessages(1, 1, 1, 100, nil, nil)
		require.NoError(t, err)
		require.Len(t, bridgeMessages, 1)
		require.Equal(t, uint64(1), bridgeMessages[0].ID.Uint64())
		require.Len(t, s.pendingBridgeBatchesE2I, 0)
	})
}

func createTestLogForBridgeMessageResultEvent(t *testing.T, bridgeMessageEventID uint64) *ethgo.Log {
	t.Helper()

	data, err := abi.MustNewType("tuple(uint256 a, uint256 b, bool c, bytes d)").Encode(
		[]interface{}{big.NewInt(1), big.NewInt(100), false, []byte("")})
	require.NoError(t, err)

	return &ethgo.Log{
		Topics: []ethgo.Hash{
			bridgeMessageResultEventSig,
			ethgo.BytesToHash(common.EncodeUint64ToBytes(bridgeMessageEventID)),
			ethgo.BytesToHash(common.EncodeUint64ToBytes(1)),
		},
		Data: data,
	}
}

type mockTopic struct {
	published proto.Message
}

func (m *mockTopic) consume() proto.Message {
	msg := m.published

	if m.published != nil {
		m.published = nil
	}

	return msg
}

func (m *mockTopic) Publish(obj proto.Message) error {
	m.published = obj

	return nil
}

func (m *mockTopic) Subscribe(handler func(obj interface{}, from peer.ID)) error {
	return nil
}

type mockMsg struct {
	hash  []byte
	epoch uint64
}

func newMockMsg() *mockMsg {
	hash := make([]byte, 32)
	rand.Read(hash)

	return &mockMsg{hash: hash}
}

func (m *mockMsg) WithHash(hash []byte) *mockMsg {
	m.hash = hash

	return m
}

func (m *mockMsg) sign(val *validator.TestValidator, domain []byte) (*BridgeBatchVote, error) {
	signature, err := val.MustSign(m.hash, domain).Marshal()
	if err != nil {
		return nil, err
	}

	return &BridgeBatchVote{
		Hash: m.hash,
		BridgeBatchVoteConsensusData: &BridgeBatchVoteConsensusData{
			Signature: signature,
			Sender:    val.Address().String(),
		},
		EpochNumber: m.epoch,
	}, nil
}

type mockRuntime struct {
	isActiveValidator bool
}

func (m *mockRuntime) IsActiveValidator() bool {
	return m.isActiveValidator
}

type mockJSONRPCClient struct {
	mock.Mock
}

func (m *mockJSONRPCClient) GetBlockByNumber(num jsonrpc.BlockNumber, full bool) (*types.Block, error) {
	block, _ := m.Called(num)[0].(*types.Block)

	return block, nil
}

var _ BridgeManager = (*mockBridgeManager)(nil)

type mockBridgeManager struct {
	chainID uint64
	state   *BridgeManagerStore
}

func (*mockBridgeManager) AddLog(chainID *big.Int, eventLog *ethgo.Log) error {
	return nil
}

func (*mockBridgeManager) Close() {}

func (*mockBridgeManager) GetLogFilters() map[types.Address][]types.Hash {
	return nil
}

// PostBlock implements BridgeManager.
func (*mockBridgeManager) PostBlock(req *oracle.PostBlockRequest) error {
	return nil
}

// PostEpoch implements BridgeManager.
func (mbm *mockBridgeManager) PostEpoch(req *oracle.PostEpochRequest) error {
	if err := mbm.state.insertEpoch(req.NewEpochID, req.DBTx, mbm.chainID); err != nil {
		return err
	}

	return nil
}

// ProcessLog implements BridgeManager.
func (*mockBridgeManager) ProcessLog(header *types.Header, log *ethgo.Log, dbTx *bolt.Tx) error {
	return nil
}

// Start implements BridgeManager.
func (*mockBridgeManager) Start(runtimeCfg *config.Runtime) error {
	return nil
}

func (mbm *mockBridgeManager) BuildExitEventRoot(epoch uint64) (types.Hash, error) {
	return types.ZeroHash, nil
}
func (mbm *mockBridgeManager) BridgeBatch(pendingBlockNumber uint64) ([]*BridgeBatchSigned, error) {
	return nil, nil
}

func (*mockBridgeManager) GetInternalGatewayAddr() types.Address { return [20]byte{} }
