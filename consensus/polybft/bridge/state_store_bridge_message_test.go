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

func TestState_getBridgeMessages_Mix(t *testing.T) {
	t.Parallel()

	state := newTestState(t)

	for i := 1; i <= 10; i++ {
		if i <= 7 {
			assert.NoError(t, state.insertBridgeMessageEvent(&contractsapi.BridgeMsgEvent{
				ID:                 big.NewInt(int64(i)),
				Data:               []byte{1, 2},
				SourceChainID:      big.NewInt(100),
				DestinationChainID: big.NewInt(1),
			}, false, nil))
		} else {
			assert.NoError(t, state.insertBridgeMessageEvent(&contractsapi.BridgeMsgEvent{
				ID:                 big.NewInt(int64(i)),
				Data:               []byte{1, 2},
				SourceChainID:      big.NewInt(100),
				DestinationChainID: big.NewInt(1),
			}, true, nil))
		}
	}

	sysState := &systemstate.SystemStateMock{}
	sysState.On("GetConfirmedRollbackedE2I").Return(false)

	messages, numOfOrdinary, err := state.getBridgeMessages(1, maxNumberOfBatchEvents, 100, 1, sysState, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(7), numOfOrdinary)
	assert.Equal(t, 10, len(messages))
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

func insertTestBridgeBatches(t *testing.T, state *BridgeManagerStore, numberOfBatches uint64) {
	t.Helper()

	for i := uint64(0); i <= numberOfBatches; i++ {
		signedBridgeBatch := CreateTestBridgeBatchMessage(t, 10, 10*i)
		require.NoError(t, state.insertBridgeBatchMessage(signedBridgeBatch, nil))
	}
}
