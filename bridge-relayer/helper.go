package bridgerelayer

import (
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
)

func GetBridgeBatchesFromNumber(batchID *big.Int,
	internalRelayer txrelayer.TxRelayer) ([]contractsapi.SignedBridgeMessageBatch, error) {
	funcName := "getCommittedBatches"

	getCommittedBatchFn := contractsapi.BridgeStorage.Abi.GetMethod(funcName)
	if getCommittedBatchFn == nil {
		return nil, fmt.Errorf("failed to resolve %s function", funcName)
	}

	encode, err := getCommittedBatchFn.Encode([]interface{}{batchID})
	if err != nil {
		return nil, err
	}

	response, err := internalRelayer.Call(types.ZeroAddress, contracts.BridgeStorageContract, encode)
	if err != nil {
		return nil, err
	}

	byteResponse, err := hex.DecodeHex(response)
	if err != nil {
		return nil, fmt.Errorf("unable to decode hex response, %w", err)
	}

	decoded, err := getCommittedBatchFn.Outputs.Decode(byteResponse)
	if err != nil {
		return nil, err
	}

	decodedSlice, ok := decoded.(map[string]interface{})["0"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert decoded output to slice")
	}

	signedBridgeBatches := make([]contractsapi.SignedBridgeMessageBatch, len(decodedSlice))

	for i, v := range decodedSlice {
		decodedBatch, ok := v["batch"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid format of the batch")
		}

		decodedSourceChainID, ok := decodedBatch["sourceChainId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the source chain ID")
		}

		decodedDestinationChainID, ok := decodedBatch["destinationChainId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the destination chain ID")
		}

		decodedThreshold, ok := decodedBatch["threshold"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the threshold")
		}

		decodedNumberOfRegularEvents, ok := decodedBatch["numberOfRegularEvents"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the number of regular events")
		}

		decodedValidationCounter, ok := decodedBatch["validationCounter"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the validation counter")
		}

		rawMessages, ok := decodedBatch["messages"].([]map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid format of the batch messages")
		}

		decodedMessages, err := decodeBridgeMessages(rawMessages)
		if err != nil {
			return nil, err
		}

		decodedBitmap, ok := v["bitmap"].([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid format of the bitmap")
		}

		decodedSignature, ok := v["signature"].([2]*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the signature")
		}

		decodedValidatorSetBatchID, ok := v["validatorSetBatchId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the validator set batch ID")
		}

		signedBridgeBatches[i] = contractsapi.SignedBridgeMessageBatch{
			Batch: &contractsapi.BridgeMessageBatch{
				Messages:              decodedMessages,
				SourceChainID:         decodedSourceChainID,
				DestinationChainID:    decodedDestinationChainID,
				Threshold:             decodedThreshold,
				NumberOfRegularEvents: decodedNumberOfRegularEvents,
				ValidationCounter:     decodedValidationCounter,
			},
			Signature:           decodedSignature,
			Bitmap:              decodedBitmap,
			ValidatorSetBatchID: decodedValidatorSetBatchID,
		}
	}

	return signedBridgeBatches, nil
}

func decodeBridgeMessages(rawMessages []map[string]interface{}) ([]*contractsapi.BridgeMessage, error) {
	bridgeMessages := make([]*contractsapi.BridgeMessage, len(rawMessages))

	for i, v := range rawMessages {
		decodedID, ok := v["id"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the root hash")
		}

		decodedSourceChainID, ok := v["sourceChainId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the source chain ID")
		}

		decodedDestinationChainID, ok := v["destinationChainId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the destination chain ID")
		}

		decodedSender, ok := v["sender"].(ethgo.Address)
		if !ok {
			return nil, fmt.Errorf("invalid format of the sender")
		}

		decodedReceiver, ok := v["receiver"].(ethgo.Address)
		if !ok {
			return nil, fmt.Errorf("invalid format of the receiver")
		}

		decodedIsRollback, ok := v["isRollback"].(bool)
		if !ok {
			return nil, fmt.Errorf("invalid format of the rollback flag")
		}

		decodedPayload, ok := v["payload"].([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid format of the payload")
		}

		bridgeMessages[i] = &contractsapi.BridgeMessage{
			ID:                 decodedID,
			SourceChainID:      decodedSourceChainID,
			DestinationChainID: decodedDestinationChainID,
			Sender:             types.Address(decodedSender),
			Receiver:           types.Address(decodedReceiver),
			IsRollback:         decodedIsRollback,
			Payload:            decodedPayload,
		}
	}

	return bridgeMessages, nil
}

func GetBridgeValidatorSet(
	commitValidatorSetid *big.Int,
	txrelayer txrelayer.TxRelayer) (*contractsapi.SignedValidatorSet, error) {
	funcName := "getCommittedValidatorSet"
	validatorSet := &contractsapi.SignedValidatorSet{}

	getCommittedBatchFn := contractsapi.BridgeStorage.Abi.GetMethod(funcName)
	if getCommittedBatchFn == nil {
		return nil, fmt.Errorf("failed to resolve %s function", funcName)
	}

	encode, err := getCommittedBatchFn.Encode([]interface{}{commitValidatorSetid})
	if err != nil {
		return nil, err
	}

	response, err := txrelayer.Call(types.ZeroAddress, contracts.BridgeStorageContract, encode)
	if err != nil {
		return nil, err
	}

	byteResponse, err := hex.DecodeHex(response)
	if err != nil {
		return nil, fmt.Errorf("unable to decode hex response, %w", err)
	}

	if err := validatorSet.DecodeAbi(byteResponse[types.StorageSlotSize:]); err != nil {
		return nil, err
	}

	return validatorSet, nil
}
