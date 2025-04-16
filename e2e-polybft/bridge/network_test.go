package bridge

import (
	"context"
	"fmt"
	"math/big"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/bridge/common"
	bridgeHelper "github.com/0xPolygon/polygon-edge/command/bridge/helper"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// TestE2E_Bridge_NetworkFailureAndRestart is an end-to-end test for the bridge functionality
// in a PolyBFT network. It simulates a network failure and restart scenario to ensure the
// bridge operates correctly under such conditions. The test performs the following steps:
//
//   - Initializes a test cluster with a specified number of validators, bridges, and other configurations.
//   - Generates accounts and keys for token transfers.
//   - Deploys and mints ERC20 tokens on both internal and external chains.
//   - Deposits ERC20 tokens using the bridge and verifies the deposits.
//   - Simulates a network failure by stopping validators and waits for a block threshold to be reached.
//   - Restarts the validators and relayer, then performs additional deposits to verify bridge functionality.
//   - Ensures all events are processed correctly and validates the final token balances on both chains.
//
// The test ensures that the bridge can handle network disruptions and continue processing
// transactions correctly after the network is restored.
func TestE2E_Bridge_NetworkFailureAndRestart(t *testing.T) {
	const (
		validatorsCount = 4
		transfersCount  = 1
		epochSize       = 10
		numberOfBridges = 1
		bridgeAmount    = 100
		bridgeAmountStr = "100"
		threshold       = 10
	)

	accountAddrs := make([]types.Address, transfersCount)
	accounts := make([]string, transfersCount)
	accountKeys := make([]string, transfersCount)

	for i := 0; i < transfersCount; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		rawKey, err := key.MarshallPrivateKey()
		require.NoError(t, err)

		accountKeys[i] = hex.EncodeToString(rawKey)
		accountAddrs[i] = key.Address()
		accounts[i] = key.Address().String()

		t.Logf("Receiver#%d=%s\n", i+1, accounts[i])
	}

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, validatorsCount,
		framework.WithBridges(numberOfBridges),
		framework.WithEpochSize(epochSize),
		framework.WithBridgeBatchThreshold(threshold),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))

			tcc.Premine = append(tcc.Premine, accounts...)
			tcc.Premine = append(tcc.Premine, relayerPrivateKey.String())
		}),
	)

	defer cluster.Stop()

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	internalJSONRPCAddr := cluster.Servers[0].JSONRPCAddr()
	internalEndpoint := cluster.Servers[0].JSONRPC()

	externalJSONRPCAddr := cluster.Bridges[0].JSONRPCAddr()
	externalEndpoint, err := jsonrpc.NewEthClient(externalJSONRPCAddr)
	require.NoError(t, err)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(externalEndpoint))
	require.NoError(t, err)

	internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(internalEndpoint))
	require.NoError(t, err)

	externalChainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[externalChainID.Uint64()]

	bridge := cluster.Bridges[0]

	deployerKey, err := bridgeHelper.DecodePrivateKey("")
	require.NoError(t, err)

	stopRelayerFn := func(startID, endID uint64) {
		// Minimal time needed for sleep before stopping is sprint time + relayer period = 15s
		require.NoError(t, cluster.WaitUntil(20*time.Second, 2*time.Second, func() bool {
			for i := startID; i <= endID; i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		cluster.BridgeRelayers[0].Stop()
	}

	var internalERC20Addr, externalERC20Addr types.Address

	mint := func(token types.Address, amount *big.Int, account types.Address, relayer txrelayer.TxRelayer) error {
		mintFn := &contractsapi.MintRootERC20Fn{
			To:     account,
			Amount: amount,
		}

		mintInput, err := mintFn.EncodeAbi()
		if err != nil {
			return err
		}

		mintTx := types.NewTx(types.NewLegacyTx(
			types.WithTo(&token),
			types.WithInput(mintInput),
		))

		receipt, err := relayer.SendTransaction(mintTx, deployerKey)
		if err != nil {
			return err
		}

		if receipt == nil || receipt.Status != uint64(types.ReceiptSuccess) {
			return fmt.Errorf("mint failed")
		}

		return nil
	}

	deployAndMint := func(relayer txrelayer.TxRelayer, erc20Addr *types.Address) error {
		// deploy erc20
		deployTx := types.NewTx(types.NewLegacyTx(
			types.WithTo(nil),
			types.WithInput(contractsapi.RootERC20.Bytecode),
		))

		receipt, err := relayer.SendTransaction(deployTx, deployerKey)
		require.NoError(t, err)
		require.NotNil(t, receipt)
		require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

		*erc20Addr = types.Address(receipt.ContractAddress)

		// mint erc20
		for _, acc := range accountAddrs {
			if err := mint(*erc20Addr, big.NewInt(bridgeAmount*3), acc, relayer); err != nil {
				return err
			}
		}

		return nil
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	g, _ := errgroup.WithContext(timeoutCtx)

	g.Go(func() error {
		return deployAndMint(internalChainTxRelayer, &internalERC20Addr)
	})

	g.Go(func() error {
		return deployAndMint(externalChainTxRelayer, &externalERC20Addr)
	})

	if err := g.Wait(); err != nil {
		t.Fatalf("failed to deploy and mint ERC20: %v", err)
	}

	cancel()

	t.Logf("Deployed ERC20 contracts & minted tokens")

	runTest := func(erc20Addr, predicateAddr types.Address, rpcAddr string) error {
		// deposit erc20
		for i := range transfersCount {
			if err := bridge.Deposit(
				common.ERC20,
				erc20Addr,
				predicateAddr,
				accountKeys[i],
				accounts[i],
				bridgeAmountStr,
				"",
				rpcAddr,
				"",
				false,
			); err != nil {
				return err
			}
		}

		return nil
	}

	depositBothSides := func() {
		// first deposit
		timeoutCtx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		g, _ = errgroup.WithContext(timeoutCtx)

		g.Go(func() error {
			return runTest(internalERC20Addr, bridgeCfg.InternalMintableERC20PredicateAddr, internalJSONRPCAddr)
		})

		g.Go(func() error {
			return runTest(externalERC20Addr, bridgeCfg.ExternalERC20PredicateAddr, externalJSONRPCAddr)
		})

		if err := g.Wait(); err != nil {
			t.Fatalf("failed to deposit ERC20: %v", err)
		}

		cancel()
	}

	depositBothSides()

	stopRelayerFn(1, 2)

	t.Logf("Deposited first & stopped relayer")

	depositBothSides()

	t.Logf("Deposited ERC20 tokens")

	{
		// wait for batch to be created
		require.NoError(t, cluster.WaitUntil(3*time.Minute, 2*time.Second, func() bool {
			latest, err := internalEndpoint.BlockNumber()
			require.NoError(t, err)

			logs, err := getFilteredLogs((&contractsapi.NewBatchEvent{}).Sig(), 0, latest, internalEndpoint)
			require.NoError(t, err)

			if len(logs) < 2 {
				return false
			}

			return true
		}))

		wg := sync.WaitGroup{}
		wg.Add(validatorsCount)

		// stop validators
		for i := range validatorsCount {
			go func(validatorNum int) {
				defer wg.Done()

				t.Logf("Stopping validator %d", validatorNum)

				cluster.Servers[validatorNum].Stop()
			}(i)
		}

		wg.Wait()

		// wait to reach threshold block
		block := waitForBlocksOnExternal(t, threshold+1, externalEndpoint, 3*time.Minute)

		t.Logf("Reached threshold block %d", block)

		wg.Add(validatorsCount)

		// start validators
		for i := range validatorsCount {
			go func(validatorNum int) {
				defer wg.Done()

				t.Logf("Starting validator %d", validatorNum)

				cluster.Servers[validatorNum].Start()
			}(i)
		}

		wg.Wait()
	}

	// start relayer
	cluster.BridgeRelayers[0].Start()

	t.Logf("Restarted validators & relayer")

	depositBothSides()

	t.Logf("Deposited ERC20 tokens after validators restart")

	require.NoError(t, cluster.WaitUntil(3*time.Minute, 2*time.Second, func() bool {
		for i := uint64(1); i <= 3*transfersCount+1; i++ {
			if !isEventProcessed(t, bridgeCfg.InternalGatewayAddr, internalChainTxRelayer, i, false) {
				return false
			}

			if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
				return false
			}
		}

		return true
	}))

	t.Logf("Events processed")

	internalChildToken := getChildToken(t, contractsapi.RootERC20Predicate.Abi,
		bridgeCfg.InternalMintableERC20PredicateAddr, internalERC20Addr, internalChainTxRelayer)
	externalChildToken := getChildToken(t, contractsapi.RootERC20Predicate.Abi,
		bridgeCfg.ExternalERC20PredicateAddr, externalERC20Addr, externalChainTxRelayer)

	expectedBalance := big.NewInt(bridgeAmount * 3)

	for _, acc := range accountAddrs {
		balance1 := erc20BalanceOf(t, acc, internalChildToken, externalChainTxRelayer)
		if balance1.Cmp(expectedBalance) != 0 {
			t.Fatalf("expected balance=%d, got=%d", expectedBalance, balance1)
		}

		balance2 := erc20BalanceOf(t, acc, externalChildToken, internalChainTxRelayer)
		if balance2.Cmp(expectedBalance) != 0 {
			t.Fatalf("expected balance=%d, got=%d", expectedBalance, balance2)
		}
	}

	t.Logf("Test passed")
}
