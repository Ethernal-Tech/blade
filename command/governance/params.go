package governance

import (
	"errors"
)

const (
	epochSizeFlag     = "epoch-size"
	validatorKeysFlag = "validator-keys"
)

var (
	errInvalidEpochSize = errors.New("epoch size must be greater than 0")
)

// sanityCheckParams holds the parameters for the sanity check command
type governanceParams struct {
	jsonRPCAddress string

	toJSON bool

	validatorKeys []string
}
