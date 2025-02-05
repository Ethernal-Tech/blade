package bridge

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/bls"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/helpers"
	polytypes "github.com/0xPolygon/polygon-edge/consensus/polybft/types"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/types"
)

// PendingBridgeBatch holds pending bridge batch for epoch
type PendingBridgeBatch struct {
	*contractsapi.BridgeMessageBatch
	Epoch uint64
}

// NewPendingBridgeBatch creates a new PendingBridgeBatch object
func NewPendingBridgeBatch(epoch uint64,
	bridgeEvents []*contractsapi.BridgeMsgEvent) (*PendingBridgeBatch, error) {
	if len(bridgeEvents) == 0 {
		return nil, nil
	}

	messages := make([]*contractsapi.BridgeMessage, len(bridgeEvents))

	for i, bridgeEvent := range bridgeEvents {
		messages[i] = &contractsapi.BridgeMessage{
			ID:                 bridgeEvent.ID,
			Sender:             bridgeEvent.Sender,
			Receiver:           bridgeEvent.Receiver,
			Payload:            bridgeEvent.Data,
			SourceChainID:      bridgeEvent.SourceChainID,
			DestinationChainID: bridgeEvent.DestinationChainID}
	}

	firstBridgeMessage := bridgeEvents[0]

	return &PendingBridgeBatch{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           messages,
			SourceChainID:      firstBridgeMessage.SourceChainID,
			DestinationChainID: firstBridgeMessage.DestinationChainID,
			Threshold:          big.NewInt(0),
			IsRollback:         false,
		},
		Epoch: epoch,
	}, nil
}

// Hash calculates hash value for PendingBridgeBatch object.
func (pbb *PendingBridgeBatch) Hash() (types.Hash, error) {
	data, err := pbb.BridgeMessageBatch.EncodeAbi()
	if err != nil {
		return types.ZeroHash, err
	}

	return crypto.Keccak256Hash(data), nil
}

var _ contractsapi.ABIEncoder = &BridgeBatchSigned{}

// BridgeBatchSigned encapsulates bridge batch with aggregated signatures
type BridgeBatchSigned struct {
	*contractsapi.BridgeMessageBatch
	AggSignature polytypes.Signature
}

// Hash calculates hash value for BridgeBatchSigned object.
func (bbs *BridgeBatchSigned) Hash() (types.Hash, error) {
	data, err := bbs.BridgeMessageBatch.EncodeAbi()
	if err != nil {
		return types.ZeroHash, err
	}

	return crypto.Keccak256Hash(data), nil
}

// ContainsBridgeMessage checks if BridgeBatchSigned contains given bridge message event
func (bbs *BridgeBatchSigned) ContainsBridgeMessage(bridgeMessageID uint64) bool {
	length := len(bbs.Messages)
	if length == 0 {
		return false
	}

	return bbs.Messages[0].ID.Uint64() <= bridgeMessageID &&
		bbs.Messages[length-1].ID.Uint64() >= bridgeMessageID
}

func (bbs *BridgeBatchSigned) IsE2IBatch() bool {
	return !bbs.IsRollback && bbs.SourceChainID.Cmp(big.NewInt(100)) != 0 ||
		bbs.IsRollback && bbs.SourceChainID.Cmp(big.NewInt(100)) == 0
}

// EncodeAbi contains logic for encoding arbitrary data into ABI format
func (bbs *BridgeBatchSigned) EncodeAbi() ([]byte, error) {
	blsSignatrure, err := bls.UnmarshalSignature(bbs.AggSignature.AggregatedSignature)
	if err != nil {
		return nil, err
	}

	signature, err := blsSignatrure.ToBigInt()
	if err != nil {
		return nil, err
	}

	// currently it is hard-coded that the blade chain ID is 100,
	// it should be updated to dynamically read the ID from the config
	if bbs.IsE2IBatch() {
		return (&contractsapi.ReceiveBatchGatewayFn{
			SignedBatch: &contractsapi.SignedBridgeMessageBatch{
				Batch:               bbs.BridgeMessageBatch,
				Signature:           signature,
				Bitmap:              bbs.AggSignature.Bitmap,
				ValidatorSetBatchID: big.NewInt(0),
			},
		}).EncodeAbi()
	} else {
		return (&contractsapi.CommitBatchBridgeStorageFn{
			SignedBatch: &contractsapi.SignedBridgeMessageBatch{
				Batch:               bbs.BridgeMessageBatch,
				Signature:           signature,
				Bitmap:              bbs.AggSignature.Bitmap,
				ValidatorSetBatchID: big.NewInt(0),
			},
		}).EncodeAbi()
	}
}

// DecodeAbi contains logic for decoding given ABI data
func (bbs *BridgeBatchSigned) DecodeAbi(txData []byte) error {
	receiveBatchFn := contractsapi.ReceiveBatchGatewayFn{}
	commitBatchFn := contractsapi.CommitBatchBridgeStorageFn{}

	if len(txData) < helpers.AbiMethodIDLength {
		return fmt.Errorf("invalid batch data, len = %d", len(txData))
	}

	sig := txData[:helpers.AbiMethodIDLength]

	if bytes.Equal(sig, receiveBatchFn.Sig()) {
		if err := receiveBatchFn.DecodeAbi(txData); err != nil {
			return err
		}

		if err := bbs.constructFromSignedBatch(receiveBatchFn.SignedBatch); err != nil {
			return err
		}
	} else if bytes.Equal(sig, commitBatchFn.Sig()) {
		if err := commitBatchFn.DecodeAbi(txData); err != nil {
			return err
		}

		if err := bbs.constructFromSignedBatch(commitBatchFn.SignedBatch); err != nil {
			return err
		}
	}

	return nil
}

func (bbs *BridgeBatchSigned) constructFromSignedBatch(signedBatch *contractsapi.SignedBridgeMessageBatch) error {
	signature0 := signedBatch.Signature[0].Bytes()
	signature1 := signedBatch.Signature[1].Bytes()
	halfSignatureSize := bls.SignatureSize / 2
	signature := make([]byte, bls.SignatureSize)

	if len(signature0) < halfSignatureSize {
		copy(signature[halfSignatureSize-len(signature0):halfSignatureSize], signature0)
	} else {
		copy(signature[:halfSignatureSize], signature0)
	}

	if len(signature1) < halfSignatureSize {
		copy(signature[bls.SignatureSize-len(signature1):], signature1)
	} else {
		copy(signature[halfSignatureSize:], signature1)
	}

	*bbs = BridgeBatchSigned{
		BridgeMessageBatch: &contractsapi.BridgeMessageBatch{
			Messages:           signedBatch.Batch.Messages,
			SourceChainID:      signedBatch.Batch.SourceChainID,
			DestinationChainID: signedBatch.Batch.DestinationChainID,
			Threshold:          signedBatch.Batch.Threshold,
			IsRollback:         signedBatch.Batch.IsRollback,
		},
		AggSignature: polytypes.Signature{
			AggregatedSignature: signature,
			Bitmap:              signedBatch.Bitmap,
		},
	}

	return nil
}
