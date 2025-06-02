package governance

import (
	"fmt"
	"time"

	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
)

func waitUntil(timeout time.Duration, pollFrequency time.Duration, handler func() (bool, error)) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		case <-time.After(pollFrequency):
		}

		result, err := handler()
		if err != nil {
			return err
		}

		if result {
			return nil
		}
	}
}

func waitForBlock(n int64, timeout time.Duration) error {
	timer := time.NewTicker(2 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)

	defer func() {
		timer.Stop()
		ticker.Stop()
		fmt.Println("Waiting for block finished")
	}()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timed out waiting for block")
		case <-ticker.C:
			rpcBlock := jsonrpc.LatestBlockNumber
			if rpcBlock >= jsonrpc.BlockNumber(n) {
				return nil
			}
		}
	}
}

func ABICall(relayer txrelayer.TxRelayer, artifact *contracts.Artifact, contractAddress types.Address, senderAddr types.Address, method string, params ...interface{}) (string, error) {
	input, err := artifact.Abi.GetMethod(method).Encode(params)
	if err != nil {
		return "", err
	}

	return relayer.Call(senderAddr, contractAddress, input)
}
