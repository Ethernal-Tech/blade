package bridge

import (
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strings"
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
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	parent := filepath.Dir(wd)
	parent = strings.Trim(parent, "e2e-polybft")
	wd = filepath.Join(parent, "/artifacts/blade")
	os.Setenv("EDGE_BINARY", wd)
	os.Setenv("E2E_TESTS", "true")
	os.Setenv("E2E_LOGS", "true")
	os.Setenv("E2E_LOG_LEVEL", "debug")
}

func TestE2E_Bridge_NetworkFailureAndRestart(t *testing.T) {
	const (
		validatorsCount = 4
		transfersCount  = 5
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

	// stop relayer until
	relayer := cluster.BridgeRelayers[0]
	relayer.Stop()

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

	errChan := make(chan error, 1)

	deployAndMint := func(relayer txrelayer.TxRelayer, erc20Addr *types.Address) {
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
			if err := mint(*erc20Addr, big.NewInt(bridgeAmount*2), acc, relayer); err != nil {
				errChan <- err

				return
			}
		}

		errChan <- nil
	}

	go deployAndMint(internalChainTxRelayer, &internalERC20Addr)
	go deployAndMint(externalChainTxRelayer, &externalERC20Addr)

	counter := 0

loop:
	for {
		select {
		case err := <-errChan:
			require.NoError(t, err)

			if counter++; counter == 2 {
				break loop
			}
		case <-time.After(5 * time.Minute):
			t.Fatal("timeout")

			return
		}
	}

	t.Logf("Deployed ERC20 contracts & minted tokens")

	runTest := func(erc20Addr, predicateAddr types.Address, rpcAddr string) {
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
				errChan <- err

				return
			}
		}
		errChan <- nil
	}

	go runTest(internalERC20Addr, bridgeCfg.InternalMintableERC20PredicateAddr, internalJSONRPCAddr)
	go runTest(externalERC20Addr, bridgeCfg.ExternalERC20PredicateAddr, externalJSONRPCAddr)

	counter = 0

loop2:
	for {
		select {
		case err := <-errChan:
			require.NoError(t, err)

			if counter++; counter == 2 {
				break loop2
			}

		case <-time.After(5 * time.Minute):
			t.Fatal("timeout")

			return
		}
	}

	t.Logf("Deposited ERC20 tokens")

	currentBlock, err := internalEndpoint.BlockNumber()
	require.NoError(t, err)

	t.Logf("Current start block: %d", currentBlock)

	wg := sync.WaitGroup{}
	wg.Add(validatorsCount)

	thresholdChan := make(chan interface{}, 1)

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		timeout := time.After(5 * time.Minute)

		for {
			select {
			case <-ticker.C:
				block, err := internalEndpoint.BlockNumber()
				require.NoError(t, err)

				t.Logf("Current block: %d", block)

				if block >= currentBlock+threshold {
					close(thresholdChan)
					errChan <- nil

					return
				}
			case <-timeout:
				close(thresholdChan)
				errChan <- fmt.Errorf("timeout waiting for block %d", currentBlock+threshold)

				return
			}
		}
	}()

	for i := range validatorsCount {
		go func(validatorNum int) {
			defer wg.Done()
			<-thresholdChan

			t.Logf("Stopping validator %d", validatorNum)

			defer cluster.Servers[validatorNum].Start()
			cluster.Servers[validatorNum].Stop()

			time.Sleep(10 * time.Second)
		}(i)
	}

	if err := <-errChan; err != nil {
		t.Fatal(err)

		return
	}

	wg.Wait()

	// start relayer again
	relayer.Start()

	t.Logf("Restarted validators")

	go runTest(internalERC20Addr, bridgeCfg.InternalMintableERC20PredicateAddr, internalJSONRPCAddr)
	go runTest(externalERC20Addr, bridgeCfg.ExternalERC20PredicateAddr, externalJSONRPCAddr)

	counter = 0

loop3:
	for {
		select {
		case err := <-errChan:
			require.NoError(t, err)

			if counter++; counter == 2 {
				break loop3
			}

		case <-time.After(5 * time.Minute):
			t.Fatal("timeout")

			return
		}
	}

	t.Logf("Deposited ERC20 tokens after validators restart")

	require.NoError(t, cluster.WaitUntil(3*time.Minute, 2*time.Second, func() bool {
		for i := uint64(1); i <= 2*transfersCount+1; i++ {
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

	expectedBalance := big.NewInt(bridgeAmount * 2)

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
