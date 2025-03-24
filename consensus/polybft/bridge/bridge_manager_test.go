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

func newTestBridgeManager(t *testing.T, key *validator.TestValidator, runtime Runtime, blockchain blockchain.Blockchain) *bridgeEventManager {
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
		}, runtime, 1, 100, blockchain)

	s.nextEventIDE2I = 1
	s.nextEventIDI2E = 1

	return s
}

func TestBridgeEventManager_PostEpoch_BuildBridgeBatch(t *testing.T) {
	t.Parallel()

	vals := validator.NewTestValidators(t, 5)

	blockchain := new(blockchain.BlockchainMock)
	blockchain.On("CurrentHeader").Return(&types.Header{Number: 10})

	t.Run("When node is validator", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, blockchain)

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

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: false}, blockchain)

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

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil)

		s.epoch = 1
		msg := &BridgeBatchVote{
			EpochNumber: 0,
		}

		err := s.saveVote(msg)
		require.NoError(t, err)
	})

	t.Run("Sender is not a validator", func(t *testing.T) {
		t.Parallel()

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil)
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

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil)
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

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil)
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

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil)
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

	s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, nil)
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
		Messages:              []*contractsapi.BridgeMessage{&bridgeMessage},
		SourceChainID:         big.NewInt(1),
		DestinationChainID:    big.NewInt(2),
		Threshold:             big.NewInt(10000),
		NumberOfRegularEvents: big.NewInt(1),
		CommitCounter:         big.NewInt(1),
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

	s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, blockchain)
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
		ss.On("GetConfirmedRollbackedI2E", mock.Anything).Return(true)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
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
		ss.On("GetConfirmedRollbackedE2I", mock.Anything).Return(true)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
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
		ss.On("GetConfirmedRollbackedI2E", mock.Anything).Return(false)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
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
		ss.On("GetConfirmedRollbackedE2I", mock.Anything).Return(false)

		bm := newTestBridgeManager(t,
			vals.GetValidator("0"),
			&mockRuntime{isActiveValidator: true},
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

func TestBridgeEventManager_AddLog_BuildBridgeBatches(t *testing.T) {
	t.Parallel()

	vals := validator.NewTestValidators(t, 5)

	t.Run("Node is a validator", func(t *testing.T) {
		t.Skip()
		t.Parallel()

		blockchain := new(blockchain.BlockchainMock)

		blockchain.On("CurrentHeader").Return(&types.Header{Number: 10})
		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: true}, blockchain)

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

		s := newTestBridgeManager(t, vals.GetValidator("0"), &mockRuntime{isActiveValidator: false}, blockchain)

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
