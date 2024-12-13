package relayer

// relayerParams defines all the configuration parameters necessary for the
// bridge relayer to function properly.
type relayerParams struct {
	// internalChainRPC represents the RPC endpoint of the internal blockchain
	// network (Blade).
	internalChainRPC string

	// relayerPrivateKey contains the cryptographic private key utilized by the
	// relayer to sign and send transactions.
	// SECURITY WARNING: This key must be kept strictly confidential and never
	// exposed or committed to any public service or version control system.
	relayerPrivateKey string
}
