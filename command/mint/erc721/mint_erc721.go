package mint

import (
	"fmt"

	"github.com/0xPolygon/polygon-edge/command"
	bridgeHelper "github.com/0xPolygon/polygon-edge/command/bridge/helper"
	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	params mintParams
)

// GetCommand returns the external chain fund command
func GetCommand() *cobra.Command {
	mintCmd := &cobra.Command{
		Use:     "mint-erc721",
		Short:   "Mints ERC721 tokens to specified addresses",
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
		&params.minterPrivateKey,
		bridgeHelper.PrivateKeyFlag,
		"",
		"the minter private key",
	)

	cmd.Flags().StringSliceVar(
		&params.addresses,
		bridgeHelper.AddressesFlag,
		nil,
		"receivers addresses",
	)

	cmd.Flags().StringVar(
		&params.tokenAddr,
		bridgeHelper.TokenFlag,
		"",
		"erc721 token address",
	)

	cmd.Flags().DurationVar(
		&params.txTimeout,
		helper.TxTimeoutFlag,
		txrelayer.DefaultTimeoutTransactions,
		helper.TxTimeoutDesc,
	)

	_ = cmd.MarkFlagRequired(bridgeHelper.TokenFlag)
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

	g, ctx := errgroup.WithContext(cmd.Context())

	results := make([]command.CommandResult, len(params.addresses))
	tokenAddr := types.StringToAddress(params.tokenAddr)

	for i := 0; i < len(params.addresses); i++ {
		i := i

		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()

			default:
				// mint tokens to address
				addr := types.StringToAddress(params.addresses[i])

				mintTxn, err := bridgeHelper.CreateMintERC721Txn(addr, tokenAddr, false)
				if err != nil {
					return fmt.Errorf("failed to create mint native tokens transaction for validator '%s'. err: %w",
						addr, err)
				}

				receipt, err := txRelayer.SendTransaction(mintTxn, deployerKey)
				if err != nil {
					return fmt.Errorf("failed to send mint native tokens transaction to validator '%s'. err: %w", addr, err)
				}

				if receipt.Status == uint64(types.ReceiptFailed) {
					return fmt.Errorf("failed to mint native tokens to validator '%s'", addr)
				}

				results[i] = &mintResult{
					Address: addr,
					TxHash:  types.Hash(receipt.TransactionHash),
				}

				return nil
			}
		})
	}

	if err := g.Wait(); err != nil {
		outputter.SetError(err)
		_, _ = outputter.Write([]byte("[MINT-ERC721] Successfully minted tokens to following accounts\n"))

		for _, result := range results {
			if result != nil {
				// In case an error happened, some of the indices may not be populated.
				// Filter those out.
				outputter.SetCommandResult(result)
			}
		}

		return
	}

	outputter.SetCommandResult(command.Results(results))
}
