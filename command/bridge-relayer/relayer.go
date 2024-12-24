package relayer

import (
	"fmt"

	bridgerelayer "github.com/0xPolygon/polygon-edge/bridge-relayer"
	"github.com/0xPolygon/polygon-edge/command/server/config"
	"github.com/hashicorp/go-hclog"
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
	relayer, err := bridgerelayer.NewBridgeRelayer(params.internalChainRPC, params.relayerPrivateKey,
		bridgerelayer.WithExternalChainID(uint64(params.externalChainID)),
		bridgerelayer.WithGenesisPath(params.genesisPath),
		bridgerelayer.WithLogLevel(hclog.LevelFromString(params.logLevel)),
		bridgerelayer.WithLogJsonFormat(params.jsonFormatOuttputter),
	)

	if err != nil {
		fmt.Println(err)

		return
	}

	relayer.Start()
}

func setFlags(cmd *cobra.Command) {
	defaultConfig := config.DefaultConfig()

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

	cmd.Flags().BoolVarP(
		&params.jsonFormatOuttputter,
		"json",
		"j",
		defaultConfig.JSONLogFormat,
		"logger json format",
	)

	cmd.Flags().StringVarP(
		&params.logLevel,
		"log-level",
		"l",
		defaultConfig.LogLevel,
		"logger level",
	)

	cmd.Flags().StringVar(
		&params.logFilePath,
		"log-path",
		defaultConfig.LogFilePath,
		"write all logs to the file at specified location instead of writing them to console",
	)

	_ = cmd.MarkFlagRequired("private-key")
}
