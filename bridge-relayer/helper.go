package bridgerelayer

import (
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
)

func GetBridgeBatchesFromNumber(batchID *big.Int, internalRelayer txrelayer.TxRelayer) ([]contractsapi.SignedBridgeMessageBatch, error) {
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

	decodedSlice, ok := decoded.([]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert decoded output to slice")
	}

	var signedBridgeBatches []contractsapi.SignedBridgeMessageBatch

	for _, v := range decodedSlice {

		decodedOutputsMap, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("could not convert decoded outputs to map")
		}

		innerMap, ok := decodedOutputsMap["0"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("could not convert decoded outputs map to inner map")
		}

		bridgeBatch := contractsapi.SignedBridgeMessageBatch{
			RootHash:           innerMap["rootHash"].(types.Hash),
			StartID:            innerMap["startId"].(*big.Int),
			EndID:              innerMap["endId"].(*big.Int),
			SourceChainID:      innerMap["sourceChainId"].(*big.Int),
			DestinationChainID: innerMap["destinationChainId"].(*big.Int),
			Bitmap:             innerMap["bitmap"].([]byte),
			Threshold:          innerMap["threshold"].(*big.Int),
			IsRollback:         innerMap["isRollback"].(bool),
		}

		decodedSignature, ok := innerMap["signature"].([]interface{})
		if !ok || len(decodedSignature) != 2 {
			return nil, fmt.Errorf("invalid format for signature")
		}

		for i, v := range decodedSignature {
			if bigIntVal, ok := v.(*big.Int); ok {
				bridgeBatch.Signature[i] = bigIntVal
			} else {
				return nil, fmt.Errorf("failed to cast signature[%d] to *big.Int", i)
			}
		}

		signedBridgeBatches = append(signedBridgeBatches, bridgeBatch)
	}

	return signedBridgeBatches, nil
}

func GetBridgeMessagesInRange(startID, endID *big.Int, txrelayer txrelayer.TxRelayer, gatewayContract types.Address) ([]contractsapi.BridgeMessage, error) {
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

	decodedSlice, ok := decoded.([]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert decoded output to slice")
	}

	var bridgeMessages []contractsapi.BridgeMessage

	for _, v := range decodedSlice {

		decodedOutputsMap, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("could not convert decoded outputs to map")
		}

		innerMap, ok := decodedOutputsMap["0"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("could not convert decoded outputs map to inner map")
		}

		bridgeBatch := contractsapi.BridgeMessage{
			ID:                 innerMap["id"].(*big.Int),
			SourceChainID:      innerMap["sourceChainId"].(*big.Int),
			DestinationChainID: innerMap["destinationChainId"].(*big.Int),
			Sender:             innerMap["sender"].(types.Address),
			Receiver:           innerMap["receiver"].(types.Address),
			Payload:            innerMap["payload"].([]byte),
		}

		bridgeMessages = append(bridgeMessages, bridgeBatch)
	}

	return bridgeMessages, nil
}
