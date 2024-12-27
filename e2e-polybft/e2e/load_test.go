package e2e

import (
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/command/bridge/common"
	bridgeHelper "github.com/0xPolygon/polygon-edge/command/bridge/helper"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	"github.com/0xPolygon/polygon-edge/helper/hex"
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

func TestE2E_Load_MultipleDepositBothEnds(t *testing.T) {
	const (
		transfersCount        = 10
		numBlockConfirmations = 2
		epochSize             = 40
		sprintSize            = uint64(10)
		numberOfAttempts      = 7
		stateSyncedLogsCount  = 2 // map token and deposit
		numberOfBridges       = 1
		numberOfMapTokenEvent = 1
		withRelayerRestart    = true
	)

	var (
		bridgeAmount        = ethgo.Ether(2)
		bridgeMessageResult contractsapi.BridgeMessageResultEvent
	)

	accountAddrs := make([]types.Address, transfersCount)
	accounts := make([]string, transfersCount)
	amounts := make([]string, transfersCount)
	accountKeys := make([]string, transfersCount)

	for i := 0; i < transfersCount; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		rawKey, err := key.MarshallPrivateKey()
		require.NoError(t, err)

		accountKeys[i] = hex.EncodeToString(rawKey)
		accountAddrs[i] = key.Address()
		accounts[i] = key.Address().String()
		amounts[i] = fmt.Sprintf("%d", bridgeAmount)

		t.Logf("Receiver#%d=%s\n", i+1, accounts[i])
	}

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithTestRewardToken(),
		framework.WithNumBlockConfirmations(numBlockConfirmations),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBatchThreshold(100),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				// premine receivers, so that they are able to do withdrawals
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))

			tcc.Premine = append(tcc.Premine, accounts...)
			tcc.Premine = append(tcc.Premine, relayerPrivateKey.String())
		}))

	defer cluster.Stop()

	bridgeOne := 0

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, chainConfigFileName))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]

	validatorEndpoint := validatorSrv.JSONRPC()

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(validatorEndpoint))
	require.NoError(t, err)

	deployerKey, err := bridgeHelper.DecodePrivateKey("")
	require.NoError(t, err)

	deployTx := types.NewTx(types.NewLegacyTx(
		types.WithTo(nil),
		types.WithInput(contractsapi.RootERC20.Bytecode),
	))

	receipt, err := externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

	rootERC20Token := types.Address(receipt.ContractAddress)

	finalBlockNum := 1 * sprintSize
	require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

	bridge := cluster.Bridges[bridgeOne]

	channel := make(chan error)

	// deposit external to internal
	go func(channel chan<- error) {
		for i := range accounts {
			if err := bridge.Deposit(
				common.ERC20,
				rootERC20Token,
				bridgeCfg.ExternalERC20PredicateAddr,
				bridgeHelper.TestAccountPrivKey,
				accounts[i],
				amounts[i],
				"",
				bridge.JSONRPCAddr(),
				bridgeHelper.TestAccountPrivKey,
				false,
			); err != nil {
				channel <- err
				return
			}

			t.Log("deposit made for account=", accounts[i], "external to internal")
		}

		finalBlockNum = 10 * sprintSize
		if err := cluster.WaitForBlock(finalBlockNum, 5*time.Minute); err != nil {
			channel <- err
			return
		}

		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, finalBlockNum, validatorEndpoint)
		if err != nil {
			channel <- err
			return
		}

		assertBridgeEventResultSuccess(t, logs, transfersCount+1)

		childERC20Token := getChildToken(t, contractsapi.RootERC20Predicate.Abi,
			bridgeCfg.ExternalERC20PredicateAddr, rootERC20Token, externalChainTxRelayer)

		for _, receiver := range accounts {
			balance := erc20BalanceOf(t, types.StringToAddress(receiver), childERC20Token, txRelayer)
			t.Log("balance=", balance, "receiver=", receiver)
			if balance.Cmp(bridgeAmount) != 0 {
				channel <- fmt.Errorf("balance=%d, expected=%d", balance, bridgeAmount)
				return
			}
		}

		channel <- nil
	}(channel)

	go func(channel chan<- error) {
		amount := int64(100)
		amountStr := fmt.Sprintf("%d", amount)
		for i := range accounts {
			if err := bridge.Deposit(
				common.ERC20,
				contracts.NativeERC20TokenContract,
				bridgeCfg.InternalMintableERC20PredicateAddr,
				accountKeys[i],
				accounts[i],
				amountStr,
				"",
				validatorSrv.JSONRPCAddr(),
				"",
				false,
			); err != nil {
				channel <- err
				return
			}
			t.Log("deposit made for account=", accounts[i], "internal to external")
		}

		if err := cluster.WaitUntil(time.Minute*5, time.Second*2, func() bool {
			for i := uint64(1); i <= transfersCount+1; i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}); err != nil {
			channel <- err
			return
		}

		l1ChildToken := getChildToken(t, contractsapi.ChildERC20Predicate.Abi, bridgeCfg.ExternalMintableERC20PredicateAddr,
			contracts.NativeERC20TokenContract, externalChainTxRelayer)

		for _, acc := range accountAddrs {
			balance := erc20BalanceOf(t, acc, l1ChildToken, externalChainTxRelayer)
			if balance.Cmp(big.NewInt(amount)) != 0 {
				channel <- fmt.Errorf("balance=%d, expected=%d", balance, amount)
				return
			}
		}

		channel <- nil
	}(channel)

	if withRelayerRestart {
		go func(channel chan<- error) {
			if len(cluster.BridgeRelayers) == 0 {
				channel <- fmt.Errorf("no relayers found")
				return
			}

			time.Sleep(time.Second * 10)
			cluster.BridgeRelayers[0].Stop()
			time.Sleep(time.Second * 10)
			cluster.BridgeRelayers[0].Start()
		}(channel)
	}

	timeChan := time.After(time.Minute * transfersCount)
	for range 2 {
		select {
		case err = <-channel:
			if err != nil {
				t.Fatal(err)
				return
			}

		case <-timeChan:
			t.Fatal("timeout")
		}
	}
}
