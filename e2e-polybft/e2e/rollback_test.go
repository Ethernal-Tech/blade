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

func TestE2E_TestRollback_E2I(t *testing.T) {
	const (
		transfersCount        = 1
		numBlockConfirmations = 2
		// make epoch size long enough, so that all exit events are processed within the same epoch
		epochSize             = 40
		sprintSize            = uint64(5)
		numberOfAttempts      = 7
		stateSyncedLogsCount  = 2 // map token and deposit
		numberOfBridges       = 1
		numberOfMapTokenEvent = 1
		bridgeERC1155Amount   = 100
	)

	var (
		bridgeERC20Amount   = ethgo.Ether(2)
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
		amounts[i] = fmt.Sprintf("%d", bridgeERC20Amount)

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

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
	require.NoError(t, err)

	deployerKey, err := bridgeHelper.DecodePrivateKey("")
	require.NoError(t, err)

	t.Run("Rollback_TestERC20", func(t *testing.T) {
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

		// wait for a couple of sprints
		finalBlockNum = 10 * sprintSize
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

		// the bridge transactions are processed and there should be a success state sync events
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, finalBlockNum, childEthEndpoint)
		require.NoError(t, err)

		// assert that all deposits are rollbacked (no success events)
		assertBridgeEventResultSuccess(t, logs, 0)

		childERC20Token := getChildToken(t, contractsapi.RootERC20Predicate.Abi,
			bridgeCfg.ExternalERC20PredicateAddr, rootERC20Token, externalChainTxRelayer)

		for _, receiver := range receivers {
			balanceOfFn := &contractsapi.BalanceOfRootERC20Fn{Account: types.StringToAddress(receiver)}
			balanceOfInput, err := balanceOfFn.EncodeAbi()
			require.NoError(t, err)

			balanceRaw, err := txRelayer.Call(types.ZeroAddress, childERC20Token, balanceOfInput)
			require.NoError(t, err)
			// Child token balance should be equal to 0x because rollback is executed
			require.Equal(t, balanceRaw, "0x")
		}
		t.Log("Deposits were successfully rollbacked")
	})

	t.Run("Rollback_TestERC721", func(t *testing.T) {
		tokenIDs := make([]string, transfersCount)

		for i := 0; i < transfersCount; i++ {
			key, err := crypto.GenerateECDSAKey()
			require.NoError(t, err)

			rawKey, err := key.MarshallPrivateKey()
			require.NoError(t, err)

			receiverKeys[i] = hex.EncodeToString(rawKey)
			receivers[i] = key.Address().String()
			receiversAddrs[i] = key.Address()
			tokenIDs[i] = fmt.Sprintf("%d", i)

			t.Logf("Receiver#%d=%s\n", i+1, receivers[i])
		}

		deployTx := types.NewTx(&types.LegacyTx{
			BaseTx: &types.BaseTx{
				To:    nil,
				Input: contractsapi.RootERC721.Bytecode,
			},
		})

		// deploy root ERC 721 token
		receipt, err := externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
		require.NoError(t, err)

		rootERC721Addr := types.Address(receipt.ContractAddress)

		// DEPOSIT ERC721 TOKENS
		// send a few transactions to the bridge
		require.NoError(
			t,
			bridge.Deposit(
				common.ERC721,
				rootERC721Addr,
				bridgeCfg.ExternalERC721PredicateAddr,
				bridgeHelper.TestAccountPrivKey,
				strings.Join(receivers, ","),
				"",
				strings.Join(tokenIDs, ","),
				bridge.JSONRPCAddr(),
				bridgeHelper.TestAccountPrivKey,
				false),
		)

		// wait for a few more sprints
		require.NoError(t, cluster.WaitForBlock(50, 4*time.Minute))

		validatorSrv := cluster.Servers[0]
		childEthEndpoint := validatorSrv.JSONRPC()

		// the transactions are processed and there should be a success events
		var bridgeMessageResult contractsapi.BridgeMessageResultEvent

		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, uint64(50+2*epochSize), childEthEndpoint)
		require.NoError(t, err)

		// It shouldn't transfer any token, it should rollback
		assertBridgeEventResultSuccess(t, logs, 0)

		// retrieve child token address (from both chains, and assert they are the same)
		externalChildTokenAddr := getChildToken(t, contractsapi.RootERC721Predicate.Abi, bridgeCfg.ExternalERC721PredicateAddr,
			rootERC721Addr, externalChainTxRelayer)

		t.Log("External child token", externalChildTokenAddr)

		owner := erc721OwnerOf(t, big.NewInt(0), externalChildTokenAddr, externalChainTxRelayer)
		require.NotEqual(t, deployerKey.Address(), owner)

		t.Log("Deposits were successfully rollback")
	})

	t.Run("Rollback_TestERC1155", func(t *testing.T) {
		tokenIDs := make([]string, transfersCount)

		for i := 0; i < transfersCount; i++ {
			key, err := crypto.GenerateECDSAKey()
			require.NoError(t, err)

			rawKey, err := key.MarshallPrivateKey()
			require.NoError(t, err)

			receiverKeys[i] = hex.EncodeToString(rawKey)
			receivers[i] = key.Address().String()
			receiversAddrs[i] = key.Address()
			amounts[i] = fmt.Sprintf("%d", bridgeERC1155Amount)
			tokenIDs[i] = fmt.Sprintf("%d", i+1)

			t.Logf("Receiver#%d=%s\n", i+1, receivers[i])
		}

		deployTx := types.NewTx(&types.LegacyTx{
			BaseTx: &types.BaseTx{
				To:    nil,
				Input: contractsapi.RootERC1155.Bytecode,
			},
		})

		receipt, err := externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
		require.NoError(t, err)

		rootERC1155Addr := types.Address(receipt.ContractAddress)
		require.NoError(
			t,
			bridge.Deposit(
				common.ERC1155,
				rootERC1155Addr,
				bridgeCfg.ExternalERC1155PredicateAddr,
				bridgeHelper.TestAccountPrivKey,
				strings.Join(receivers, ","),
				strings.Join(amounts, ","),
				strings.Join(tokenIDs, ","),
				bridge.JSONRPCAddr(),
				bridgeHelper.TestAccountPrivKey,
				false),
		)

		// wait for a few more sprints
		require.NoError(t, cluster.WaitForBlock(50, 4*time.Minute))

		validatorSrv := cluster.Servers[0]
		childEthEndpoint := validatorSrv.JSONRPC()

		// the transactions are processed and there should be a success events
		var bridgeMessageResult contractsapi.BridgeMessageResultEvent

		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, uint64(50+2*epochSize), childEthEndpoint)
		require.NoError(t, err)

		assertBridgeEventResultSuccess(t, logs, 0)

		txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
		require.NoError(t, err)

		// retrieve child token address
		l1ChildTokenAddr := getChildToken(t, contractsapi.RootERC1155Predicate.Abi, bridgeCfg.ExternalERC1155PredicateAddr,
			rootERC1155Addr, externalChainTxRelayer)
		l2ChildTokenAddr := getChildToken(t, contractsapi.ChildERC1155Predicate.Abi, bridgeCfg.InternalERC1155PredicateAddr,
			rootERC1155Addr, txRelayer)

		t.Log("L1 child token", l1ChildTokenAddr)
		t.Log("L2 child token", l2ChildTokenAddr)
		// require.Equal(t, l1ChildTokenAddr, l2ChildTokenAddr)

		// check receivers balances got increased by deposited amount
		for i := range receivers {
			balanceOfFn := &contractsapi.BalanceOfChildERC1155Fn{
				Account: deployerKey.Address(),
				ID:      big.NewInt(int64(i + 1)),
			}

			balanceInput, err := balanceOfFn.EncodeAbi()
			require.NoError(t, err)

			balanceRaw, err := externalChainTxRelayer.Call(types.ZeroAddress, l1ChildTokenAddr, balanceInput)
			require.NoError(t, err)

			require.Equal(t, balanceRaw, "0x")
		}

		t.Log("Deposits were successfully processed")
	})
}
