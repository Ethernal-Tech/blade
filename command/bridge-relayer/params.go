package relayer

// commandParams defines all the existing parameters for `bridge-relayer` command
// that can be used for configuring bridge relayer.
type commandParams struct {
	// internalChainRPC represents the RPC endpoint of the internal blockchain
	// network (Blade).
	internalChainRPC string

	// genesisPath denotes the path to the genesis file of the internal blockchain
	// network (Blade).
	genesisPath string

	// externalChainID represents the unique identifier of the external blockchain
	// network to which the bridge relayer will connect and relay transactions.
	externalChainID int

	// pollInterval specifies the interval (in seconds) at which the relayer checks
	// for new cross-chain transactions to process.
	pollInterval int

	// relayerPrivateKey contains the cryptographic private key utilized by the
	// relayer to sign and send transactions.
	// SECURITY WARNING: This key must be kept strictly confidential and never
	// exposed or committed to any public service or version control system.
	relayerPrivateKey string

	// metricsEndpoint specifies the port and path (in the format :PORT/PATH) on
	// which metrics for Prometheus will be exposed.
	metricsEndpoint string

	// heartbeatThreshold denotes balance (expressed in the lowest unit (e.g. wei)
	// of the external chain native tokens) below which the relayer is considered
	// in alarm. When the relayer's balance falls below this value, the relayer will
	// signal an alarm, that is, it will send 0 instead of 1 to Prometheus. This
	// param only takes effect if the metricsEndpoint is set/configured.
	heartbeatThreshold string

	// this parameter enables JSON-formatted logs by setting its value to true.
	jsonFormatOuttputter bool

	// determine the verbosity of log messages.
	logLevel string

	// path to log file.
	logFilePath string

	// path to bolt db
	boltDBPath string
}
