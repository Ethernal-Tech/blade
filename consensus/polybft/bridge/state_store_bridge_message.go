package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	systemstate "github.com/0xPolygon/polygon-edge/consensus/polybft/system_state"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/types"
	bolt "go.etcd.io/bbolt"
)

// The following variables denote the names of all buckets within the bolt DB used to implement
// bridging logic (see newBridgeManagerStore for more details about buckets structure).
var (
	// bridgeMessages (int. name "bridgeMessages") is the root bucket of the tree structure that
	// contains all the necessary buckets for proper coordination of the operations related to
	// the bridge messages, such as handling ordinary and rollback messages, denoting messages
	// as executed, and more.
	bridgeMessages = []byte("bridgeMessages")

	// ordinaryMessages (int. name "ordinary") is a bucket that contains all ordinary messages
	// that need to be transferred to the destination chain and executed there. This bucket is
	// part of the bucket tree structure that starts with the "bridgeMessages" bucket. It is at
	// the same level in the tree as the "rollbackMessages" and "executedMessages" buckets. The
	// message is written to the bucket when a `BridgeMsg` event occurs on the source chain and
	// is removed upon its execution (successful or unsuccessful) on the destination chain, i.e.
	// when a `BridgeMessageResult` event occurs. See BridgeManager ProcessLog/AddLog methods for
	// more details on when these events are triggered. The key in the bucket represents the ID
	// of the message, while the value is the content of the `BridgeMsg` event (more precisely,
	// its struct representation in Go). Note, it is also used within "executedMessages" bucket
	// (read doc for "executedMessages" for more info).
	ordinaryMessages = []byte("ordinary")

	// rollbackMessages (int. name "rollback") is a bucket that contains all rollback messages
	// that need to be transferred to the destination chain and executed there. This bucket is
	// part of the bucket tree structure that starts with the "bridgeMessages" bucket. It is at
	// the same level in the tree as the "ordinaryMessages" and "executedMessages" buckets. If
	// the ordinary message fails to execute on the destination chain (incidated by the emission
	// of a `BridgeMessageResult` event with the status field set to "false"), then it becomes
	// a rollback message and is written to this bucket. A message is removed from the bucket
	// once it is committed to the BridgeStorage. The key in the bucket represents the message
	// ID, while the value is the content of the `BridgeMsg` event (more precisely, its struct
	// representation in Go) - the same as in the "ordinaryMessages" bucket. Note, it is also
	// used within "executedMessages" bucket (read doc for "executedMessages" for more info).
	rollbackMessages = []byte("rollback")

	// executedMessages (int. name "executed") is a bucket that contains results of all messages
	// that have been executed (successfully or unsuccessfully) on the destination chain. This
	// bucket is part of the bucket structure that starts with the "bridgeMessages" bucket. It
	// is at the same level in the tree as the "ordinaryMessages" and "rollbackMessages" buckets.
	// The bucket consists of two sub-buckets: "ordinaryMessages" and "rollbackMessages". These
	// names are only used for categorization and are not related to the previously described
	// buckets with the same names. The execution result of a message is written to the bucket
	// upon its execution on the destination chain, that is, when a `BridgeMessageResult` event
	// occurs. If the message was an ordinary message, its execution result is stored in the
	// "ordinaryMessages" sub-bucket. If it was a rollback message, the result is stored in the
	// "rollbackMessages" sub-bucket. The content of these buckets is never deleted. The key in
	// the bucket (or more precisely, in the sub-buckets) represents the message ID, while the
	// value stores the contents of the `BridgeMessageResult` event (more precisely, its struct
	// representation in Go).
	executedMessages = []byte("executed")

	// unexecutedBatches (int. name "unexecuted") is a bucket that contains all I2E batches that
	// have not yet been executed on the external chain. A batch is written to this bucket upon
	// commit to BridgeStorage (i.e., when a `NewBatch` event occurs), and removed once all its
	// messages have been executed on the external chain. The key in this bucket represents the
	// base hash of the batch, while the value is the serial number of the committed batch. Both
	// of them, the base hash and the serial number, can be used to uniquely identify any batch,
	// but on different levels (since base hash doesn't include threshold and commit counter, it
	// is the same for the original batch and all its retry versions).
	//
	// Base hash represents a hash of three concatenated encoded batch metadata fields:
	// 1. source chain ID,
	// 2. destination chain ID,
	// 3. list of message IDs being bridged (note: in current implementation, complete messages
	// are used instead of IDs).
	unexecutedBatches = []byte("unexecuted")
)

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

// BridgeManagerStore is a wrapper around boltDB that provides all the necessary methods for the
// bridging "cold" storage logic.
type BridgeManagerStore struct {
	db               *bolt.DB
	externalChainIDs []uint64
	internalChainID  uint64
}

// newBridgeManagerStore takes an instance of the bolt database and creates a complete bucket
// tree structure within it necessary for the proper functioning of the bridging process. For
// a detailed description of this structure, see the comment within the function itself. The
// IDs of all external chains to which the internal (Blade) chain realizes bridging should be
// provided through the externalChainsIDs argument.
func newBridgeManagerStore(
	db *bolt.DB,
	externalChainsIDs []uint64,
	internalChainID uint64,
	dbTx *bolt.Tx) (*BridgeManagerStore, error) {
	// The bucket tree structure intended for manipulating messages is organized as follows (by
	// levels):
	//
	// 1. Root level ("bridgeMessages" bucket)
	//
	// 2. Source chain level:
	//    - Since each chain can be source of the bridging, we create one bucket for each chain
	//      participating in the bridging process.
	//    - Bucket names are the chain IDs.
	//    - Example: With internal (Blade) chain ID 100 and external chain IDs 1, 2, 3, we will
	//      have four buckets named "100", "1", "2", and "3", respectively.
	//    - In the context of a bridge message, this represents the ID of the chain from which
	//      the message originates.
	//
	// 3. Destination chain level:
	//    - If the internal chain is at level 2, we create a bucket for each external chain.
	//    - If the external chain is at level 2, we create a single bucket for internal chain.
	//    - This structure is logical since the internal chain can bridge to any external chain,
	//      but external chains can only bridge to the internal chain.
	//    - Example: With internal (Blade) chain at level 2, we will have three buckets named
	//      "1", "2" and "3" at this level. With external chain at level 2, we will have only
	//      one bucket named "100" at this level.
	//    - In the context of a bridge message, this represents the ID of the chain to which the
	//      message is going (that is, to which the message is being relayed).
	//
	// 4. Message state level:
	//    - Each bucket from level 3 contains three buckets:
	//      1. "ordinaryMessages",
	//      2. "rollbackMessages", and
	//      3. "executedMessages"
	//
	// 5. Executed messages sublevel:
	//    - The "executedMessages" bucket from level 4 contains two additional buckets:
	//      1. "ordinaryMessages", and
	//      2. "rollbackMessages"
	initFn := func(tx *bolt.Tx) error {
		// First (root) level bucket.
		bridgeMessagesBucket, err := tx.CreateBucketIfNotExists(bridgeMessages)
		if err != nil {
			return fmt.Errorf("failed to create root bucket for bridge messages: %w", err)
		}

		unexecutedBatchesBucket, err := tx.CreateBucketIfNotExists(unexecutedBatches)
		if err != nil {
			return fmt.Errorf("failed to create root bucket for unexecuted batches: %w", err)
		}

		internalIDBytes := common.EncodeUint64ToBytes(internalChainID)

		// Second (source chain) level bucket.
		internalChainBucket, err := bridgeMessagesBucket.CreateBucketIfNotExists(internalIDBytes)
		if err != nil {
			return fmt.Errorf("failed to create (source) internal chain bucket: %w", err)
		}

		// Function to create buckets for ordinary and rollback messages on the fourth and fifth
		// level within bucket tree structure.
		createOrdinaryAndRollbackBuckets := func(bucket *bolt.Bucket) error {
			if _, err := bucket.CreateBucketIfNotExists(ordinaryMessages); err != nil {
				return fmt.Errorf("failed to create bucket for ordinary messages: %w", err)
			}

			if _, err := bucket.CreateBucketIfNotExists(rollbackMessages); err != nil {
				return fmt.Errorf("failed to create bucket for rollback messages: %w", err)
			}

			return nil
		}

		for _, externalChainID := range externalChainsIDs {
			id := common.EncodeUint64ToBytes(externalChainID)

			// Second (source chain) level bucket.
			externalChainBucket, err := bridgeMessagesBucket.CreateBucketIfNotExists(id)
			if err != nil {
				return fmt.Errorf("failed to create (source) external chain (%v) bucket: %w",
					externalChainID,
					err)
			}

			if _, err = unexecutedBatchesBucket.CreateBucketIfNotExists(id); err != nil {
				return fmt.Errorf("failed to create unexecuted batches bucket for chain %v: %w",
					externalChainID,
					err)
			}

			// Third (destination chain) level buckets.
			var buckets [2]*bolt.Bucket

			buckets[0], err = externalChainBucket.CreateBucketIfNotExists(internalIDBytes)
			if err != nil {
				return fmt.Errorf("failed to create (destination) internal chain bucket: %w", err)
			}

			buckets[1], err = internalChainBucket.CreateBucketIfNotExists(id)
			if err != nil {
				return fmt.Errorf("failed to create (destination) external chain (%v) bucket: %w",
					externalChainID,
					err)
			}

			for _, bucket := range buckets {
				// Fourth (message state) level bucket.
				if err := createOrdinaryAndRollbackBuckets(bucket); err != nil {
					return err
				}

				// Fourth (message state) level bucket.
				executedMessagesBucket, err := bucket.CreateBucketIfNotExists(executedMessages)
				if err != nil {
					return fmt.Errorf("failed to create bucket for executed messages: %w", err)
				}

				// Fifth (executed messages) level bucket.
				if err := createOrdinaryAndRollbackBuckets(executedMessagesBucket); err != nil {
					return err
				}
			}
		}

		return nil
	}

	var (
		err   error
		store *BridgeManagerStore
	)

	if dbTx == nil {
		err = db.Update(initFn)
	} else {
		err = initFn(dbTx)
	}

	if err == nil {
		store = &BridgeManagerStore{
			db:               db,
			externalChainIDs: externalChainsIDs,
			internalChainID:  internalChainID,
		}
	}

	return store, err
}

func (bms *BridgeManagerStore) beginDBTransaction(isWriteTx bool) (*bolt.Tx, error) {
	return bms.db.Begin(isWriteTx)
}

func (bms *BridgeManagerStore) insertBridgeMessageEvent(
	event *contractsapi.BridgeMsgEvent,
	isRollback bool,
	dbTx *bolt.Tx) error {
	insertFn := func(tx *bolt.Tx) error {
		raw, err := json.Marshal(event)
		if err != nil {
			return err
		}

		bucket := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(event.SourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(event.DestinationChainID.Uint64()))

		if isRollback {
			return bucket.Bucket(rollbackMessages).
				Put(common.EncodeUint64ToBytes(event.ID.Uint64()), raw)
		}

		return bucket.Bucket(ordinaryMessages).
			Put(common.EncodeUint64ToBytes(event.ID.Uint64()), raw)
	}

	if dbTx == nil {
		return bms.db.Update(insertFn)
	}

	return insertFn(dbTx)
}

func (bms *BridgeManagerStore) removeBridgeMessageEvent(
	messageID,
	sourceChainID,
	destinationChainID *big.Int,
	isRollback bool,
	dbTx *bolt.Tx) error {
	removeFn := func(tx *bolt.Tx) error {
		id := common.EncodeUint64ToBytes(messageID.Uint64())

		bucket := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(sourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(destinationChainID.Uint64()))

		if isRollback {
			return bucket.Bucket(rollbackMessages).Delete(id)
		}

		return bucket.Bucket(ordinaryMessages).Delete(id)
	}

	if dbTx == nil {
		return bms.db.Update(removeFn)
	}

	return removeFn(dbTx)
}

func (bms *BridgeManagerStore) getBridgeMessageEvent(
	messageID,
	sourceChainID,
	destinationChainID *big.Int,
	isRollback bool,
	dbTx *bolt.Tx) (*contractsapi.BridgeMsgEvent, error) {
	var message *contractsapi.BridgeMsgEvent

	getFn := func(tx *bolt.Tx) error {
		id := common.EncodeUint64ToBytes(messageID.Uint64())

		bucket := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(sourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(destinationChainID.Uint64()))

		var data []byte

		if isRollback {
			data = bucket.Bucket(rollbackMessages).Get(id)
		} else {
			data = bucket.Bucket(ordinaryMessages).Get(id)
		}

		if data == nil {
			return nil
		}

		if err := json.Unmarshal(data, &message); err != nil {
			return err
		}

		return nil
	}

	if dbTx == nil {
		if err := bms.db.View(getFn); err != nil {
			return nil, err
		}

		return message, nil
	}

	if err := getFn(dbTx); err != nil {
		return nil, err
	}

	return message, nil
}

func (bms *BridgeManagerStore) isBridgeMessageKnown(
	message *contractsapi.BridgeMessage,
	dbTx *bolt.Tx) bool {
	var known bool

	getFn := func(tx *bolt.Tx) error {
		id := common.EncodeUint64ToBytes(message.ID.Uint64())

		bucket := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(message.SourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(message.DestinationChainID.Uint64()))

		var data []byte

		if message.IsRollback {
			data = bucket.Bucket(rollbackMessages).Get(id)
		} else {
			data = bucket.Bucket(ordinaryMessages).Get(id)
		}

		if data != nil {
			known = true
		}

		return nil
	}

	if dbTx == nil {
		bms.db.View(getFn) //nolint:errcheck
	} else {
		getFn(dbTx) //nolint:errcheck
	}

	return known
}

func (bms *BridgeManagerStore) isBridgeMessageExecuted(
	message *contractsapi.BridgeMessage,
	dbTx *bolt.Tx) bool {
	var executed bool

	getFn := func(tx *bolt.Tx) error {
		id := common.EncodeUint64ToBytes(message.ID.Uint64())

		bucket := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(message.SourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(message.DestinationChainID.Uint64())).
			Bucket(executedMessages)

		var data []byte

		if message.IsRollback {
			data = bucket.Bucket(rollbackMessages).Get(id)
		} else {
			data = bucket.Bucket(ordinaryMessages).Get(id)
		}

		if data != nil {
			executed = true
		}

		return nil
	}

	if dbTx == nil {
		bms.db.View(getFn) //nolint:errcheck
	} else {
		getFn(dbTx) //nolint:errcheck
	}

	return executed
}

func (bms *BridgeManagerStore) getUnexecutedBatches(
	externalChainID uint64,
	dbTx *bolt.Tx) ([]uint64, error) {
	var ids []uint64

	getFn := func(tx *bolt.Tx) error {
		id := common.EncodeUint64ToBytes(externalChainID)

		err := tx.Bucket(unexecutedBatches).Bucket(id).ForEach(func(k, v []byte) error {
			var id uint64

			if err := json.Unmarshal(v, &id); err != nil {
				ids = nil

				return err
			}

			ids = append(ids, id)

			return nil
		})

		return err
	}

	if dbTx == nil {
		return ids, bms.db.View(getFn)
	} else {
		return ids, getFn(dbTx)
	}
}

func (bms *BridgeManagerStore) insertUnexecutedBatch(
	externalChainID uint64,
	baseHash types.Hash,
	batchID *big.Int,
	dbTx *bolt.Tx) error {
	insertFn := func(tx *bolt.Tx) error {
		raw, err := json.Marshal(batchID)
		if err != nil {
			return err
		}

		return tx.Bucket(unexecutedBatches).
			Bucket(common.EncodeUint64ToBytes(externalChainID)).
			Put(baseHash.Bytes(), raw)
	}

	if dbTx == nil {
		return bms.db.Update(insertFn)
	}

	return insertFn(dbTx)
}

func (bms *BridgeManagerStore) removeUnexecutedBatch(
	externalChainID uint64,
	baseHash types.Hash,
	dbTx *bolt.Tx) error {
	removeFn := func(tx *bolt.Tx) error {
		return tx.Bucket(unexecutedBatches).
			Bucket(common.EncodeUint64ToBytes(externalChainID)).
			Delete(baseHash.Bytes())
	}

	if dbTx == nil {
		return bms.db.Update(removeFn)
	}

	return removeFn(dbTx)
}

func (bms *BridgeManagerStore) insertBridgeMessageResultEvent(
	result *contractsapi.BridgeMessageResultEvent,
	dbTx *bolt.Tx) error {
	insertFn := func(tx *bolt.Tx) error {
		raw, err := json.Marshal(result)
		if err != nil {
			return err
		}

		bucket := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(result.SourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(result.DestinationChainID.Uint64())).
			Bucket(executedMessages)

		if result.IsRollback {
			return bucket.Bucket(rollbackMessages).
				Put(common.EncodeUint64ToBytes(result.ID.Uint64()), raw)
		}

		return bucket.Bucket(ordinaryMessages).
			Put(common.EncodeUint64ToBytes(result.ID.Uint64()), raw)
	}

	if dbTx == nil {
		return bms.db.Update(insertFn)
	}

	return insertFn(dbTx)
}

func (bms *BridgeManagerStore) getBridgeMessageResult(
	message *contractsapi.BridgeMessage,
	dbTx *bolt.Tx) (*contractsapi.BridgeMessageResultEvent, error) {
	var result *contractsapi.BridgeMessageResultEvent

	getFn := func(tx *bolt.Tx) error {
		id := common.EncodeUint64ToBytes(message.ID.Uint64())

		bucket := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(message.SourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(message.DestinationChainID.Uint64())).
			Bucket(executedMessages)

		var data []byte

		if message.IsRollback {
			data = bucket.Bucket(rollbackMessages).Get(id)
		} else {
			data = bucket.Bucket(ordinaryMessages).Get(id)
		}

		if data == nil {
			return nil
		}

		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}

		return nil
	}

	if dbTx == nil {
		if err := bms.db.View(getFn); err != nil {
			return nil, err
		}

		return result, nil
	}

	if err := getFn(dbTx); err != nil {
		return nil, err
	}

	return result, nil
}

func (bms *BridgeManagerStore) getBridgeMessages(
	fromIndex,
	limit,
	sid,
	did uint64,
	sysState systemstate.SystemState,
	dbTx *bolt.Tx) ([]*contractsapi.BridgeMessage, uint64, error) {
	if limit == 0 {
		return nil, 0, nil
	}

	var (
		messages          []*contractsapi.BridgeMessage
		err               error
		limitReachedErr   = errors.New("limit reached")
		numOfOrdinaryMsgs uint64
	)

	getFn := func(tx *bolt.Tx) error {
		var taken uint64

		bucket := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(sid)).
			Bucket(common.EncodeUint64ToBytes(did)).
			Bucket(ordinaryMessages)

		for i := fromIndex; i < fromIndex+limit; i++ {
			v := bucket.Get(common.EncodeUint64ToBytes(i))

			if v == nil {
				break
			}

			var event *contractsapi.BridgeMsgEvent
			if err := json.Unmarshal(v, &event); err != nil {
				return err
			}

			message := &contractsapi.BridgeMessage{
				ID:                 event.ID,
				SourceChainID:      event.SourceChainID,
				DestinationChainID: event.DestinationChainID,
				Sender:             event.Sender,
				Receiver:           event.Receiver,
				Payload:            event.Data,
				IsRollback:         false,
			}

			messages = append(messages, message)

			taken++
		}

		numOfOrdinaryMsgs = taken

		if taken == limit {
			return nil
		}

		var toRemove []*big.Int

		err := tx.Bucket(bridgeMessages).
			Bucket(common.EncodeUint64ToBytes(sid)).
			Bucket(common.EncodeUint64ToBytes(did)).
			Bucket(rollbackMessages).ForEach(func(k, v []byte) error {
			var event *contractsapi.BridgeMsgEvent
			if err := json.Unmarshal(v, &event); err != nil {
				return err
			}

			var (
				committed bool
				err       error
			)

			if bms.internalChainID == sid {
				committed, err = sysState.GetCommittedRollbackedI2E(did, event.ID)
			} else {
				committed, err = sysState.GetCommittedRollbackedE2I(sid, event.ID)
			}

			if err != nil {
				return err
			}

			if committed {
				toRemove = append(toRemove, event.ID)

				return nil
			}

			message := &contractsapi.BridgeMessage{
				ID:                 event.ID,
				SourceChainID:      event.SourceChainID,
				DestinationChainID: event.DestinationChainID,
				Sender:             event.Sender,
				Receiver:           event.Receiver,
				Payload:            event.Data,
				IsRollback:         true,
			}

			messages = append(messages, message)

			taken++

			if taken == limit {
				return limitReachedErr
			}

			return nil
		})

		for _, id := range toRemove {
			id := common.EncodeUint64ToBytes(id.Uint64())

			if err := tx.Bucket(bridgeMessages).
				Bucket(common.EncodeUint64ToBytes(sid)).
				Bucket(common.EncodeUint64ToBytes(did)).
				Bucket(rollbackMessages).Delete(id); err != nil {
				return err
			}
		}

		if err != nil {
			if errors.Is(err, limitReachedErr) {
				return nil
			} else {
				return err
			}
		}

		return nil
	}

	if dbTx == nil {
		err = bms.db.Update(getFn)
	} else {
		err = getFn(dbTx)
	}

	return messages, numOfOrdinaryMsgs, err
}
