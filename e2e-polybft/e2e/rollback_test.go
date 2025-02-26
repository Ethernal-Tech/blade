package e2e

import (
	"fmt"
	"math/big"
	"path"
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
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/stretchr/testify/require"
)

// TestE2E_Rollback_E2I tests the end-to-end rollback functionality for ERC20, ERC721, and ERC1155 tokens
// from an external chain to an internal chain. It sets up a test cluster, deploys the necessary contracts,
// performs deposits, and verifies that the deposits are successfully rollbacked.
//
// TestE2E_Rollback_I2E tests the end-to-end rollback functionality for ERC20 and ERC721 tokens
// from an internal chain to an external chain. It sets up a test cluster, deploys the necessary contracts,
// performs deposits, and verifies that the deposits are successfully rollbacked.
//
// The tests include the following scenarios:
// - Rollback_ERC20: Tests the rollback of ERC20 token deposits.
// - Rollback_ERC721: Tests the rollback of ERC721 token deposits.
// - Rollback_ERC1155: Tests the rollback of ERC1155 token deposits.
//
// The tests use the following constants:
// - transfersCount: Number of transfers to perform.
// - numBlockConfirmations: Number of block confirmations required.
// - epochSize: Size of an epoch.
// - sprintSize: Size of a sprint.
// - numberOfAttempts: Number of attempts for certain operations.
// - stateSyncedLogsCount: Number of state synced logs.
// - numberOfBridges: Number of bridges to use.
// - numberOfMapTokenEvent: Number of map token events.
// - bridgeERC1155Amount: Amount of ERC1155 tokens to transfer.
// - amount: Amount of tokens to transfer.
//
// The tests use the following variables:
// - bridgeERC20Amount: Amount of ERC20 tokens to transfer.
// - receiversAddrs: List of receiver addresses.
// - receivers: List of receiver addresses as strings.
// - amounts: List of amounts to transfer as strings.
// - receiverKeys: List of receiver private keys as strings.
// - depositorKeys: List of depositor private keys as strings.
// - depositors: List of depositor addresses.
// - funds: List of funds to transfer.
// - singleToken: Amount of a single token.
//
// The tests perform the following steps:
// 1. Generate receiver and depositor keys and addresses.
// 2. Set up a test cluster with the specified configuration.
// 3. Deploy the necessary contracts on the external and internal chains.
// 4. Perform deposits of ERC20, ERC721, and ERC1155 tokens.
// 5. Wait for the deposits to be processed and verify the rollback events.

func TestE2E_Rollback_E2I(t *testing.T) {
	const (
		transfersCount        = 5
		numOfRollback         = int((transfersCount + 1) / 2)
		numBlockConfirmations = 2
		epochSize             = 10
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

	// Generating receiver keys and addresses
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

	// relayer key
	relayerKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	gatewayAddr := types.StringToAddress("0x2222")
	// Setting up the test cluster with rollback gateway contract
	cluster := framework.NewTestCluster(t, 5,
		framework.WithTestRewardToken(),
		framework.WithRollback(framework.E2IRollback),
		framework.WithNumBlockConfirmations(numBlockConfirmations),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBatchThreshold(100),
		framework.WithPredeploy(fmt.Sprintf("%s:TestRollbackGateway", gatewayAddr)),
		framework.WithRelayerPrivateKey(relayerKey),
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

	internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Servers[0].JSONRPCAddr()))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]

	externalRPC, err := jsonrpc.NewEthClient(cluster.Bridges[0].JSONRPCAddr())
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]
	bridge := cluster.Bridges[0]

	require.NoError(t, err)

	// Default deployer key
	deployerKey, err := bridgeHelper.DecodePrivateKey("")
	require.NoError(t, err)

	evNum := uint64(1) // to track with which event numbers starts & ends
	startEventNum := func() uint64 { return (evNum-1)*transfersCount + evNum }
	endEventNum := func() uint64 { return (evNum)*transfersCount + evNum }

	validateBridgeRollback := func(externalBlockStart uint64, internalBlockStart uint64) {
		latest, err := validatorSrv.JSONRPC().BlockNumber()
		require.NoError(t, err)

		var bridgeMessageResult contractsapi.BridgeMessageResultEvent
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), internalBlockStart, latest, validatorSrv.JSONRPC())
		require.NoError(t, err)

		assertBridgeEventResultNotSuccessful(t, logs, numOfRollback)

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := startEventNum(); i <= endEventNum(); i++ {
				if i%2 == 0 && !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, true) {
					return false
				}
			}

			return true
		}))

		latest = waitForBlocksOnExternal(t, 20, externalRPC, 2*time.Minute)

		logs, err = getFilteredLogs(bridgeMessageResult.Sig(), externalBlockStart, latest, externalRPC)
		require.NoError(t, err)

		assertBridgeEventResultSuccessful(t, logs, numOfRollback)
	}

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

		// Wait for the rollback to be processed
		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := startEventNum(); i <= endEventNum(); i++ {
				if !isEventProcessed(t, bridgeCfg.InternalGatewayAddr, internalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		validateBridgeRollback(0, 0)
	})

	evNum++

	t.Run("Rollback_ERC721", func(t *testing.T) {
		tokenIDs := make([]string, transfersCount)

		for i := 0; i < transfersCount; i++ {
			tokenIDs[i] = fmt.Sprintf("%d", i)
		}

		deployTx := types.NewTx(&types.LegacyTx{
			BaseTx: &types.BaseTx{
				To:    nil,
				Input: contractsapi.RootERC721.Bytecode,
			},
		})

		waitForBlocksOnExternal(t, 10, externalRPC, 2*time.Minute)

		startBlockInternal, err := validatorSrv.JSONRPC().BlockNumber()
		require.NoError(t, err)

		startBlockExternal, err := externalRPC.BlockNumber()
		require.NoError(t, err)

		receipt, err := externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
		require.NoError(t, err)

		rootERC721Addr := types.Address(receipt.ContractAddress)

		for i := range transfersCount {
			require.NoError(
				t,
				bridge.Deposit(
					common.ERC721,
					rootERC721Addr,
					bridgeCfg.ExternalERC721PredicateAddr,
					bridgeHelper.TestAccountPrivKey,
					receivers[i],
					"",
					tokenIDs[i],
					bridge.JSONRPCAddr(),
					bridgeHelper.TestAccountPrivKey,
					false),
			)
		}

		// Wait for rollback to be processed
		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := startEventNum(); i <= endEventNum(); i++ {
				if !isEventProcessed(t, bridgeCfg.InternalGatewayAddr, internalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		validateBridgeRollback(startBlockExternal, startBlockInternal)
	})

	evNum++

	t.Run("Rollback_ERC1155", func(t *testing.T) {
		tokenIDs := make([]string, transfersCount)
		for i := 0; i < transfersCount; i++ {
			tokenIDs[i] = fmt.Sprintf("%d", i+1)
		}

		deployTx := types.NewTx(&types.LegacyTx{
			BaseTx: &types.BaseTx{
				To:    nil,
				Input: contractsapi.RootERC1155.Bytecode,
			},
		})

		waitForBlocksOnExternal(t, 10, externalRPC, 2*time.Minute)

		startBlockExternal, err := externalRPC.BlockNumber()
		require.NoError(t, err)

		startBlockInternal, err := validatorSrv.JSONRPC().BlockNumber()
		require.NoError(t, err)

		receipt, err := externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
		require.NoError(t, err)

		rootERC1155Addr := types.Address(receipt.ContractAddress)
		for i := range receivers {
			require.NoError(
				t,
				bridge.Deposit(
					common.ERC1155,
					rootERC1155Addr,
					bridgeCfg.ExternalERC1155PredicateAddr,
					bridgeHelper.TestAccountPrivKey,
					receivers[i],
					amounts[i],
					tokenIDs[i],
					bridge.JSONRPCAddr(),
					bridgeHelper.TestAccountPrivKey,
					false),
			)
		}

		// Wait for rollback to be processed
		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := startEventNum(); i <= endEventNum(); i++ {
				if !isEventProcessed(t, bridgeCfg.InternalGatewayAddr, internalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		validateBridgeRollback(startBlockExternal, startBlockInternal)
	})
}

func TestE2E_Rollback_I2E(t *testing.T) {
	const (
		transfersCount   = uint64(5)
		numOfRollback    = int((transfersCount + 1) / 2)
		amount           = 100
		epochSize        = 10
		sprintSize       = uint64(5)
		numberOfAttempts = 4
		numberOfBridges  = 1
	)

	var (
		depositorKeys = make([]string, transfersCount)
		depositors    = make([]types.Address, transfersCount)
		amounts       = make([]string, transfersCount)
		funds         = make([]*big.Int, transfersCount)
		singleToken   = ethgo.Ether(1)
	)

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

	// relayer key
	relayerKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithNumBlockConfirmations(0),
		framework.WithBridgeBatchThreshold(100),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBlockListAdmin(adminAddr),
		framework.WithRollback(framework.I2ERollback),
		framework.WithRelayerPrivateKey(relayerKey),
		framework.WithBlockGasLimit(100000000),
		framework.WithPremine(append(depositors, adminAddr)...))
	defer cluster.Stop()

	bridgeOne := 0
	bridge := cluster.Bridges[bridgeOne]

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, chainConfigFileName))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]
	childEthEndpoint := validatorSrv.JSONRPC()

	require.NoError(t, validatorSrv.ExternalChainFundFor(depositors, funds, uint64(bridgeOne)))

	cluster.WaitForReady(t)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(bridge.JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
	require.NoError(t, err)

	externalRPC, err := jsonrpc.NewEthClient(bridge.JSONRPCAddr())
	require.NoError(t, err)

	evNum := uint64(1) // to track with which event numbers starts & ends
	startEventNum := func() uint64 { return (evNum-1)*transfersCount + evNum }
	endEventNum := func() uint64 { return evNum*transfersCount + evNum }

	// validate bridge rollback with events
	validateBridgeRollback := func(externalBlockStart, internalBlockStart uint64) {
		latest := waitForBlocksOnExternal(t, 20, externalRPC, 2*time.Minute)

		var bridgeMessageResult contractsapi.BridgeMessageResultEvent
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), externalBlockStart, latest, externalRPC)
		require.NoError(t, err)

		assertBridgeEventResultNotSuccessful(t, logs, numOfRollback)

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := startEventNum(); i <= endEventNum(); i++ {
				if i%2 == 0 && !isEventProcessed(t, bridgeCfg.InternalGatewayAddr, internalChainTxRelayer, i, true) {
					return false
				}
			}

			return true
		}))

		latest, err = validatorSrv.JSONRPC().BlockNumber()
		require.NoError(t, err)

		logs, err = getFilteredLogs(bridgeMessageResult.Sig(), internalBlockStart, latest, validatorSrv.JSONRPC())
		require.NoError(t, err)

		assertBridgeEventResultSuccessful(t, logs, numOfRollback)
	}

	t.Run("Rollback_ERC20", func(t *testing.T) {
		rootToken := contracts.NativeERC20TokenContract

		for i, key := range depositorKeys {
			err = bridge.Deposit(
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
			for i := startEventNum(); i <= endEventNum(); i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		validateBridgeRollback(0, 0)
	})

	evNum++

	t.Run("Rollback_ERC721", func(t *testing.T) {
		erc721DeployTxn := cluster.Deploy(t, admin, contractsapi.RootERC721.Bytecode)
		require.True(t, erc721DeployTxn.Succeed())
		rootERC721Token := types.Address(erc721DeployTxn.Receipt().ContractAddress)

		waitForBlocksOnExternal(t, 10, externalRPC, time.Minute)

		// Just processing logs after Rollback ERC20
		startBlockOnExternal, err := externalRPC.BlockNumber()
		require.NoError(t, err)

		startBlockOnInternal, err := validatorSrv.JSONRPC().BlockNumber()
		require.NoError(t, err)

		for _, depositor := range depositors {
			mintFn := &contractsapi.MintRootERC721Fn{To: depositor}
			mintInput, err := mintFn.EncodeAbi()
			require.NoError(t, err)

			mintTxn := cluster.MethodTxn(t, admin, rootERC721Token, mintInput)
			require.True(t, mintTxn.Succeed())
		}

		for i, depositorKey := range depositorKeys {
			err = bridge.Deposit(
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
			for i := startEventNum(); i <= endEventNum(); i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		validateBridgeRollback(startBlockOnExternal, startBlockOnInternal)
	})
}

func TestE2E_Retry_I2E(t *testing.T) {
	const (
		transfersCount   = uint64(5)
		amount           = 100
		epochSize        = 10
		threshold        = 20
		sprintSize       = uint64(5)
		numberOfAttempts = 4
		numberOfBridges  = 1
	)

	const (
		startEventERC721 = transfersCount + 2
		endEventERC721   = startEventERC721 + transfersCount
	)

	var (
		depositorKeys = make([]string, transfersCount)
		depositors    = make([]types.Address, transfersCount)
		amounts       = make([]string, transfersCount)
	)

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
		amounts[i] = fmt.Sprintf("%d", amount)

		t.Logf("Depositor#%d=%s\n", i+1, depositors[i])
	}

	// relayer key
	relayerKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithNumBlockConfirmations(0),
		framework.WithBridgeBatchThreshold(threshold),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithRelayerPrivateKey(relayerKey),
		framework.WithPremine(append(depositors, adminAddr)...))
	defer cluster.Stop()

	cluster.WaitForReady(t)

	bridge := cluster.Bridges[0]

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, chainConfigFileName))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]
	internalRPC := validatorSrv.JSONRPC()

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(bridge.JSONRPCAddr()))
	require.NoError(t, err)

	internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(internalRPC))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	t.Run("Retry_ERC20", func(t *testing.T) {
		// Stop relayer to simulate retry
		cluster.BridgeRelayers[0].Stop()

		rootToken := contracts.NativeERC20TokenContract

		for i, key := range depositorKeys {
			require.NoError(t,
				bridge.Deposit(
					common.ERC20,
					rootToken,
					bridgeCfg.InternalMintableERC20PredicateAddr,
					key,
					depositors[i].String(),
					amounts[i],
					"",
					validatorSrv.JSONRPCAddr(),
					"",
					true))
		}

		currentBlock, err := internalRPC.BlockNumber()
		require.NoError(t, err)

		require.NoError(t, cluster.WaitForBlock(currentBlock+2*threshold, 2*time.Minute))

		// Start relayer again to retry
		cluster.BridgeRelayers[0].Start()

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := uint64(1); i <= transfersCount+1; i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		childToken := getChildToken(t,
			contractsapi.RootERC20Predicate.Abi, bridgeCfg.InternalMintableERC20PredicateAddr,
			contracts.NativeERC20TokenContract, internalChainTxRelayer)

		for _, depositor := range depositors {
			balance := erc20BalanceOf(t, depositor, childToken, externalChainTxRelayer)
			require.Equal(t, big.NewInt(amount), balance)
		}
	})

	t.Run("Retry_ERC721", func(t *testing.T) {
		cluster.BridgeRelayers[0].Stop()

		erc721DeployTxn := cluster.Deploy(t, admin, contractsapi.RootERC721.Bytecode)
		require.True(t, erc721DeployTxn.Succeed())
		rootERC721Token := types.Address(erc721DeployTxn.Receipt().ContractAddress)

		for _, depositor := range depositors {
			mintFn := &contractsapi.MintRootERC721Fn{To: depositor}
			mintInput, err := mintFn.EncodeAbi()
			require.NoError(t, err)

			mintTxn := cluster.MethodTxn(t, admin, rootERC721Token, mintInput)
			require.True(t, mintTxn.Succeed())
		}

		for i, depositorKey := range depositorKeys {
			require.NoError(t,
				bridge.Deposit(
					common.ERC721,
					rootERC721Token,
					bridgeCfg.InternalMintableERC721PredicateAddr,
					depositorKey,
					depositors[i].String(),
					"",
					fmt.Sprintf("%d", i),
					validatorSrv.JSONRPCAddr(),
					"",
					true))
		}

		currentBlock, err := internalRPC.BlockNumber()
		require.NoError(t, err)

		require.NoError(t, cluster.WaitForBlock(currentBlock+2*threshold, 2*time.Minute))

		cluster.BridgeRelayers[0].Start()

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := startEventERC721; i <= endEventERC721; i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		childERC721Token := getChildToken(t, contractsapi.RootERC721Predicate.Abi,
			bridgeCfg.InternalMintableERC721PredicateAddr, rootERC721Token, internalChainTxRelayer)

		for i, depositor := range depositors {
			owner := erc721OwnerOf(t, big.NewInt(int64(i)), childERC721Token, externalChainTxRelayer)
			require.Equal(t, depositor, owner)
		}
	})
}
