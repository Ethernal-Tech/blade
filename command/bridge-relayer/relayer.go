package relayer

import (
	"fmt"

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
	relayer, err := bridgerelayer.NewBridgeRelayer(params.internalChainRPC,
		bridgerelayer.WithExternalChainID(uint64(params.externalChainID)),
		bridgerelayer.WithGenesisPath(params.genesisPath),
		bridgerelayer.WithPrivateKey(params.relayerPrivateKey),
	)

	if err != nil {
		fmt.Println(err)

		return
	}

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
		&params.genesisPath,
		"genesis-path",
		"g",
		"./genesis.json",
		"internal chain genesis path",
	)

	cmd.Flags().IntVarP(
		&params.externalChainID,
		"external-chain-id",
		"c",
		1,
		"external chain id",
	)

	_ = cmd.MarkFlagRequired("external-chain-id")

	cmd.Flags().IntVarP(
		&params.pollInterval,
		"poll-interval",
		"p",
		10,
		"poll interval",
	)

	cmd.Flags().StringVarP(
		&params.relayerPrivateKey,
		"private-key",
		"k",
		"",
		"relayer's private key",
	)

	_ = cmd.MarkFlagRequired("private-key")
}
