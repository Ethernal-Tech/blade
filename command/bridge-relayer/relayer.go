package relayer

import (
	"fmt"

	"github.com/spf13/cobra"
)

var params relayerParams

// GetCommand returns new bridge relayer [*cobra.Command].
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
	fmt.Println("Bridge relayer is running...")
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
		"",
		"internal chain genesis path",
	)

	_ = cmd.MarkFlagRequired("genesis-path")

	cmd.Flags().StringVarP(
		&params.relayerPrivateKey,
		"private-key",
		"k",
		"",
		"relayer's private key",
	)

	_ = cmd.MarkFlagRequired("private-key")
}
