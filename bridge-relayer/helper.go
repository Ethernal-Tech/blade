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
		decodeRootHash, ok := v["rootHash"].([32]uint8)
		if !ok {
			return nil, fmt.Errorf("invalid format of the root hash")
		}

		decodedStartID, ok := v["startId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the start ID")
		}

		decodedEndID, ok := v["endId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the end ID")
		}

		decodedSourceChainID, ok := v["sourceChainId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the source chain ID")
		}

		decodedDestinationChainID, ok := v["destinationChainId"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the destination chain ID")
		}

		decodedBitmap, ok := v["bitmap"].([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid format of the bitmap")
		}

		decodedThreshold, ok := v["threshold"].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the threshold")
		}

		decodedIsRollback, ok := v["isRollback"].(bool)
		if !ok {
			return nil, fmt.Errorf("invalid format of the rollback flag")
		}

		decodedSignature, ok := v["signature"].([2]*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid format of the signature")
		}

		signedBridgeBatches[i] = contractsapi.SignedBridgeMessageBatch{
			RootHash:           decodeRootHash,
			StartID:            decodedStartID,
			EndID:              decodedEndID,
			SourceChainID:      decodedSourceChainID,
			DestinationChainID: decodedDestinationChainID,
			Signature:          decodedSignature,
			Bitmap:             decodedBitmap,
			Threshold:          decodedThreshold,
			IsRollback:         decodedIsRollback,
		}
	}

	return signedBridgeBatches, nil
}

func GetBridgeMessagesInRange(startID, endID *big.Int, txrelayer txrelayer.TxRelayer,
	gatewayContract types.Address) ([]*contractsapi.BridgeMessage, error) {

	funcName := "getMessagesInRange"

	getCommittedBatchFn := contractsapi.Gateway.Abi.GetMethod(funcName)
	if getCommittedBatchFn == nil {
		return nil, fmt.Errorf("failed to resolve %s function", funcName)
	}

	encode, err := getCommittedBatchFn.Encode([]interface{}{startID, endID})
	if err != nil {
		return nil, err
	}

	response, err := txrelayer.Call(types.ZeroAddress, gatewayContract, encode)
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

	bridgeMessages := make([]*contractsapi.BridgeMessage, len(decodedSlice))

	for i, v := range decodedSlice {
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
			Payload:            decodedPayload,
		}
	}

	return bridgeMessages, nil
}
