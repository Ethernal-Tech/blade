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

func TestE2E_TestRollback(t *testing.T) {
	const (
		transfersCount        = 3
		numBlockConfirmations = 2
		// make epoch size long enough, so that all exit events are processed within the same epoch
		epochSize             = 40
		sprintSize            = uint64(5)
		numberOfAttempts      = 7
		stateSyncedLogsCount  = 2 // map token and deposit
		numberOfBridges       = 1
		numberOfMapTokenEvent = 1
	)

	var (
		bridgeAmount        = ethgo.Ether(2)
		bridgeMessageResult contractsapi.BridgeMessageResultEvent
	)

	receiversAddrs := make([]types.Address, transfersCount)
	receivers := make([]string, transfersCount)
	amounts := make([]string, transfersCount)
	receiverKeys := make([]string, transfersCount)

	for i := 0; i < transfersCount; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		rawKey, err := key.MarshallPrivateKey()
		require.NoError(t, err)

		receiverKeys[i] = hex.EncodeToString(rawKey)
		receiversAddrs[i] = key.Address()
		receivers[i] = key.Address().String()
		amounts[i] = fmt.Sprintf("%d", bridgeAmount)

		t.Logf("Receiver#%d=%s\n", i+1, receivers[i])
	}

	cluster := framework.NewTestCluster(t, 5,
		framework.WithTestRewardToken(),
		framework.WithTestRollback(),
		framework.WithNumBlockConfirmations(numBlockConfirmations),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithThreshold(25),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				// premine receivers, so that they are able to do withdrawals
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.Premine = append(tcc.Premine, receivers...)
		}))

	defer cluster.Stop()

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, chainConfigFileName))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]

	childEthEndpoint := validatorSrv.JSONRPC()

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]
	bridge := cluster.Bridges[0]

	// txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
	// require.NoError(t, err)

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
	t.Log("External chain token address:", rootERC20Token)

	// wait for a couple of sprints
	finalBlockNum := 1 * sprintSize
	require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

	t.Run("Bridge_TestE2I", func(t *testing.T) {
		erc20balances := make(map[string]*big.Int)
		for _, receiver := range receivers {
			erc20balances[receiver] = erc20BalanceOf(t, types.StringToAddress(receiver), rootERC20Token, externalChainTxRelayer)
		}

		// trying deposit for rollback
		require.NoError(t,
			bridge.Deposit(
				common.ERC20,
				rootERC20Token,
				bridgeCfg.ExternalERC20PredicateAddr,
				bridgeHelper.TestAccountPrivKey,
				strings.Join(receivers, ","),
				strings.Join(amounts, ","),
				"",
				bridge.JSONRPCAddr(),
				bridgeHelper.TestAccountPrivKey,
				false,
			))

		finalBlockNum := 10 * sprintSize
		// wait for a couple of sprints
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

		// the bridge transactions are processed and there should be a success state sync events
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, finalBlockNum, childEthEndpoint)
		require.NoError(t, err)

		// assert that all deposits are executed successfully
		// because of the token mapping with the first deposit
		assertBridgeEventResultSuccess(t, logs, 0)

		for _, receiver := range receivers {
			// assert that receiver's balance on RootERC20 smart contract is as expected
			balance := erc20BalanceOf(t, types.StringToAddress(receiver), rootERC20Token, externalChainTxRelayer)
			require.True(t, erc20balances[receiver].Cmp(balance) == 0)
		}

		t.Log("Deposits were successfully rollbacked")
	})
}
