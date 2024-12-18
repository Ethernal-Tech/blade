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
	externalChainID uint64

	// pollInterval specifies the interval (in seconds) at which the relayer checks
	// for new cross-chain transactions to process.
	pollInterval uint64

	// relayerPrivateKey contains the cryptographic private key utilized by the
	// relayer to sign and send transactions.
	// SECURITY WARNING: This key must be kept strictly confidential and never
	// exposed or committed to any public service or version control system.
	relayerPrivateKey string
}
