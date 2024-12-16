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
	"github.com/0xPolygon/polygon-edge/state/runtime/addresslist"
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

func TestE2E_Rollback_E2I(t *testing.T) {
	const (
		transfersCount        = 1
		numBlockConfirmations = 2
		epochSize             = 40
		sprintSize            = uint64(5)
		numberOfAttempts      = 7
		stateSyncedLogsCount  = 2
		numberOfBridges       = 1
		numberOfMapTokenEvent = 1
		bridgeERC1155Amount   = 100
	)

	var (
		bridgeERC20Amount = ethgo.Ether(2)
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
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.Premine = append(tcc.Premine, receivers...)
		}))

	defer cluster.Stop()

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, chainConfigFileName))
	require.NoError(t, err)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]
	bridge := cluster.Bridges[0]

	require.NoError(t, err)

	deployerKey, err := bridgeHelper.DecodePrivateKey("")
	require.NoError(t, err)

	t.Run("Rollback_ERC20", func(t *testing.T) {
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

		finalBlockNum := 1 * sprintSize
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

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

		finalBlockNum = 10 * sprintSize
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := range receivers {
				if !isEventProcessedRollback(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, uint64(i+1)) {
					return false
				}
			}

			return true
		}))

		t.Log("Deposits were successfully rollbacked")
	})

	t.Run("Rollback_ERC721", func(t *testing.T) {
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

		receipt, err := externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
		require.NoError(t, err)

		rootERC721Addr := types.Address(receipt.ContractAddress)

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

		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := range receivers {
				if !isEventProcessedRollback(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, uint64(i+1)) {
					return false
				}
			}

			return true
		}))

		t.Log("Deposits were successfully rollbacked")
	})

	t.Run("Rollback_ERC1155", func(t *testing.T) {
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

		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := range receivers {
				if !isEventProcessedRollback(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, uint64(i+1)) {
					return false
				}
			}

			return true
		}))

		t.Log("Deposits were successfully processed")
	})
}

func TestE2E_Rollback_I2E(t *testing.T) {
	const (
		transfersCount   = uint64(4)
		amount           = 100
		epochSize        = 30
		sprintSize       = uint64(5)
		numberOfAttempts = 4
		numberOfBridges  = 1
	)

	depositorKeys := make([]string, transfersCount)
	depositors := make([]types.Address, transfersCount)
	amounts := make([]string, transfersCount)
	funds := make([]*big.Int, transfersCount)
	singleToken := ethgo.Ether(1)

	admin, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	adminAddr := admin.Address()

	for i := uint64(0); i < transfersCount; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		rawKey, err := key.MarshallPrivateKey()
		require.NoError(t, err)

		depositorKeys[i] = hex.EncodeToString(rawKey)
		depositors[i] = key.Address()
		funds[i] = singleToken
		amounts[i] = fmt.Sprintf("%d", amount)

		t.Logf("Depositor#%d=%s\n", i+1, depositors[i])
	}

	cluster := framework.NewTestCluster(t, 5,
		framework.WithNumBlockConfirmations(0),
		framework.WithTestRollback(),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBlockListAdmin(adminAddr),
		framework.WithPremine(append(depositors, adminAddr)...)) //nolint:makezero
	defer cluster.Stop()

	bridgeOne := 0

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, chainConfigFileName))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]
	childEthEndpoint := validatorSrv.JSONRPC()

	require.NoError(t, validatorSrv.ExternalChainFundFor(depositors, funds, uint64(bridgeOne)))

	cluster.WaitForReady(t)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[bridgeOne].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
	require.NoError(t, err)

	t.Run("Rollback_ERC20", func(t *testing.T) {
		rootToken := contracts.NativeERC20TokenContract

		for i, key := range depositorKeys {
			err = cluster.Bridges[bridgeOne].Deposit(
				common.ERC20,
				rootToken,
				bridgeCfg.InternalMintableERC20PredicateAddr,
				key,
				depositors[i].String(),
				amounts[i],
				"",
				validatorSrv.JSONRPCAddr(),
				"",
				true)
			require.NoError(t, err)
		}

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := uint64(1); i <= transfersCount+1; i++ {
				if !isEventProcessedRollback(t, bridgeCfg.InternalGatewayAddr, internalChainTxRelayer, i) {
					return false
				}
			}

			return true
		}))
	})

	t.Run("Rollback_ERC721", func(t *testing.T) {
		erc721DeployTxn := cluster.Deploy(t, admin, contractsapi.RootERC721.Bytecode)
		require.True(t, erc721DeployTxn.Succeed())
		rootERC721Token := types.Address(erc721DeployTxn.Receipt().ContractAddress)

		for _, depositor := range depositors {
			mintFn := &contractsapi.MintRootERC721Fn{To: depositor}
			mintInput, err := mintFn.EncodeAbi()
			require.NoError(t, err)

			mintTxn := cluster.MethodTxn(t, admin, rootERC721Token, mintInput)
			require.True(t, mintTxn.Succeed())

			setAccessListRole(t, cluster, contracts.BlockListBridgeAddr, depositor, addresslist.EnabledRole, admin)
		}

		err = cluster.Bridges[bridgeOne].Deposit(
			common.ERC721,
			rootERC721Token,
			bridgeCfg.InternalMintableERC721PredicateAddr,
			depositorKeys[0],
			depositors[0].String(),
			"",
			fmt.Sprintf("%d", 0),
			validatorSrv.JSONRPCAddr(),
			"",
			true)
		require.Error(t, err)

		for i, depositorKey := range depositorKeys {
			setAccessListRole(t, cluster, contracts.BlockListBridgeAddr, depositors[i], addresslist.NoRole, admin)

			err = cluster.Bridges[bridgeOne].Deposit(
				common.ERC721,
				rootERC721Token,
				bridgeCfg.InternalMintableERC721PredicateAddr,
				depositorKey,
				depositors[i].String(),
				"",
				fmt.Sprintf("%d", i),
				validatorSrv.JSONRPCAddr(),
				"",
				true)
			require.NoError(t, err)
		}

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := uint64(1); i <= transfersCount+1; i++ {
				if !isEventProcessedRollback(t, bridgeCfg.InternalGatewayAddr, internalChainTxRelayer, i) {
					return false
				}
			}

			return true
		}))

	})
}
