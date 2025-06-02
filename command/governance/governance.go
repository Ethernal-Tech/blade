package governance

import (
	"time"

	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/0xPolygon/polygon-edge/command/loadtest"
	"github.com/0xPolygon/polygon-edge/external-tests/governance"
	"github.com/spf13/cobra"
)

var (
	params governanceParams
)

func GetCommand() *cobra.Command {
	loadTestCmd := &cobra.Command{
		Use:     "governance",
		Short:   "Runs governance tests on a specified network",
		PreRunE: preRunCommand,
		Run:     runCommand,
	}

	helper.RegisterJSONRPCFlag(loadTestCmd)

	setFlags(loadTestCmd)

	return loadTestCmd
}

func preRunCommand(cmd *cobra.Command, _ []string) error {
	params.jsonRPCAddress = helper.GetJSONRPCAddress(cmd)

	return params.validateFlags()
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&params.mnemonic,
		loadtest.MnemonicFlag,
		"",
		"the mnemonic used to fund accounts if needed for the governance test",
	)

	cmd.Flags().Uint64Var(
		&params.epochSize,
		epochSizeFlag,
		10,
		"epoch size on the network for which the governance test is run",
	)

	cmd.Flags().DurationVar(
		&params.receiptsTimeout,
		loadtest.ReceiptsTimeoutFlag,
		30*time.Second,
		"the timeout for waiting for transaction receipts",
	)

	cmd.Flags().BoolVar(
		&params.toJSON,
		loadtest.SaveToJSONFlag,
		false,
		"saves results to JSON file",
	)

	cmd.Flags().StringSliceVar(
		&params.validatorKeys,
		validatorKeysFlag,
		nil,
		"private keys of validators on the network for which governance tests is run",
	)

	_ = cmd.MarkFlagRequired(loadtest.MnemonicFlag)
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	governanceRunner, err := governance.NewGovernanceTestRunner(
		&governance.GovernanceTestConfig{
			Mnemonic:        params.mnemonic,
			JSONRPCUrl:      params.jsonRPCAddress,
			ReceiptsTimeout: params.receiptsTimeout,
			EpochSize:       params.epochSize,
			ValidatorKeys:   params.validatorKeys,
			ResultsToJSON:   params.toJSON,
		},
	)

	if err != nil {
		outputter.SetError(err)

		return
	}

	if err = governanceRunner.Run(); err != nil {
		outputter.SetError(err)
	}
}
