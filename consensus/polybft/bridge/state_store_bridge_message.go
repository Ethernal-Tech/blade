package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/helper/common"
	bolt "go.etcd.io/bbolt"
)

var (
	// bucket to store bridge events
	bridgeMessageEventsBucket = []byte("bridgeMessageEvents")
	// bucket to store bridge buckets
	bridgeBatchBucket = []byte("bridgeBatches")
	// bucket to store message votes (signatures)
	messageVotesBucket = []byte("votes")
	// bucket to store epochs and all its nested buckets (message votes and message pool events)
	epochsBucket = []byte("epochs")

	ordinaryMessages = []byte("ordinary")
	rollbackMessages = []byte("rollback")

	// errNotEnoughBridgeEvents error message
	errNotEnoughBridgeEvents = errors.New("there is either a gap or not enough bridge events")
	// errNoBridgeBatchForBridgeEvent error message
	errNoBridgeBatchForBridgeEvent = errors.New("no bridge batch found for given bridge message events")
)

// BridgeBatchVoteConsensusData encapsulates sender identifier and its signature
type BridgeBatchVoteConsensusData struct {
	// Signer of the vote
	Sender string
	// Signature of the message
	Signature []byte
}

// BridgeBatchVote represents the payload which is gossiped across the network
type BridgeBatchVote struct {
	*BridgeBatchVoteConsensusData
	// Hash is encoded data
	Hash []byte
	// Number of epoch
	EpochNumber uint64
	// SourceChainID from bridge batch
	SourceChainID uint64
	// DestinationChainID from bridge batch
	DestinationChainID uint64
}

type BridgeManagerStore struct {
	db       *bolt.DB
	chainIDs []uint64
}

func newBridgeManagerStore(db *bolt.DB, dbTx *bolt.Tx, externalChainsIDs []uint64,
	internalChainID uint64) (*BridgeManagerStore, error) {
	var err error

	store := &BridgeManagerStore{db: db, chainIDs: externalChainsIDs}

	initFn := func(tx *bolt.Tx) error {
		var bridgeMessageBucket, bridgeBatchesBucket, epochBucket *bolt.Bucket

		if bridgeMessageBucket, err = tx.CreateBucketIfNotExists(bridgeMessageEventsBucket); err != nil {
			return fmt.Errorf("failed to create bucket=%s: %w", string(bridgeMessageEventsBucket), err)
		}

		if bridgeBatchesBucket, err = tx.CreateBucketIfNotExists(bridgeBatchBucket); err != nil {
			return fmt.Errorf("failed to create bucket=%s: %w", string(bridgeBatchBucket), err)
		}

		if epochBucket, err = tx.CreateBucketIfNotExists(epochsBucket); err != nil {
			return fmt.Errorf("failed to create bucket=%s: %w", string(epochsBucket), err)
		}

		// because we have multiple chains that can generate same events
		// we need to create bridge message bucket for each pair of internal and external chains
		createBucketsAndReturnBridgeMessageBucket := func(chainIDBytes []byte) (*bolt.Bucket, error) {
			bridgeMessageChainIDBucket, err := bridgeMessageBucket.CreateBucketIfNotExists(chainIDBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to create bucket chainID=%s: %w", string(bridgeMessageEventsBucket), err)
			}

			if _, err := bridgeBatchesBucket.CreateBucketIfNotExists(chainIDBytes); err != nil {
				return nil, fmt.Errorf("failed to create bucket chainID=%s: %w", string(bridgeBatchBucket), err)
			}

			if _, err := epochBucket.CreateBucketIfNotExists(chainIDBytes); err != nil {
				return nil, fmt.Errorf("failed to create bucket chainID=%s: %w", string(epochsBucket), err)
			}

			return bridgeMessageChainIDBucket, nil
		}

		internalChainIDBytes := common.EncodeUint64ToBytes(internalChainID)

		internalBridgeMessageBucket, err := createBucketsAndReturnBridgeMessageBucket(internalChainIDBytes)
		if err != nil {
			return err
		}

		for _, chainID := range externalChainsIDs {
			chainIDBytes := common.EncodeUint64ToBytes(chainID)

			bridgeMessageChainIDBucket, err := createBucketsAndReturnBridgeMessageBucket(chainIDBytes)
			if err != nil {
				return err
			}

			var buckets [2]*bolt.Bucket

			// create bucket for internal chainID in external chainID bucket
			if buckets[0], err = bridgeMessageChainIDBucket.CreateBucketIfNotExists(internalChainIDBytes); err != nil {
				return fmt.Errorf("failed to create bucket chainID=%s: %w", string(bridgeMessageEventsBucket), err)
			}

			// create bucket for external chainID in internal chainID bucket
			if buckets[1], err = internalBridgeMessageBucket.CreateBucketIfNotExists(chainIDBytes); err != nil {
				return fmt.Errorf("failed to create bucket chainID=%s: %w", string(bridgeMessageEventsBucket), err)
			}

			for _, bucket := range buckets {
				if _, err := bucket.CreateBucketIfNotExists(ordinaryMessages); err != nil {
					return fmt.Errorf("failed to create bucket for ordinary messages: %w", err)
				}

				if _, err := bucket.CreateBucketIfNotExists(rollbackMessages); err != nil {
					return fmt.Errorf("failed to create bucket for rollback messages: %w", err)
				}
			}
		}

		return nil
	}

	if dbTx == nil {
		err = db.Update(initFn)
	} else {
		err = initFn(dbTx)
	}

	return store, err
}

// insertBridgeMessageEvent inserts a new bridge message event to state event bucket in db
func (bms *BridgeManagerStore) insertBridgeMessageEvent(event *contractsapi.BridgeMsgEvent, isRollback bool, dbTx *bolt.Tx) error {
	insertFn := func(tx *bolt.Tx) error {
		raw, err := json.Marshal(event)
		if err != nil {
			return err
		}

		bucket := tx.Bucket(bridgeMessageEventsBucket).
			Bucket(common.EncodeUint64ToBytes(event.SourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(event.DestinationChainID.Uint64()))

		if isRollback {
			return bucket.Bucket(rollbackMessages).Put(common.EncodeUint64ToBytes(event.ID.Uint64()), raw)
		}

		return bucket.Bucket(ordinaryMessages).Put(common.EncodeUint64ToBytes(event.ID.Uint64()), raw)
	}

	if dbTx == nil {
		return bms.db.Update(insertFn)
	}

	return insertFn(dbTx)
}

func (bms *BridgeManagerStore) removeBridgeMessageEvent(messageID, sourceChainID, destinationChainID *big.Int, isRollback bool,
	dbTx *bolt.Tx) error {
	insertFn := func(tx *bolt.Tx) error {
		id := common.EncodeUint64ToBytes(messageID.Uint64())

		bucket := tx.Bucket(bridgeMessageEventsBucket).
			Bucket(common.EncodeUint64ToBytes(sourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(destinationChainID.Uint64()))

		if isRollback {
			return bucket.Bucket(rollbackMessages).Delete(id)
		}

		return bucket.Bucket(ordinaryMessages).Delete(id)
	}

	if dbTx == nil {
		return bms.db.Update(insertFn)
	}

	return insertFn(dbTx)
}

func (bms *BridgeManagerStore) getBridgeMessageEvent(messageID, sourceChainID, destinationChainID *big.Int, isRollback bool,
	dbTx *bolt.Tx) (*contractsapi.BridgeMsgEvent, error) {
	var message *contractsapi.BridgeMsgEvent

	insertFn := func(tx *bolt.Tx) error {
		id := common.EncodeUint64ToBytes(messageID.Uint64())

		bucket := tx.Bucket(bridgeMessageEventsBucket).
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
		if err := bms.db.Update(insertFn); err != nil {
			return nil, err
		}

		return message, nil
	}

	if err := insertFn(dbTx); err != nil {
		return nil, err
	}

	return message, nil
}

// removeBridgeEvents removes bridge events and their proofs from the buckets in db
func (bms *BridgeManagerStore) removeBridgeEvents(
	bridgeMessageResult contractsapi.BridgeMessageResultEvent, dbTx *bolt.Tx) error {
	insertFn := func(tx *bolt.Tx) error {
		eventsBucket := tx.Bucket(bridgeMessageEventsBucket).
			Bucket(common.EncodeUint64ToBytes(bridgeMessageResult.SourceChainID.Uint64())).
			Bucket(common.EncodeUint64ToBytes(bridgeMessageResult.DestinationChainID.Uint64()))

		bridgeMessageID := bridgeMessageResult.ID.Uint64()
		bridgeMessageEventIDKey := common.EncodeUint64ToBytes(bridgeMessageID)

		if err := eventsBucket.Delete(bridgeMessageEventIDKey); err != nil {
			return fmt.Errorf("failed to remove bridge message event (ID=%d): %w", bridgeMessageID, err)
		}

		return nil
	}

	if dbTx == nil {
		return bms.db.Update(func(tx *bolt.Tx) error {
			return insertFn(tx)
		})
	}

	return insertFn(dbTx)
}

func (bms *BridgeManagerStore) list(internalChainID uint64) ([]*contractsapi.BridgeMessage, error) {
	messages := []*contractsapi.BridgeMessage{}

	icid := common.EncodeUint64ToBytes(internalChainID)

	getMsgsFn := func(k, v []byte, isRollback bool) error {

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
			IsRollback:         isRollback,
		}

		messages = append(messages, message)

		return nil
	}

	ordMsgsFn := func(k, v []byte) error {
		return getMsgsFn(k, v, false)
	}

	rollMsgsFn := func(k, v []byte) error {
		return getMsgsFn(k, v, true)
	}

	err := bms.db.View(func(tx *bolt.Tx) error {
		bridgeMessageBucket := tx.Bucket(bridgeMessageEventsBucket)

		var err error

		for _, externalChainID := range bms.chainIDs {
			ecid := common.EncodeUint64ToBytes(externalChainID)
			if err = bridgeMessageBucket.Bucket(icid).Bucket(ecid).Bucket(ordinaryMessages).ForEach(ordMsgsFn); err != nil {
				break
			}

			if err = bridgeMessageBucket.Bucket(icid).Bucket(ecid).Bucket(rollbackMessages).ForEach(rollMsgsFn); err != nil {
				break
			}

			if err = bridgeMessageBucket.Bucket(ecid).Bucket(icid).Bucket(ordinaryMessages).ForEach(ordMsgsFn); err != nil {
				break
			}

			if err = bridgeMessageBucket.Bucket(icid).Bucket(ecid).Bucket(rollbackMessages).ForEach(rollMsgsFn); err != nil {
				break
			}
		}

		return err
	})

	return messages, err
}

func (bms *BridgeManagerStore) getBridgeMessages(fromIndex, limit, sid, did uint64, dbTx *bolt.Tx) ([]*contractsapi.BridgeMessage, uint64, error) {
	if limit == 0 {
		return nil, 0, nil
	}

	var (
		messages          []*contractsapi.BridgeMessage
		err               error
		limitReachedErr   error
		numOfOrdinaryMsgs uint64
	)

	getFn := func(tx *bolt.Tx) error {
		var taken uint64

		bucket := tx.Bucket(bridgeMessageEventsBucket).
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

		err := tx.Bucket(bridgeMessageEventsBucket).
			Bucket(common.EncodeUint64ToBytes(sid)).
			Bucket(common.EncodeUint64ToBytes(did)).
			Bucket(rollbackMessages).ForEach(func(k, v []byte) error {

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
				IsRollback:         true,
			}

			messages = append(messages, message)

			taken++

			if taken == limit {
				return limitReachedErr
			}

			return nil
		})

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
		err = bms.db.View(getFn)
	} else {
		err = getFn(dbTx)
	}

	return messages, numOfOrdinaryMsgs, err
}

// getBridgeBatchForBridgeEvents returns the bridgeBatch that contains given bridge event if it exists
func (bms *BridgeManagerStore) getBridgeBatchForBridgeEvents(
	bridgeMessageID,
	chainID uint64) (*BridgeBatchSigned, error) {
	var signedBridgeBatch *BridgeBatchSigned

	err := bms.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bridgeBatchBucket).Bucket(common.EncodeUint64ToBytes(chainID)).Cursor()

		k, v := c.Seek(common.EncodeUint64ToBytes(bridgeMessageID))
		if k == nil {
			return errNoBridgeBatchForBridgeEvent
		}

		if err := json.Unmarshal(v, &signedBridgeBatch); err != nil {
			return err
		}

		if !signedBridgeBatch.ContainsBridgeMessage(bridgeMessageID) {
			return errNoBridgeBatchForBridgeEvent
		}

		return nil
	})

	return signedBridgeBatch, err
}

// insertBridgeBatchMessage inserts signed batch to db
func (bms *BridgeManagerStore) insertBridgeBatchMessage(signedBridgeBatch *BridgeBatchSigned,
	dbTx *bolt.Tx) error {
	insertFn := func(tx *bolt.Tx) error {
		raw, err := json.Marshal(signedBridgeBatch)
		if err != nil {
			return err
		}

		if err := tx.Bucket(bridgeBatchBucket).
			Bucket(common.EncodeUint64ToBytes(
				signedBridgeBatch.BridgeMessageBatch.SourceChainID.Uint64())).Put(
			common.EncodeUint64ToBytes(
				signedBridgeBatch.BridgeMessageBatch.Messages[len(signedBridgeBatch.Messages)-1].
					ID.Uint64()), raw); err != nil {
			return err
		}

		return nil
	}

	if dbTx == nil {
		return bms.db.Update(func(tx *bolt.Tx) error {
			return insertFn(tx)
		})
	}

	return insertFn(dbTx)
}

// insertConsensusData inserts given batch consensus data to corresponding bucket of given epoch
func (bms *BridgeManagerStore) insertConsensusData(epoch uint64, key []byte,
	vote *BridgeBatchVoteConsensusData, dbTx *bolt.Tx, sourceChainID uint64) (int, error) {
	var (
		numOfSignatures int
		err             error
	)

	insertFn := func(tx *bolt.Tx) error {
		signatures, err := bms.getMessageVotesLocked(tx, epoch, key, sourceChainID)
		if err != nil {
			return err
		}

		// check if the signature has already being included
		for _, sigs := range signatures {
			if sigs.Sender == vote.Sender {
				return nil
			}
		}

		if signatures == nil {
			signatures = []*BridgeBatchVoteConsensusData{vote}
		} else {
			signatures = append(signatures, vote)
		}

		raw, err := json.Marshal(signatures)
		if err != nil {
			return err
		}

		bucket, err := getNestedBucketInEpoch(tx, epoch, messageVotesBucket, sourceChainID)
		if err != nil {
			return err
		}

		numOfSignatures = len(signatures)

		return bucket.Put(key, raw)
	}

	if dbTx == nil {
		err = bms.db.Update(func(tx *bolt.Tx) error {
			return insertFn(tx)
		})
	} else {
		err = insertFn(dbTx)
	}

	return numOfSignatures, err
}

// getMessageVotes gets all signatures from db associated with given epoch and hash
func (bms *BridgeManagerStore) getMessageVotes(
	epoch uint64,
	hash []byte,
	sourceChainID uint64) ([]*BridgeBatchVoteConsensusData, error) {
	var signatures []*BridgeBatchVoteConsensusData

	err := bms.db.View(func(tx *bolt.Tx) error {
		res, err := bms.getMessageVotesLocked(tx, epoch, hash, sourceChainID)
		if err != nil {
			return err
		}

		signatures = res

		return nil
	})

	if err != nil {
		return nil, err
	}

	return signatures, nil
}

// getMessageVotesLocked gets all signatures from db associated with given epoch and hash
func (bms *BridgeManagerStore) getMessageVotesLocked(tx *bolt.Tx, epoch uint64,
	hash []byte, sourceChainID uint64) ([]*BridgeBatchVoteConsensusData, error) {
	bucket, err := getNestedBucketInEpoch(tx, epoch, messageVotesBucket, sourceChainID)
	if err != nil {
		return nil, err
	}

	v := bucket.Get(hash)
	if v == nil {
		return nil, nil
	}

	var signatures []*BridgeBatchVoteConsensusData
	if err := json.Unmarshal(v, &signatures); err != nil {
		return nil, err
	}

	return signatures, nil
}

// getNestedBucketInEpoch returns a nested (child) bucket from db associated with given epoch
func getNestedBucketInEpoch(tx *bolt.Tx, epoch uint64, bucketKey []byte, chainID uint64) (*bolt.Bucket, error) {
	epochBucket, err := getEpochBucket(tx, epoch, chainID)
	if err != nil {
		return nil, err
	}

	bucket := epochBucket.Bucket(bucketKey)
	if bucket == nil {
		return nil, fmt.Errorf("could not find %v bucket for epoch: %v", string(bucketKey), epoch)
	}

	return bucket, nil
}

// getEpochBucket returns bucket from db associated with given epoch
func getEpochBucket(tx *bolt.Tx, epoch uint64, chainID uint64) (*bolt.Bucket, error) {
	epochBucket := tx.Bucket(epochsBucket).
		Bucket(common.EncodeUint64ToBytes(chainID)).
		Bucket(common.EncodeUint64ToBytes(epoch))
	if epochBucket == nil {
		return nil, fmt.Errorf("could not find bucket for epoch: %v", epoch)
	}

	return epochBucket, nil
}

// insertEpoch inserts a new epoch to db with its meta data
func (bms *BridgeManagerStore) insertEpoch(epoch uint64, dbTx *bolt.Tx, chainID uint64) error {
	insertFn := func(tx *bolt.Tx) error {
		chainIDBucket, err := tx.Bucket(epochsBucket).CreateBucketIfNotExists(common.EncodeUint64ToBytes(chainID))
		if err != nil {
			return err
		}

		epochBucket, err := chainIDBucket.CreateBucketIfNotExists(common.EncodeUint64ToBytes(epoch))
		if err != nil {
			return err
		}

		_, err = epochBucket.CreateBucketIfNotExists(messageVotesBucket)
		if err != nil {
			return err
		}

		return err
	}

	if dbTx == nil {
		return bms.db.Update(func(tx *bolt.Tx) error {
			return insertFn(tx)
		})
	}

	return insertFn(dbTx)
}

// cleanEpochsFromDB cleans epoch buckets from db
func (bms *BridgeManagerStore) cleanEpochsFromDB(dbTx *bolt.Tx) error {
	cleanFn := func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(epochsBucket); err != nil {
			return err
		}

		epochBucket, err := tx.CreateBucket(epochsBucket)
		if err != nil {
			return err
		}

		for _, chainID := range bms.chainIDs {
			if _, err := epochBucket.CreateBucket(common.EncodeUint64ToBytes(chainID)); err != nil {
				return err
			}
		}

		return nil
	}

	if dbTx == nil {
		return bms.db.Update(func(tx *bolt.Tx) error {
			return cleanFn(tx)
		})
	}

	return cleanFn(dbTx)
}
