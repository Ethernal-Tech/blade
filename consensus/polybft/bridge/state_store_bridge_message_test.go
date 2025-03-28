package bridge

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	systemstate "github.com/0xPolygon/polygon-edge/consensus/polybft/system_state"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

func TestState_InsertEvent(t *testing.T) {
	t.Parallel()

	state := newTestState(t)
	event := &contractsapi.BridgeMsgEvent{
		ID:                 big.NewInt(1),
		Sender:             types.Address{},
		Receiver:           types.Address{},
		Data:               []byte{},
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
	}

	err := state.insertBridgeMessageEvent(event, false, nil)
	assert.NoError(t, err)

	events, err := state.list(100)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestState_Insert_And_Get_MessageVotes(t *testing.T) {
	t.Parallel()

	state := newTestState(t)
	epoch := uint64(1)
	assert.NoError(t, state.insertEpoch(epoch, nil, 0))

	hash := []byte{1, 2}
	_, err := state.insertConsensusData(1, hash, &BridgeBatchVoteConsensusData{
		Sender:    "NODE_1",
		Signature: []byte{1, 2},
	}, nil, 0)

	assert.NoError(t, err)

	votes, err := state.getMessageVotes(epoch, hash, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(votes))
	assert.Equal(t, "NODE_1", votes[0].Sender)
	assert.True(t, bytes.Equal([]byte{1, 2}, votes[0].Signature))
}

func TestState_getBridgeEventsForBridgeBatch(t *testing.T) {
	t.Parallel()

	state := newTestState(t)

	for i := 1; i <= maxNumberOfBatchEvents; i++ {
		assert.NoError(t, state.insertBridgeMessageEvent(&contractsapi.BridgeMsgEvent{
			ID:                 big.NewInt(int64(i)),
			Data:               []byte{1, 2},
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
		}, false, nil))
	}

	t.Run("Return all - forced. Enough messages", func(t *testing.T) {
		t.Parallel()

		messages, _, err := state.getBridgeMessages(1, maxNumberOfBatchEvents, 100, 1, nil, nil)
		require.NoError(t, err)
		require.Equal(t, maxNumberOfBatchEvents, len(messages))
	})

	t.Run("Return all - forced. Not enough messages", func(t *testing.T) {
		t.Parallel()

		messages, _, err := state.getBridgeMessages(1, 30, 100, 1, nil, nil)
		require.NoError(t, err)
		require.Equal(t, maxNumberOfBatchEvents, len(messages))
	})
}

func TestState_getBridgeBatchForBridgeEvents(t *testing.T) {
	const (
		numOfBridgeBatches = 10
	)

	state := newTestState(t)

	insertTestBridgeBatches(t, state, numOfBridgeBatches)

	var cases = []struct {
		bridgeMessageID uint64
		hasBatch        bool
	}{
		{1, true},
		{10, true},
		{11, true},
		{7, true},
		{999, false},
		{121, false},
		{99, true},
		{101, true},
		{111, false},
		{75, true},
		{5, true},
		{102, true},
		{211, false},
		{21, true},
		{30, true},
		{81, true},
		{90, true},
	}

	for _, c := range cases {
		signedBridgeBatch, err := state.getBridgeBatchForBridgeEvents(c.bridgeMessageID, 1)

		if c.hasBatch {
			require.NoError(t, err, fmt.Sprintf("bridge event %v", c.bridgeMessageID))
			require.Equal(t, c.hasBatch, signedBridgeBatch.ContainsBridgeMessage(c.bridgeMessageID))
		} else {
			require.ErrorIs(t, errNoBridgeBatchForBridgeEvent, err)
		}
	}
}

func TestState_GetNestedBucketInEpoch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		epochNumber uint64
		bucketName  []byte
		errMsg      string
	}{
		{
			name:        "Not existing inner bucket",
			epochNumber: 3,
			bucketName:  []byte("Foo"),
			errMsg:      "could not find Foo bucket for epoch: 3",
		},
		{
			name:        "Happy path",
			epochNumber: 5,
			bucketName:  messageVotesBucket,
			errMsg:      "",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var (
				nestedBucket *bbolt.Bucket
				err          error
			)

			s := newTestState(t)
			require.NoError(t, s.insertEpoch(c.epochNumber, nil, 0))

			err = s.db.View(func(tx *bbolt.Tx) error {
				nestedBucket, err = getNestedBucketInEpoch(tx, c.epochNumber, c.bucketName, 0)

				return err
			})
			if c.errMsg != "" {
				require.ErrorContains(t, err, c.errMsg)
				require.Nil(t, nestedBucket)
			} else {
				require.NoError(t, err)
				require.NotNil(t, nestedBucket)
			}
		})
	}
}

func Test_getBridgeMessages(t *testing.T) {
	insertFn := func(num int, isRollback bool, s *BridgeManagerStore) {
		for i := range num {
			err := s.insertBridgeMessageEvent(&contractsapi.BridgeMsgEvent{
				ID:                 big.NewInt(int64(i + 1)),
				SourceChainID:      big.NewInt(100),
				DestinationChainID: big.NewInt(1),
			}, isRollback, nil)

			require.NoError(t, err)
		}
	}

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, but only 4 ordinary messages (IDs ranging from 1 to 4, incl) exist in the DB
	// (system).
	//
	// Expected: 4 ordinary messages (IDs ranging from 1 to 4, incl) should be retrieved.
	t.Run("1", func(t *testing.T) {
		s := newTestState(t)

		insertFn(4, false, s)

		msgs, ord, err := s.getBridgeMessages(1, 10, 100, 1, nil, nil)

		require.NoError(t, err)
		require.EqualValues(t, 4, len(msgs))
		require.EqualValues(t, 4, ord)

		for i, msg := range msgs {
			require.EqualValues(t, i+1, msg.ID.Uint64())
		}
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, but there is no message in the DB (system).
	//
	// Expected: no message should be retrieved.
	t.Run("2", func(t *testing.T) {
		s := newTestState(t)

		msgs, _, err := s.getBridgeMessages(1, 10, 100, 1, nil, nil)

		require.NoError(t, err)
		require.EqualValues(t, 0, len(msgs))
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 3, are
	// requested, but only 4 ordinary messages (IDs ranging from 1 to 4, incl) exist in the DB
	// (system).
	//
	// Expected: 2 ordinary messages (IDs ranging from 3 to 4, incl) should be retrieved.
	t.Run("3", func(t *testing.T) {
		s := newTestState(t)

		insertFn(4, false, s)

		msgs, ord, err := s.getBridgeMessages(3, 10, 100, 1, nil, nil)

		require.NoError(t, err)
		require.EqualValues(t, 2, len(msgs))
		require.EqualValues(t, 2, ord)
		require.EqualValues(t, 3, msgs[0].ID.Uint64())
		require.EqualValues(t, 4, msgs[1].ID.Uint64())
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, but only 4 ordinary messages (IDs ranging from 1 to 4, incl) and 2 rollback
	// uncommitted messages exist in the DB (system).
	//
	// Expected: 4 ordinary (IDs ranging from 1 to 4, incl) and 2 uncommitted rollback messages
	// should be retrieved.
	t.Run("4", func(t *testing.T) {
		s := newTestState(t)

		insertFn(4, false, s)
		insertFn(2, true, s)

		sysState := &systemstate.SystemStateMock{}
		sysState.On("GetCommittedRollbackedI2E", mock.Anything).Return(false)

		msgs, ord, err := s.getBridgeMessages(1, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 6, len(msgs))
		require.EqualValues(t, 4, ord)

		rollbackCounter := 0

		for i, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}

			require.EqualValues(t, i+1, msg.ID.Uint64())
		}

		require.EqualValues(t, 2, rollbackCounter)
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, while the DB contains 12 ordinary messages (IDs ranging from 1 to 12, incl)
	// and 2 uncommitted rollback messages.
	//
	// Expected: 10 ordinary messages (IDs ranging from 1 to 10, incl) should be retrieved.
	t.Run("5", func(t *testing.T) {
		s := newTestState(t)

		insertFn(12, false, s)
		insertFn(2, true, s)

		msgs, ord, err := s.getBridgeMessages(1, 10, 100, 1, nil, nil)

		require.NoError(t, err)
		require.EqualValues(t, 10, len(msgs))
		require.EqualValues(t, 10, ord)

		for i, msg := range msgs {
			require.EqualValues(t, i+1, msg.ID.Uint64())
		}
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 7, are
	// requested, while the DB contains 12 ordinary messages (IDs ranging from 1 to 12, incl)
	// and 2 uncommitted rollback messages.
	//
	// Expected: 6 ordinary (IDs ranging from 7 to 12, incl) and 2 uncommitted rollback messages
	// should be retrieved.
	t.Run("6", func(t *testing.T) {
		s := newTestState(t)

		insertFn(12, false, s)
		insertFn(2, true, s)

		sysState := &systemstate.SystemStateMock{}
		sysState.On("GetCommittedRollbackedI2E", mock.Anything).Return(false)

		msgs, ord, err := s.getBridgeMessages(7, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 8, len(msgs))
		require.EqualValues(t, 6, ord)

		rollbackCounter := 0

		for i, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}

			require.EqualValues(t, 7+i, msg.ID.Uint64())
		}

		require.EqualValues(t, 2, rollbackCounter)
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 7, are
	// requested, while the DB contains 12 ordinary messages (IDs ranging from 1 to 12, incl)
	// and 7 uncommitted rollback messages.
	//
	// Expected: 6 ordinary (IDs ranging from 7 to 12, incl) and 4 uncommitted rollback messages
	// should be retrieved.
	t.Run("7", func(t *testing.T) {
		s := newTestState(t)

		insertFn(12, false, s)
		insertFn(7, true, s)

		sysState := &systemstate.SystemStateMock{}
		sysState.On("GetCommittedRollbackedI2E", mock.Anything).Return(false)

		msgs, ord, err := s.getBridgeMessages(7, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 10, len(msgs))
		require.EqualValues(t, 6, ord)

		rollbackCounter := 0

		for i, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}

			require.EqualValues(t, 7+i, msg.ID.Uint64())
		}

		require.EqualValues(t, 4, rollbackCounter)
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 7, are
	// requested, while the DB does not containt ordinary messages, but contains 4 uncommitted
	// rollback messages.
	//
	// Expected: 4 uncommitted rollback messages should be retrieved.
	t.Run("8", func(t *testing.T) {
		s := newTestState(t)

		insertFn(4, true, s)

		sysState := &systemstate.SystemStateMock{}
		sysState.On("GetCommittedRollbackedI2E", mock.Anything).Return(false)

		msgs, ord, err := s.getBridgeMessages(7, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 4, len(msgs))
		require.EqualValues(t, 0, ord)

		rollbackCounter := 0

		for _, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++
			}
		}

		require.EqualValues(t, 4, rollbackCounter)
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 7, are
	// requested, while the DB does not contain ordinary messages, but contains 12 uncommitted
	// rollback messages.
	//
	// Expected: 10 uncommitted rollback messages should be retrieved.
	t.Run("9", func(t *testing.T) {
		s := newTestState(t)

		insertFn(12, true, s)

		sysState := &systemstate.SystemStateMock{}
		sysState.On("GetCommittedRollbackedI2E", mock.Anything).Return(false)

		msgs, ord, err := s.getBridgeMessages(7, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 10, len(msgs))
		require.EqualValues(t, 0, ord)

		rollbackCounter := 0

		for _, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++
			}
		}

		require.EqualValues(t, 10, rollbackCounter)
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 7, are
	// requested, while the DB does not containt ordinary messages, but contains 3 uncommitted
	// and 2 committed rollback messages.
	//
	// Expected: 3 uncommitted rollback messages should be retrieved, while 2 committed should
	// be removed from the DB.
	t.Run("10", func(t *testing.T) {
		s := newTestState(t)

		insertFn(5, true, s)

		sysState := &systemstate.SystemStateMock{}

		committed := []bool{true, false, false, true, false}

		for i := range 5 {
			sysState.On("GetCommittedRollbackedI2E", big.NewInt(int64(i)+1)).Return(committed[i])
		}

		msgs, ord, err := s.getBridgeMessages(7, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 3, len(msgs))
		require.EqualValues(t, 0, ord)

		rollbackCounter := 0

		for _, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++
			}
		}

		require.EqualValues(t, 3, rollbackCounter)
		require.EqualValues(t, 2, msgs[0].ID.Uint64())
		require.EqualValues(t, 3, msgs[1].ID.Uint64())
		require.EqualValues(t, 5, msgs[2].ID.Uint64())

		msg, err := s.getBridgeMessageEvent(big.NewInt(1), big.NewInt(100), big.NewInt(1), true, nil)
		require.NoError(t, err)
		require.Nil(t, msg)
		msg, err = s.getBridgeMessageEvent(big.NewInt(4), big.NewInt(100), big.NewInt(1), true, nil)
		require.NoError(t, err)
		require.Nil(t, msg)
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, but only 4 ordinary (IDs ranging from 1 to 4, incl) and 5 (3 uncommitted and
	// 2 committed) rollback messages exist in the DB (system).
	//
	// Expected: 4 ordinary (IDs ranging from 1 to 4, incl) and 3 uncommitted rollback messages
	// should be retrieved, while 2 committed rollback messages should be removed from the DB.
	t.Run("11", func(t *testing.T) {
		s := newTestState(t)

		insertFn(4, false, s)
		insertFn(5, true, s)

		sysState := &systemstate.SystemStateMock{}

		committed := []bool{true, false, false, true, false}

		for i := range 5 {
			sysState.On("GetCommittedRollbackedI2E", big.NewInt(int64(i)+1)).Return(committed[i])
		}

		msgs, ord, err := s.getBridgeMessages(1, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 7, len(msgs))
		require.EqualValues(t, 4, ord)

		rollbackCounter := 0

		for i, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}

			require.EqualValues(t, i+1, msg.ID.Uint64())
		}

		require.EqualValues(t, 3, rollbackCounter)
		require.EqualValues(t, 2, msgs[4].ID.Uint64())
		require.EqualValues(t, 3, msgs[5].ID.Uint64())
		require.EqualValues(t, 5, msgs[6].ID.Uint64())

		msg, err := s.getBridgeMessageEvent(big.NewInt(1), big.NewInt(100), big.NewInt(1), true, nil)
		require.NoError(t, err)
		require.Nil(t, msg)
		msg, err = s.getBridgeMessageEvent(big.NewInt(4), big.NewInt(100), big.NewInt(1), true, nil)
		require.NoError(t, err)
		require.Nil(t, msg)
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, while the DB contains 12 ordinary messages (IDs ranging from 1 to 12, incl)
	// and 5 committed rollback messages.
	//
	// Expected: 10 ordinary messages (IDs ranging from 1 to 10, incl) should be retrieved. No
	// committed rollback message will be removed since the algorithm will not even reach them
	// (it will have enough ordinary messages).
	t.Run("12", func(t *testing.T) {
		s := newTestState(t)

		insertFn(10, false, s)
		insertFn(5, true, s)

		sysState := &systemstate.SystemStateMock{}

		msgs, ord, err := s.getBridgeMessages(1, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 10, len(msgs))
		require.EqualValues(t, 10, ord)

		rollbackCounter := 0

		for i, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}

			require.EqualValues(t, i+1, msg.ID.Uint64())
		}

		require.EqualValues(t, 0, rollbackCounter)

		for i := range 5 {
			i := big.NewInt(int64(i + 1))
			msg, err := s.getBridgeMessageEvent((i), big.NewInt(100), big.NewInt(1), true, nil)
			require.NoError(t, err)
			require.NotNil(t, msg)
		}
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, while the DB contains 9 ordinary messages (IDs ranging from 1 to 9, incl) and
	// 5 rollback messages. The first 4 rollback messages are committed, while the last one is
	// uncommitted (messages are sorted by ID in ascending order).
	//
	// Expected: 9 ordinary (IDs ranging from 1 to 9, incl) and 1 uncommitted rollback message
	// (the last one) should be retrieved. The first 4 rollback messages should be removed since
	// they are committed and the algorithm goes through them until it reaches an uncommitted one.
	t.Run("13", func(t *testing.T) {
		s := newTestState(t)

		insertFn(9, false, s)
		insertFn(5, true, s)

		sysState := &systemstate.SystemStateMock{}

		committed := []bool{true, true, true, true, false}

		for i := range 5 {
			sysState.On("GetCommittedRollbackedI2E", big.NewInt(int64(i)+1)).Return(committed[i])
		}

		msgs, ord, err := s.getBridgeMessages(1, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 10, len(msgs))
		require.EqualValues(t, 9, ord)

		rollbackCounter := 0

		for i, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}

			require.EqualValues(t, i+1, msg.ID.Uint64())
		}

		require.EqualValues(t, 1, rollbackCounter)

		for i := range 4 {
			i := big.NewInt(int64(i + 1))
			msg, err := s.getBridgeMessageEvent(i, big.NewInt(100), big.NewInt(1), true, nil)
			require.NoError(t, err)
			require.Nil(t, msg)
		}

		msg, err := s.getBridgeMessageEvent(big.NewInt(5), big.NewInt(100), big.NewInt(1), true, nil)
		require.NoError(t, err)
		require.NotNil(t, msg)
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, while the DB contains 9 ordinary messages (IDs ranging from 1 to 9, incl) and
	// 5 rollback messages. All rollback messages except the 2nd are committed.
	//
	// Expected: 9 ordinary (IDs ranging from 1 to 9, incl) and 1 uncommitted rollback message
	// (the second one) should be retrieved. The first rollback messages should be deleted since
	// it is committed and the algorithm goes through it until it reaches an uncommitted one. On
	// the other hand, other uncommitted rollback messages remain in the DB since the algorithm
	// will not even reach them.
	t.Run("14", func(t *testing.T) {
		s := newTestState(t)

		insertFn(9, false, s)
		insertFn(5, true, s)

		sysState := &systemstate.SystemStateMock{}

		committed := []bool{true, false, true, true, true}

		for i := range 5 {
			sysState.On("GetCommittedRollbackedI2E", big.NewInt(int64(i)+1)).Return(committed[i])
		}

		msgs, ord, err := s.getBridgeMessages(1, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 10, len(msgs))
		require.EqualValues(t, 9, ord)

		rollbackCounter := 0

		for i, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}

			require.EqualValues(t, i+1, msg.ID.Uint64())
		}

		require.EqualValues(t, 1, rollbackCounter)

		for i := range 5 {
			i := big.NewInt(int64(i + 1))
			if i.Cmp(big.NewInt(2)) >= 0 {
				msg, err := s.getBridgeMessageEvent(i, big.NewInt(100), big.NewInt(1), true, nil)
				require.NoError(t, err)
				require.NotNil(t, msg)

				continue
			}

			msg, err := s.getBridgeMessageEvent(i, big.NewInt(100), big.NewInt(1), true, nil)

			require.NoError(t, err)
			require.Nil(t, msg)
		}
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 4, are
	// requested, while the DB contains 12 ordinary messages (IDs ranging from 1 to 12, incl)
	// and 7 rollback messages. All rollback messages except the 2nd and 6th are uncommitted.
	//
	// Expected: 9 ordinary (IDs ranging from 4 to 12, incl) and 1 uncommitted rollback message
	// (the second one) should be retrieved. The first rollback messages should be removed since
	// it is committed and the algorithm goes through it until reaching an uncommitted one. On
	// the other hand, other committed rollback messages remain in the DB since the algorithm
	// will not even reach them.
	t.Run("15", func(t *testing.T) {
		s := newTestState(t)

		insertFn(12, false, s)
		insertFn(7, true, s)

		sysState := &systemstate.SystemStateMock{}

		committed := []bool{true, false, true, true, true, false, true}

		for i := range 7 {
			sysState.On("GetCommittedRollbackedI2E", big.NewInt(int64(i)+1)).Return(committed[i])
		}

		msgs, ord, err := s.getBridgeMessages(4, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 10, len(msgs))
		require.EqualValues(t, 9, ord)

		rollbackCounter := 0

		for i, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}

			require.EqualValues(t, 4+i, msg.ID.Uint64())
		}

		require.EqualValues(t, 1, rollbackCounter)

		for i := range 7 {
			i := big.NewInt(int64(i + 1))
			if i.Cmp(big.NewInt(1)) == 0 {
				msg, err := s.getBridgeMessageEvent(i, big.NewInt(100), big.NewInt(1), true, nil)
				require.NoError(t, err)
				require.Nil(t, msg)

				continue
			}

			msg, err := s.getBridgeMessageEvent(i, big.NewInt(100), big.NewInt(1), true, nil)

			require.NoError(t, err)
			require.NotNil(t, msg)
		}
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 4, are
	// requested, while the DB contains 2 ordinary messages (IDs ranging from 1 to 2, incl) and
	// 12 rollback messages. Every odd rollback message is committed.
	//
	// Expected: 6 uncommitted rollback messages should be retrieved, while the other rollback
	// messages should be removed.
	t.Run("16", func(t *testing.T) {
		s := newTestState(t)

		insertFn(2, false, s)
		insertFn(12, true, s)

		sysState := &systemstate.SystemStateMock{}

		committed := make([]bool, 12)

		for i := range 12 {
			if i%2 == 0 {
				committed[i] = true

				continue
			}

			committed[i] = false
		}

		for i := range 12 {
			sysState.On("GetCommittedRollbackedI2E", big.NewInt(int64(i)+1)).Return(committed[i])
		}

		msgs, ord, err := s.getBridgeMessages(4, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 6, len(msgs))
		require.EqualValues(t, 0, ord)

		rollbackCounter := 0

		for _, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}
		}

		require.EqualValues(t, 6, rollbackCounter)

		for i := range 12 {
			i := big.NewInt(int64(i + 1))
			if i.Uint64()%2 == 0 {
				msg, err := s.getBridgeMessageEvent(i, big.NewInt(100), big.NewInt(1), true, nil)

				require.NoError(t, err)
				require.NotNil(t, msg)

				continue
			}

			msg, err := s.getBridgeMessageEvent(i, big.NewInt(100), big.NewInt(1), true, nil)
			require.NoError(t, err)
			require.Nil(t, msg)
		}
	})

	// This test illustrates a scenario where 10 messages, starting from ordinary index 1, are
	// requested, but only 1 uncommitted rollback message exist in the DB (system).
	//
	// Expected: no messages should be retrieved while the committed rollback message should be
	// removed.
	t.Run("17", func(t *testing.T) {
		s := newTestState(t)

		insertFn(1, true, s)

		sysState := &systemstate.SystemStateMock{}

		sysState.On("GetCommittedRollbackedI2E", big.NewInt(1)).Return(true)

		msgs, ord, err := s.getBridgeMessages(1, 10, 100, 1, sysState, nil)

		require.NoError(t, err)
		require.EqualValues(t, 0, len(msgs))
		require.EqualValues(t, 0, ord)

		rollbackCounter := 0

		for _, msg := range msgs {
			if msg.IsRollback {
				rollbackCounter++

				continue
			}
		}

		require.EqualValues(t, 0, rollbackCounter)

		msg, err := s.getBridgeMessageEvent(big.NewInt(1), big.NewInt(100), big.NewInt(1), true, nil)
		require.NoError(t, err)
		require.Nil(t, msg)
	})
}

func insertTestBridgeBatches(t *testing.T, state *BridgeManagerStore, numberOfBatches uint64) {
	t.Helper()

	for i := uint64(0); i <= numberOfBatches; i++ {
		signedBridgeBatch := CreateTestBridgeBatchMessage(t, 10, 10*i)
		require.NoError(t, state.insertBridgeBatchMessage(signedBridgeBatch, nil))
	}
}
