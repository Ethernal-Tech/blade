package loadtest

import (
	"errors"
	"fmt"
	"time"

	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/0xPolygon/polygon-edge/loadtest/runner"
)

const (
	MnemonicFlag        = "mnemonic"
	SaveToJSONFlag      = "to-json"
	ReceiptsTimeoutFlag = "receipts-timeout"

	loadTestTypeFlag = "type"
	loadTestNameFlag = "name"

	txPoolTimeoutFlag = "txpool-timeout"

	vusFlag        = "vus"
	txsPerUserFlag = "txs-per-user"
	dynamicTxsFlag = "dynamic"
	batchSizeFlag  = "batch-size"

	waitForTxPoolToEmptyFlag = "wait-txpool"

	executionTimeFlag     = "execution-time"
	stateReadThreadsFlag  = "state-read-threads"
	txpoolReadThreadsFlag = "txpool-read-threads"
)

var (
	ErrNoMnemonicProvided      = errors.New("no mnemonic provided")
	errNoLoadTestTypeProvided  = errors.New("no load test type provided")
	errUnsupportedLoadTestType = errors.New("unsupported load test type")
	errInvalidVUs              = errors.New("vus must be greater than 0")
	errInvalidTxsPerUser       = errors.New("txs-per-user must be greater than 0")
	errInvalidBatchSize        = errors.New("batch-size must be greater than 0 " +
		"and less or equal to txs-per-user")
	errInvalidExecutionTime                 = errors.New("when set execution-time must be at least 1s or greater")
	errInvalidExecutionTimeAndTxPoolTimeout = errors.New("txpool-timeout must be greater than execution-time")
	errInvalidNumOfJsonRPCAddresses         = errors.New("at least one JSON-RPC address must be provided")
)

type loadTestParams struct {
	mnemonic         string
	loadTestType     string
	loadTestName     string
	jsonRPCAddresses []string

	receiptsTimeout time.Duration
	txPoolTimeout   time.Duration

	vus        int
	txsPerUser int
	batchSize  int

	dynamicTxs           bool
	toJSON               bool
	waitForTxPoolToEmpty bool

	executionTime     time.Duration
	stateReadThreads  uint32
	txpoolReadThreads uint32
}

func (ltp *loadTestParams) validateFlags() error {
	if ltp.mnemonic == "" {
		return ErrNoMnemonicProvided
	}

	if ltp.loadTestType == "" {
		return errNoLoadTestTypeProvided
	}

	if !runner.IsLoadTestSupported(ltp.loadTestType) {
		return errUnsupportedLoadTestType
	}

	if ltp.vus < 1 {
		return errInvalidVUs
	}

	if ltp.txsPerUser < 1 {
		return errInvalidTxsPerUser
	}

	if ltp.batchSize < 1 || (ltp.batchSize > ltp.txsPerUser && ltp.executionTime == 0) {
		return errInvalidBatchSize
	}

	if len(ltp.jsonRPCAddresses) == 0 {
		return errInvalidNumOfJsonRPCAddresses
	} else {
		// validate each address
		for _, addr := range ltp.jsonRPCAddresses {
			if _, err := helper.ParseJSONRPCAddress(addr); err != nil {
				return fmt.Errorf("failed to parse json rpc address: %s. Error: %w", addr, err)
			}
		}
	}

	if ltp.executionTime > 0 {
		if ltp.executionTime < time.Second {
			return errInvalidExecutionTime
		}

		if ltp.txPoolTimeout < ltp.executionTime {
			return errInvalidExecutionTimeAndTxPoolTimeout
		}
	}

	return nil
}
