package relayer

import (
	bridgerelayer "github.com/0xPolygon/polygon-edge/bridge-relayer"
	"github.com/spf13/cobra"
)

var params commandParams

// GetCommand returns a new bridge relayer command of type [*cobra.Command].
func GetCommand() *cobra.Command {
	relayerCmd := &cobra.Command{
		Use:   "bridge-relayer",
		Short: "Bridge Relayer command starts new bridge relayer responsible for cross-chain asset transfers.",
		Run:   runCommand,
	}

	setFlags(relayerCmd)

	return relayerCmd
}

func runCommand(*cobra.Command, []string) {
	relayer, _ := bridgerelayer.NewBridgeRelayer(params.genesisPath,
		bridgerelayer.WithExternalChainID(params.externalChainID),
		bridgerelayer.WithGenesisPath(params.genesisPath),
	)

	relayer.Start()
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(
		&params.internalChainRPC,
		"internal-chain-rpc",
		"i",
		"",
		"internal chain rpc endpoint",
	)

	_ = cmd.MarkFlagRequired("internal-chain-rpc")

	cmd.Flags().StringVarP(
		&params.internalChainRPC,
		"genesis-path",
		"g",
		"./genesis.json",
		"internal chain genesis path",
	)

	cmd.Flags().StringVarP(
		&params.internalChainRPC,
		"external-chain-id",
		"c",
		"",
		"external chain id",
	)

	_ = cmd.MarkFlagRequired("external-chain-id")

	cmd.Flags().StringVarP(
		&params.internalChainRPC,
		"poll-interval",
		"p",
		"",
		"poll interval",
	)

	cmd.Flags().StringVarP(
		&params.relayerPrivateKey,
		"private-key",
		"k",
		"",
		"relayer's private key",
	)
}
