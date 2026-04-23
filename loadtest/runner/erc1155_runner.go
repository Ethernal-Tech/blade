package runner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
)

// ERC1155Runner represents a load test runner for ERC1155 tokens.
type ERC1155Runner struct {
	*BaseLoadTestRunner

	erc1155Token         types.Address
	erc1155TokenArtifact *contracts.Artifact
	txInput              []byte
}

// NewERC1155Runner creates a new ERC1155Runner instance with the given LoadTestConfig.
// It returns a pointer to the created ERC1155Runner and an error, if any.
func NewERC1155Runner(cfg LoadTestConfig) (*ERC1155Runner, error) {
	runner, err := NewBaseLoadTestRunner(cfg)
	if err != nil {
		return nil, err
	}

	return &ERC1155Runner{BaseLoadTestRunner: runner}, nil
}

// Run executes the ERC1155 load test.
// It performs the following steps:
// 1. Creates virtual users (VUs).
// 2. Funds the VUs with native tokens.
// 3. Deploys the ERC1155 token contract.
// 4. Mints ERC1155 tokens to the VUs.
// 5. Sends transactions using the VUs.
// 6. Waits for the transaction pool to empty.
// 7. Waits for transaction receipts.
// 8. Calculates the transactions per second (TPS) based on block information and transaction statistics.
// Returns an error if any of the steps fail.
//
//nolint:dupl
func (e *ERC1155Runner) Run(ctx context.Context) error {
	fmt.Println("Running ERC1155 load test", e.cfg.LoadTestName)

	// print state db metrics before and after test
	e.printStateDBMetrics()
	defer e.printStateDBMetrics()

	if err := e.createVUs(); err != nil {
		return err
	}

	if err := e.fundVUs(); err != nil {
		return err
	}

	if err := e.deployERC1155Token(); err != nil {
		return err
	}

	if err := e.mintERC1155TokenToVUs(); err != nil {
		return err
	}

	cancelableCtx, cancel := context.WithCancel(ctx)

	defer func() {
		cancel()

		e.resultsCollector.PrintResults()
	}()

	go e.resultsCollector.CollectResults(ctx)
	go e.readState(cancelableCtx)
	go e.readTxPool(cancelableCtx)

	if !e.cfg.WaitForTxPoolToEmpty {
		go e.waitForReceiptsParallel(cancelableCtx)
		go e.calculateResultsParallel()

		_, err := e.sendTransactions(e.createERC1155Transaction)
		if err != nil {
			return err
		}

		if err := <-e.done; err != nil {
			return err
		}

		nodeInfos, err := e.queryLatestBlocks()
		if err != nil {
			return err
		}

		return e.printNodeInfos(nodeInfos)
	}

	txHashes, err := e.sendTransactions(e.createERC1155Transaction)
	if err != nil {
		return err
	}

	if err := e.waitForTxPoolToEmpty(); err != nil {
		return err
	}

	if err := e.calculateResults(e.waitForReceipts(txHashes)); err != nil {
		return err
	}

	nodeInfos, err := e.queryLatestBlocks()
	if err != nil {
		return err
	}

	if err := e.tearDown(); err != nil {
		return err
	}

	return e.printNodeInfos(nodeInfos)
}

// deployERC1155Token deploys an ERC1155 token contract.
// It loads the contract artifact from the specified file path,
// encodes the constructor inputs, creates a new transaction,
// sends the transaction using a transaction relayer,
// and retrieves the deployment receipt.
// If the deployment is successful, it sets the ERC1155 token address
// and artifact in the ERC1155Runner instance.
// Returns an error if any step of the deployment process fails.
func (e *ERC1155Runner) deployERC1155Token() error {
	fmt.Println("=============================================================")
	fmt.Println("Deploying ERC1155 token contract")

	start := time.Now().UTC()
	artifact := contractsapi.ZexERC1155

	txn := types.NewTx(types.NewLegacyTx(
		types.WithTo(nil),
		types.WithInput(artifact.Bytecode),
		types.WithFrom(e.loadTestAccount.key.Address()),
	))

	txRelayer, err := txrelayer.NewTxRelayer(
		txrelayer.WithClient(e.clients.getClient()),
		txrelayer.WithReceiptsTimeout(e.cfg.ReceiptsTimeout))
	if err != nil {
		return err
	}

	receipt, err := txRelayer.SendTransaction(txn, e.loadTestAccount.key)
	if err != nil {
		return err
	}

	if receipt == nil || receipt.Status == uint64(types.ReceiptFailed) {
		return fmt.Errorf("failed to deploy ERC1155 token")
	}

	e.erc1155Token = types.Address(receipt.ContractAddress)
	e.erc1155TokenArtifact = artifact

	input, err := e.erc1155TokenArtifact.Abi.Methods["transfer"].Encode(map[string]interface{}{
		"to":     e.receivers.getReceiver(),
		"id":     big.NewInt(1),
		"amount": big.NewInt(1),
	})
	if err != nil {
		return err
	}

	e.txInput = input

	fmt.Printf("Deploying ERC1155 token took %s\n", time.Since(start))

	return nil
}

// mintERC1155TokenToVUs mints ERC1155 tokens to the specified virtual users (VUs).
// It sends a transfer transaction to each VU's address, minting the specified number of tokens.
// The transaction is sent using a transaction relayer, and the result is checked for success.
// If any error occurs during the minting process, an error is returned.
func (e *ERC1155Runner) mintERC1155TokenToVUs() error {
	fmt.Println("=============================================================")

	start := time.Now().UTC()
	bar := progressbar.Default(int64(e.cfg.VUs), "Minting ERC1155 tokens to VUs")
	client := e.clients.getClient()

	defer func() {
		_ = bar.Close()

		fmt.Printf("Minting ERC1155 tokens took %s\n", time.Since(start))
	}()

	txRelayer, err := txrelayer.NewTxRelayer(
		txrelayer.WithClient(client),
		txrelayer.WithoutNonceGet(),
		txrelayer.WithReceiptsTimeout(e.cfg.ReceiptsTimeout),
	)
	if err != nil {
		return err
	}

	nonce, err := client.GetNonce(e.loadTestAccount.key.Address(), jsonrpc.PendingBlockNumberOrHash)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(context.Background())

	for i, vu := range e.vus {
		i := i
		vu := vu

		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				input, err := e.erc1155TokenArtifact.Abi.Methods["mint"].Encode(map[string]interface{}{
					"to":     vu.key.Address(),
					"id":     big.NewInt(1),
					"amount": big.NewInt(int64(e.cfg.TxsPerUser)),
					"data":   []byte{},
				})
				if err != nil {
					return err
				}

				tx := types.NewTx(types.NewLegacyTx(
					types.WithTo(&e.erc1155Token),
					types.WithInput(input),
					types.WithNonce(nonce+uint64(i)),
					types.WithFrom(e.loadTestAccount.key.Address()),
				))

				receipt, err := txRelayer.SendTransaction(tx, e.loadTestAccount.key)
				if err != nil {
					return err
				}

				if receipt == nil || receipt.Status != uint64(types.ReceiptSuccess) {
					return fmt.Errorf("failed to mint ERC1155 tokens to %s", vu.key.Address())
				}

				_ = bar.Add(1)

				return nil
			}
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// createERC1155Transaction creates an ERC1155 transaction
func (e *ERC1155Runner) createERC1155Transaction(account *account, feeData *feeData,
	chainID *big.Int) (*types.Transaction, error) {
	if e.cfg.DynamicTxs {
		return types.NewTx(types.NewDynamicFeeTx(
			types.WithNonce(account.nonce),
			types.WithTo(&e.erc1155Token),
			types.WithFrom(account.key.Address()),
			types.WithGasFeeCap(feeData.gasFeeCap),
			types.WithGasTipCap(feeData.gasTipCap),
			types.WithChainID(chainID),
			types.WithInput(e.txInput),
		)), nil
	}

	return types.NewTx(types.NewLegacyTx(
		types.WithNonce(account.nonce),
		types.WithTo(&e.erc1155Token),
		types.WithGasPrice(feeData.gasPrice),
		types.WithFrom(account.key.Address()),
		types.WithInput(e.txInput),
	)), nil
}
