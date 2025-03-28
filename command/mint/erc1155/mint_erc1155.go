package mint

import (
	"fmt"

	"github.com/0xPolygon/polygon-edge/command"
	bridgeHelper "github.com/0xPolygon/polygon-edge/command/bridge/helper"

	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/spf13/cobra"
)

var (
	params mintParams
)

// GetCommand returns the external chain fund command
func GetCommand() *cobra.Command {
	mintCmd := &cobra.Command{
		Use:     "mint-erc1155",
		Short:   "Mints ERC1155 tokens to specified address",
		PreRunE: preRunCommand,
		Run:     runCommand,
	}

	helper.RegisterJSONRPCFlag(mintCmd)

	setFlags(mintCmd)

	return mintCmd
}

func preRunCommand(cmd *cobra.Command, _ []string) error {
	params.jsonRPCAddress = helper.GetJSONRPCAddress(cmd)

	return params.validateFlags()
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&params.tokenAddr,
		bridgeHelper.TokenFlag,
		"",
		"erc1155 token address",
	)

	_ = cmd.MarkFlagRequired(bridgeHelper.TokenFlag)

	cmd.Flags().StringVar(
		&params.minterPrivateKey,
		bridgeHelper.PrivateKeyFlag,
		"",
		"minter's private key",
	)

	_ = cmd.MarkFlagRequired("minter")

	cmd.Flags().StringVar(
		&params.address,
		bridgeHelper.AddressesFlag,
		"",
		"address for which tokens are minted",
	)

	_ = cmd.MarkFlagRequired("address")

	cmd.Flags().StringSliceVar(
		&params.tokens,
		"tokens",
		nil,
		"tokens being minted (their IDs)",
	)

	_ = cmd.MarkFlagRequired("tokens")

	cmd.Flags().StringSliceVar(
		&params.amounts,
		"amounts",
		nil,
		"amounts being minted (matching the order of tokens)",
	)

	_ = cmd.MarkFlagRequired("amounts")

	cmd.Flags().DurationVar(
		&params.txTimeout,
		helper.TxTimeoutFlag,
		txrelayer.DefaultTimeoutTransactions,
		helper.TxTimeoutDesc,
	)
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(params.jsonRPCAddress),
		txrelayer.WithReceiptsTimeout(params.txTimeout))
	if err != nil {
		outputter.SetError(fmt.Errorf("failed to initialize tx relayer: %w", err))

		return
	}

	deployerKey, err := bridgeHelper.DecodePrivateKey(params.minterPrivateKey)
	if err != nil {
		outputter.SetError(fmt.Errorf("failed to initialize deployer private key: %w", err))

		return
	}

	tokenAddr := types.StringToAddress(params.tokenAddr)

	// mint tokens to address
	addr := types.StringToAddress(params.address)

	mintTxn, err := bridgeHelper.CreateMintERC1155Txn(addr, params.tokensValues, params.amountValues, tokenAddr, false)
	if err != nil {
		outputter.SetError(fmt.Errorf("failed to create mint ERC1155 tokens transaction for '%s'. err: %w",
			addr, err))

		return
	}

	receipt, err := txRelayer.SendTransaction(mintTxn, deployerKey)
	if err != nil {
		outputter.SetError(fmt.Errorf("failed to send mint ERC1155 tokens transaction for '%s'. err: %w", addr, err))

		return
	}

	if receipt.Status == uint64(types.ReceiptFailed) {
		outputter.SetError(fmt.Errorf("failed to mint ERC1155 tokens for '%s'", addr))

		return
	}

	outputter.SetCommandResult(&mintResult{
		Address: addr,
		TxHash:  types.Hash(receipt.TransactionHash),
	})
}
