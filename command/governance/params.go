package governance

const (
	validatorKeysFlag = "validator-keys"
)

// sanityCheckParams holds the parameters for the sanity check command
type governanceParams struct {
	jsonRPCAddress string

	toJSON bool

	validatorKeys []string
}
