package bridge

import (
	"math/big"
	"testing"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	systemstate "github.com/0xPolygon/polygon-edge/consensus/polybft/system_state"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_BridgeMessageResult(t *testing.T) {
	s := newTestState(t)

	// This test checks the correctness of the following BridgeManagerStore methods:
	// 1. (*BridgeManagerStore).insertBridgeMessageResultEvent
	// 2. (*BridgeManagerStore).getBridgeMessageResult
	// 3. (*BridgeManagerStore).isBridgeMessageExecuted

	ordinaryMessage := &contractsapi.BridgeMessage{
		ID:                 big.NewInt(20),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
		IsRollback:         false,
	}

	rollbackMessage := &contractsapi.BridgeMessage{
		ID:                 big.NewInt(80),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
		IsRollback:         true,
	}

	ordinaryResult := &contractsapi.BridgeMessageResultEvent{
		ID:                 big.NewInt(20),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
		IsRollback:         false,
	}

	rollbackResult := &contractsapi.BridgeMessageResultEvent{
		ID:                 big.NewInt(80),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
		IsRollback:         true,
	}

	require.NoError(t, s.insertBridgeMessageResultEvent(ordinaryResult, nil))
	require.NoError(t, s.insertBridgeMessageResultEvent(rollbackResult, nil))

	res, err := s.getBridgeMessageResult(ordinaryMessage, nil)
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(20), res.ID)
	require.False(t, res.IsRollback)
	require.True(t, s.isBridgeMessageExecuted(ordinaryMessage, nil))

	res, err = s.getBridgeMessageResult(rollbackMessage, nil)
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(80), res.ID)
	require.True(t, res.IsRollback)
	require.True(t, s.isBridgeMessageExecuted(rollbackMessage, nil))

	require.False(t, s.isBridgeMessageExecuted(&contractsapi.BridgeMessage{
		ID:                 big.NewInt(1),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
		IsRollback:         true,
	}, nil))

	require.False(t, s.isBridgeMessageExecuted(&contractsapi.BridgeMessage{
		ID:                 big.NewInt(1),
		SourceChainID:      big.NewInt(100),
		DestinationChainID: big.NewInt(1),
		IsRollback:         false,
	}, nil))
}

func Test_BridgeMessageEvent(t *testing.T) {
	s := newTestState(t)

	// This test checks the correctness of the following BridgeManagerStore methods:
	// 1. (*BridgeManagerStore).insertBridgeMessageEvent
	// 2. (*BridgeManagerStore).getBridgeMessageEvent
	// 3. (*BridgeManagerStore).removeBridgeMessageEvent
	// 4. (*BridgeManagerStore).isBridgeMessageKnown

	sid := big.NewInt(100)
	did := big.NewInt(1)

	ordinaryEvent := &contractsapi.BridgeMsgEvent{
		ID:                 big.NewInt(20),
		SourceChainID:      sid,
		DestinationChainID: did,
	}

	rollbackEvent := &contractsapi.BridgeMsgEvent{
		ID:                 big.NewInt(80),
		SourceChainID:      sid,
		DestinationChainID: did,
	}

	ordinaryMessage := &contractsapi.BridgeMessage{
		ID:                 big.NewInt(20),
		SourceChainID:      sid,
		DestinationChainID: did,
		IsRollback:         false,
	}

	rollbackMessage := &contractsapi.BridgeMessage{
		ID:                 big.NewInt(80),
		SourceChainID:      sid,
		DestinationChainID: did,
		IsRollback:         true,
	}

	require.NoError(t, s.insertBridgeMessageEvent(ordinaryEvent, false, nil))
	require.NoError(t, s.insertBridgeMessageEvent(rollbackEvent, true, nil))

	res, err := s.getBridgeMessageEvent(big.NewInt(20), sid, did, false, nil)
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(20), res.ID)
	require.True(t, s.isBridgeMessageKnown(ordinaryMessage, nil))
	require.NoError(t, s.removeBridgeMessageEvent(big.NewInt(20), sid, did, false, nil))
	require.False(t, s.isBridgeMessageKnown(ordinaryMessage, nil))

	res, err = s.getBridgeMessageEvent(big.NewInt(80), sid, did, true, nil)
	require.NoError(t, err)
	require.EqualValues(t, big.NewInt(80), res.ID)
	require.True(t, s.isBridgeMessageKnown(rollbackMessage, nil))
	require.NoError(t, s.removeBridgeMessageEvent(big.NewInt(80), sid, did, true, nil))
	require.False(t, s.isBridgeMessageKnown(rollbackMessage, nil))
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

func Test_UnexecutedBatches(t *testing.T) {
	s := newTestState(t)

	// This test checks the correctness of the following BridgeManagerStore methods:
	// 1. (*BridgeManagerStore).insertUnexecutedBatch
	// 2. (*BridgeManagerStore).getUnexecutedBatches
	// 3. (*BridgeManagerStore).removeUnexecutedBatch

	b1 := PendingBridgeBatch{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           []*contractsapi.BridgeMessage{},
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Threshold:          big.NewInt(0),
			CommitCounter:      big.NewInt(0),
		},
	}

	h1, err := b1.Hash()
	require.NoError(t, err)

	b2 := PendingBridgeBatch{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           []*contractsapi.BridgeMessage{},
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(1),
			Threshold:          big.NewInt(0),
			CommitCounter:      big.NewInt(0),
		},
	}

	h2, err := b2.Hash()
	require.NoError(t, err)

	b3 := PendingBridgeBatch{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           []*contractsapi.BridgeMessage{},
			SourceChainID:      big.NewInt(100),
			DestinationChainID: big.NewInt(2),
			Threshold:          big.NewInt(1),
			CommitCounter:      big.NewInt(400),
		},
	}

	h3, err := b3.Hash()
	require.NoError(t, err)

	err = s.insertUnexecutedBatch(1, h1, big.NewInt(12), nil)
	require.NoError(t, err)

	ids, err := s.getUnexecutedBatches(1, nil)

	require.NoError(t, err)
	require.EqualValues(t, 1, len(ids))
	require.EqualValues(t, 12, ids[0])

	err = s.insertUnexecutedBatch(1, h2, big.NewInt(24), nil)
	require.NoError(t, err)

	ids, err = s.getUnexecutedBatches(1, nil)

	require.NoError(t, err)
	require.EqualValues(t, 1, len(ids))
	require.EqualValues(t, 24, ids[0])

	err = s.insertUnexecutedBatch(1, h3, big.NewInt(35), nil)
	require.NoError(t, err)

	ids, err = s.getUnexecutedBatches(1, nil)

	require.NoError(t, err)
	require.EqualValues(t, 2, len(ids))
	require.EqualValues(t, 35, ids[0])
	require.EqualValues(t, 24, ids[1])

	err = s.removeUnexecutedBatch(1, h2, nil)

	require.NoError(t, err)

	ids, err = s.getUnexecutedBatches(1, nil)

	require.NoError(t, err)
	require.EqualValues(t, 1, len(ids))
	require.EqualValues(t, 35, ids[0])
}
