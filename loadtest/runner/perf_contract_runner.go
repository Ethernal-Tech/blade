package runner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
)

type PerfContractResultsCollector struct {
	ConfirmedBatchesCountCh chan int
	ConfirmedBatchesErrCh   chan error
	ConfirmedBatchesCount   int
	ConfirmedBatchesErrors  []error

	HashesCountCh chan *big.Int
	HashesErrCh   chan error
	HashesCount   *big.Int
	HashesErrors  []error

	LastBatchIDCh     chan *big.Int
	LastBatchIDErrCh  chan error
	LastBatchID       *big.Int
	LastBatchIDErrors []error
}

// NewPerfContractResultsCollector creates a new PerfContractResultsCollector instance.
func NewPerfContractResultsCollector() *PerfContractResultsCollector {
	return &PerfContractResultsCollector{
		ConfirmedBatchesCountCh: make(chan int, 3000),
		ConfirmedBatchesErrCh:   make(chan error, 3000),
		HashesCountCh:           make(chan *big.Int, 3000),
		HashesErrCh:             make(chan error, 3000),
		LastBatchIDCh:           make(chan *big.Int, 3000),
		LastBatchIDErrCh:        make(chan error, 3000),
		LastBatchID:             new(big.Int),
		HashesCount:             new(big.Int),
	}
}

// CollectResults collects the results of the load test.
func (p *PerfContractResultsCollector) CollectResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case count := <-p.ConfirmedBatchesCountCh:
			p.ConfirmedBatchesCount = count
		case err := <-p.ConfirmedBatchesErrCh:
			p.ConfirmedBatchesErrors = append(p.ConfirmedBatchesErrors, err)
		case count := <-p.HashesCountCh:
			if p.HashesCount.Cmp(count) < 0 {
				p.HashesCount = count
			}
		case err := <-p.HashesErrCh:
			p.HashesErrors = append(p.HashesErrors, err)
		case lastBatchID := <-p.LastBatchIDCh:
			if p.LastBatchID.Cmp(lastBatchID) < 0 {
				p.LastBatchID = lastBatchID
			}
		case err := <-p.LastBatchIDErrCh:
			p.LastBatchIDErrors = append(p.LastBatchIDErrors, err)
		}
	}
}

// PrintResults prints the results of the load test.
func (p *PerfContractResultsCollector) PrintResults() {
	fmt.Println("====================================")
	fmt.Println("Total number of confirmed batches", p.ConfirmedBatchesCount)
	fmt.Println("Total number of hashes", p.HashesCount.String())
	fmt.Println("Last batch ID", p.LastBatchID.String())

	if len(p.ConfirmedBatchesErrors) > 0 {
		fmt.Println("====================================")
		fmt.Println("Confirmed batches read errors:")

		for i, err := range p.ConfirmedBatchesErrors {
			fmt.Printf("%d: %v\n", i, err)
		}
	}

	if len(p.HashesErrors) > 0 {
		fmt.Println("====================================")
		fmt.Println("Hashes read errors:")

		for i, err := range p.HashesErrors {
			fmt.Printf("%d: %v\n", i, err)
		}
	}

	if len(p.LastBatchIDErrors) > 0 {
		fmt.Println("====================================")
		fmt.Println("Last batch ID read errors:")

		for i, err := range p.LastBatchIDErrors {
			fmt.Printf("%d: %v\n", i, err)
		}
	}
}

// PerfContractRunner represents a load test runner for performance test contract.
type PerfContractRunner struct {
	*BaseLoadTestRunner

	perfResultCollector *PerfContractResultsCollector

	contractAddr     types.Address
	contractArtifact *contracts.Artifact

	getConfirmedBatchesInput []byte
	getHashesCountInput      []byte
	getLastBatchIDInput      []byte
}

// NewPerfContractRunner creates a new PerfContractRunner instance with the given LoadTestConfig.
// It returns a pointer to the created PerfContractRunner and an error, if any.
func NewPerfContractRunner(cfg LoadTestConfig) (*PerfContractRunner, error) {
	runner, err := NewBaseLoadTestRunner(cfg)
	if err != nil {
		return nil, err
	}

	return &PerfContractRunner{
		BaseLoadTestRunner:  runner,
		perfResultCollector: NewPerfContractResultsCollector(),
	}, nil
}

func (p *PerfContractRunner) Run(ctx context.Context) error {
	fmt.Println("Running PerfContract load test", p.cfg.LoadTestName)

	// print state db metrics before and after test
	p.printStateDBMetrics()
	defer p.printStateDBMetrics()

	if err := p.createVUs(); err != nil {
		return err
	}

	if err := p.fundVUs(); err != nil {
		return err
	}

	if err := p.deployPerfContract(); err != nil {
		return fmt.Errorf("failed to deploy performance test contract: %w", err)
	}

	if err := p.getFunctionsInput(); err != nil {
		return fmt.Errorf("failed to get functions input: %w", err)
	}

	cancelableCtx, cancel := context.WithCancel(ctx)

	defer func() {
		cancel()

		p.resultsCollector.PrintResults()
		p.perfResultCollector.PrintResults()
	}()

	go p.resultsCollector.CollectResults(ctx)
	go p.perfResultCollector.CollectResults(ctx)
	go p.readState(cancelableCtx)
	go p.readTxPool(cancelableCtx)

	if !p.cfg.WaitForTxPoolToEmpty {
		go p.waitForReceiptsParallel(cancelableCtx)
		go p.calculateResultsParallel()

		_, err := p.sendTransactions(p.createPerfContractTransaction)
		if err != nil {
			return err
		}

		if err := <-p.done; err != nil {
			return err
		}

		nodeInfos, err := p.queryLatestBlocks()
		if err != nil {
			return err
		}

		return p.printNodeInfos(nodeInfos)
	}

	txHashes, err := p.sendTransactions(p.createPerfContractTransaction)
	if err != nil {
		return err
	}

	if err := p.waitForTxPoolToEmpty(); err != nil {
		return err
	}

	if err := p.calculateResults(p.waitForReceipts(txHashes)); err != nil {
		return err
	}

	nodeInfos, err := p.queryLatestBlocks()
	if err != nil {
		return err
	}

	if err := p.tearDown(); err != nil {
		return err
	}

	return p.printNodeInfos(nodeInfos)
}

// createPerfContractTransaction creates a performance test contract transaction.
func (p *PerfContractRunner) createPerfContractTransaction(
	account *account, feeData *feeData, chainID *big.Int) (*types.Transaction, error) {
	input := &contractsapi.SubmitSignedBatchTestPerformanceFn{
		SignedBatch: &contractsapi.SignedBatch{
			BatchID:     new(big.Int).SetUint64(account.nonce),
			Counter:     new(big.Int).SetUint64(account.nonce),
			ValidatorID: new(big.Int).SetUint64(uint64(account.index)),
			Signature:   []byte(fmt.Sprintf("validator-%d", account.index)),
		},
	}

	txInput, err := input.EncodeAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to encode submitSignedBatch function: %w", err)
	}

	if p.cfg.DynamicTxs {
		return types.NewTx(types.NewDynamicFeeTx(
			types.WithNonce(account.nonce),
			types.WithTo(&p.contractAddr),
			types.WithFrom(account.key.Address()),
			types.WithGasFeeCap(feeData.gasFeeCap),
			types.WithGasTipCap(feeData.gasTipCap),
			types.WithChainID(chainID),
			types.WithInput(txInput),
		)), nil
	}

	return types.NewTx(types.NewLegacyTx(
		types.WithNonce(account.nonce),
		types.WithTo(&p.contractAddr),
		types.WithGasPrice(feeData.gasPrice),
		types.WithFrom(account.key.Address()),
		types.WithInput(txInput),
	)), nil
}

// deployPerfContract deploys the performance test contract.
func (p *PerfContractRunner) deployPerfContract() error {
	fmt.Println("=============================================================")
	fmt.Println("Deploying performance test contract")

	start := time.Now().UTC()
	artifact := contractsapi.TestPerformance

	quorum := big.NewInt(int64(p.cfg.VUs / 3))

	fmt.Println("Quorum count:", quorum)

	input := &contractsapi.TestPerformanceConstructorFn{
		QuorumCnt:                          quorum,
		CheckBatchID:                       false,
		DeleteTemporaryMappingsAfterQuorum: false,
	}

	raw, err := input.EncodeAbi()
	if err != nil {
		return err
	}

	txn := types.NewTx(types.NewLegacyTx(
		types.WithTo(nil),
		types.WithInput(append(artifact.Bytecode, raw...)),
		types.WithFrom(p.loadTestAccount.key.Address()),
	))

	txRelayer, err := txrelayer.NewTxRelayer(
		txrelayer.WithClient(p.clients.getClient()),
		txrelayer.WithReceiptsTimeout(p.cfg.ReceiptsTimeout))
	if err != nil {
		return err
	}

	receipt, err := txRelayer.SendTransaction(txn, p.loadTestAccount.key)
	if err != nil {
		return err
	}

	if receipt == nil || receipt.Status == uint64(types.ReceiptFailed) {
		return fmt.Errorf("failed to deploy performance test contract")
	}

	p.contractAddr = types.Address(receipt.ContractAddress)
	p.contractArtifact = artifact

	fmt.Printf("Deploying performance test contract finished in %s\n", time.Since(start))

	return nil
}

// getFunctionsInput encodes the input for the functions of the performance test contract.
func (p *PerfContractRunner) getFunctionsInput() error {
	var err error

	p.getConfirmedBatchesInput, err = (&contractsapi.GetConfirmedBatchesTestPerformanceFn{}).EncodeAbi()
	if err != nil {
		return fmt.Errorf("failed to encode getConfirmedBatches function: %w", err)
	}

	p.getHashesCountInput, err = (&contractsapi.GetHashesCountTestPerformanceFn{}).EncodeAbi()
	if err != nil {
		return fmt.Errorf("failed to encode getHashesCount function: %w", err)
	}

	p.getLastBatchIDInput, err = (&contractsapi.GetLastBatchIDTestPerformanceFn{}).EncodeAbi()
	if err != nil {
		return fmt.Errorf("failed to encode getLastBatchID function: %w", err)
	}

	return nil
}

// readState continuously reads nonce and balance from blockchain
// for each account, with a max of StateReadThreads concurrent workers.
func (p *PerfContractRunner) readState(ctx context.Context) {
	if p.cfg.StateReadThreads == 0 {
		return
	}

	contractMap := contracts.GetProxyImplementationMapping()

	for i := 0; i < p.cfg.StateReadThreads; i++ {
		i := i

		go func() {
			client := p.clients.getClientForAccount(i)

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// read non stop the state of the accounts and contracts
					p.readBasicState(client, contractMap)
					p.readConfirmedBatchesCount(client)
					p.readHashesCount(client)
					p.readLastBatchID(client)
				}
			}
		}()
	}
}

// readLastBatchID reads the last batch ID from the performance test contract.
func (p *PerfContractRunner) readLastBatchID(client *jsonrpc.EthClient) {
	response, err := client.Call(&jsonrpc.CallMsg{
		From: p.loadTestAccount.key.Address(),
		To:   &p.contractAddr,
		Data: p.getLastBatchIDInput,
	}, jsonrpc.LatestBlockNumber, nil)
	if err != nil {
		p.perfResultCollector.LastBatchIDErrCh <- err

		return
	}

	count, err := common.ParseUint256orHex(&response)
	if err != nil {
		p.perfResultCollector.LastBatchIDErrCh <- fmt.Errorf("failed to parse uint256 response, %w", err)
	}

	p.perfResultCollector.LastBatchIDCh <- count
}

// readHashesCount reads the number of hashes from the performance test contract.
func (p *PerfContractRunner) readHashesCount(client *jsonrpc.EthClient) {
	response, err := client.Call(&jsonrpc.CallMsg{
		From: p.loadTestAccount.key.Address(),
		To:   &p.contractAddr,
		Data: p.getHashesCountInput,
	}, jsonrpc.LatestBlockNumber, nil)
	if err != nil {
		p.perfResultCollector.HashesErrCh <- err

		return
	}

	count, err := common.ParseUint256orHex(&response)
	if err != nil {
		p.perfResultCollector.HashesErrCh <- fmt.Errorf("failed to parse uint256 response, %w", err)
	}

	p.perfResultCollector.HashesCountCh <- count
}

// readConfirmedBatchesCount reads the number of confirmed batches from the performance test contract.
func (p *PerfContractRunner) readConfirmedBatchesCount(client *jsonrpc.EthClient) {
	response, err := client.Call(&jsonrpc.CallMsg{
		From: p.loadTestAccount.key.Address(),
		To:   &p.contractAddr,
		Data: p.getConfirmedBatchesInput,
	}, jsonrpc.LatestBlockNumber, nil)
	if err != nil {
		p.perfResultCollector.ConfirmedBatchesErrCh <- err

		return
	}

	byteResponse, err := hex.DecodeHex(response)
	if err != nil {
		p.perfResultCollector.ConfirmedBatchesErrCh <- fmt.Errorf("unable to decode hex response, %w", err)
	}

	decoded, err := p.contractArtifact.Abi.Methods["getConfirmedBatches"].Outputs.Decode(byteResponse)
	if err != nil {
		p.perfResultCollector.ConfirmedBatchesErrCh <- fmt.Errorf("failed to decode getConfirmedBatches response, %w", err)
	}

	decodedMap, ok := decoded.(map[string]interface{})
	if !ok {
		p.perfResultCollector.ConfirmedBatchesErrCh <- fmt.Errorf("failed to convert decoded response to map, %w", err)
	}

	if len(decodedMap) == 0 {
		return
	}

	decodedBatches, ok := decodedMap["0"].([]map[string]interface{})
	if !ok {
		p.perfResultCollector.ConfirmedBatchesErrCh <- fmt.Errorf("failed to convert decoded batches to map, %w", err)
	}

	p.perfResultCollector.ConfirmedBatchesCountCh <- len(decodedBatches)
}
