package loadtest

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	txsPerTimeUnitFlag    = "txs-per-time-unit"
	stateReadThreadsFlag  = "state-read-threads"
	txpoolReadThreadsFlag = "txpool-read-threads"

	minTxsPerTimeUnitParamsNumber = 2
)

var (
	ErrNoMnemonicProvided          = errors.New("no mnemonic provided")
	errNoLoadTestTypeProvided      = errors.New("no load test type provided")
	errUnsupportedLoadTestType     = errors.New("unsupported load test type")
	errInvalidVUs                  = errors.New("vus must be greater than 0")
	errInvalidTxsPerUser           = errors.New("txs-per-user must be greater than 0")
	errInvalidBatchSize            = errors.New("batch-size must be greater than 0 and less or equal to txs-per-user")
	errInvalidExecutionTime        = errors.New("when set execution-time must be at least 1s or greater")
	errInvalidTxsPerTimeUnit       = errors.New("invalid txs-per-time-unit format")
	errInvalidTxsPerTimeUnitParams = errors.New("invalid txs-per-time-unit params")
)

type loadTestParams struct {
	mnemonic       string
	loadTestType   string
	loadTestName   string
	jsonRPCAddress string

	receiptsTimeout time.Duration
	txPoolTimeout   time.Duration

	vus        int
	txsPerUser int
	batchSize  int

	dynamicTxs           bool
	toJSON               bool
	waitForTxPoolToEmpty bool

	executionTime     time.Duration
	txsPerTimeUnit    string
	stateReadThreads  uint32
	txpoolReadThreads uint32

	numTxsPerTimeUnit int64
	numTimeUnits      time.Duration
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

	if ltp.batchSize < 1 || ltp.batchSize > ltp.txsPerUser {
		return errInvalidBatchSize
	}

	if ltp.executionTime > 0 && ltp.executionTime < time.Second {
		return errInvalidExecutionTime
	}

	if ltp.txsPerTimeUnit != "" {
		if err := ltp.parseTxsPerTimeUnit(); err != nil {
			return err
		}
	}

	return nil
}

func (ltp *loadTestParams) parseTxsPerTimeUnit() error {
	params := strings.Split(ltp.txsPerTimeUnit, ":")
	if len(params) < minTxsPerTimeUnitParamsNumber {
		return errInvalidTxsPerTimeUnit
	}

	// num of txs
	numTxs, err := strconv.ParseInt(strings.TrimSpace(params[0]), 10, 8)
	if err != nil {
		return errInvalidTxsPerTimeUnitParams
	}

	ltp.numTxsPerTimeUnit = numTxs

	// time unit
	timeUnit := strings.TrimSpace(params[1])
	time, err := time.ParseDuration(timeUnit)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidTxsPerTimeUnitParams, err)
	}

	ltp.numTimeUnits = time

	return nil
}
