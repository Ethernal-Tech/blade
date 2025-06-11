package governance

import (
	"errors"
)

const (
	epochSizeFlag     = "epoch-size"
	blockTimeFlag     = "block-time"
	validatorKeysFlag = "validator-keys"
)

var (
	errInvalidEpochSize = errors.New("epoch size must be greater than 0")
	errInvalidBlockTime = errors.New("block time must be greater than 0")
)

// sanityCheckParams holds the parameters for the sanity check command
type governanceParams struct {
	epochSize uint64

	blockTime uint64

	jsonRPCAddress string

	toJSON bool

	validatorKeys []string
}

// validateFlags checks if the provided flags are valid
func (scp *governanceParams) validateFlags() error {
	if scp.epochSize == 0 {
		return errInvalidEpochSize
	}

	if scp.blockTime == 0 {
		return errInvalidBlockTime
	}

	return nil
}
