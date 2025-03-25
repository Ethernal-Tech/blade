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
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	helperCommon "github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

// TestE2E_Bridge_ValidatorSync is an end-to-end test that verifies validator synchronization
// in a multi-validator blockchain network with a bridge.
//
// This test ensures that validator nodes remain in sync after transactions are processed through the bridge.
// The test follows these steps:
//  1. Initializes a test cluster with multiple validators and a bridge.
//  2. Stops one validator to create an out-of-sync scenario.
//  3. Performs multiple deposit transactions through the bridge.
//  4. Restarts the stopped validator and waits for it to synchronize with the rest of the network.
//  5. Compares restarted validator with another active validator to ensure they are in sync and have identical state data.
//
// This test validates that validators can correctly resynchronize after being offline
func TestE2E_Bridge_ValidatorSync(t *testing.T) {
	const (
		transfersCount        = 10
		numBlockConfirmations = 2
		// make epoch size long enough, so that all exit events are processed within the same epoch
		epochSize             = 200
		sprintSize            = 100
		numberOfAttempts      = 7
		stateSyncedLogsCount  = 2 // map token and deposit
		numberOfBridges       = 1
		numberOfMapTokenEvent = 1
	)

	var (
		bridgeAmount = ethgo.Ether(2)
		// bridgeMessageResult contractsapi.BridgeMessageResultEvent
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

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithTestRewardToken(),
		framework.WithNumBlockConfirmations(numBlockConfirmations),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBatchThreshold(100),
		framework.WithSprintSize(sprintSize),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				// premine receivers, so that they are able to do withdrawals
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))

			tcc.Premine = append(tcc.Premine, receivers...)
			tcc.Premine = append(tcc.Premine, relayerPrivateKey.String())
		}))

	defer cluster.Stop()

	bridgeOne := 0

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

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

	validatorSrv1 := cluster.Servers[0]
	validatorSrv2 := cluster.Servers[1]

	options := &bolt.Options{ReadOnly: true}

	wg := sync.WaitGroup{}

	validatorSrv1.Stop()

	wg.Add(1)

	go func() {
		defer wg.Done()
		// DEPOSIT ERC20 TOKENS
		// send a few transactions to the bridge
		require.NoError(t,
			cluster.Bridges[bridgeOne].Deposit(
				common.ERC20,
				rootERC20Token,
				bridgeCfg.ExternalERC20PredicateAddr,
				bridgeHelper.TestAccountPrivKey,
				strings.Join(receivers, ","),
				strings.Join(amounts, ","),
				"",
				cluster.Bridges[bridgeOne].JSONRPCAddr(),
				bridgeHelper.TestAccountPrivKey,
				false,
			))
	}()

	// rootToken represents deposit token (basically native mintable token from the Supernets)
	rootToken := contracts.NativeERC20TokenContract

	for i, key := range receiverKeys {
		wg.Add(1)

		go func() {
			defer wg.Done()
			// DEPOSIT ERC20 TOKENS
			// send a few transactions to the bridge
			// make sure deposit is successfully executed
			err = cluster.Bridges[bridgeOne].Deposit(
				common.ERC20,
				rootToken,
				bridgeCfg.InternalMintableERC20PredicateAddr,
				key,
				receiversAddrs[i].String(),
				amounts[i],
				"",
				cluster.Servers[1].JSONRPCAddr(),
				"",
				true)
			require.NoError(t, err)
		}()
	}

	wg.Wait()

	t.Log("Successfully deposited all transactions")

	validator2EthEndpoint := validatorSrv2.JSONRPC()

	currentBlock, err := validator2EthEndpoint.BlockNumber()
	require.NoError(t, err)

	validatorSrv1.Start()

	require.NoError(t, cluster.WaitForBlock(currentBlock+5, time.Minute))

	validatorSrv1.Stop()
	validatorSrv2.Stop()

	db1, err := bolt.Open(filepath.Join(validatorSrv1.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	db2, err := bolt.Open(filepath.Join(validatorSrv2.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(100), helperCommon.EncodeUint64ToBytes(chainID.Uint64()), false, false)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(chainID.Uint64()), helperCommon.EncodeUint64ToBytes(100), false, false)
}

// TestE2E_Bridge_ValidatorSyncTestExecuted is an end-to-end test that verifies validator synchronization
// in a multi-validator blockchain network with a bridge.
//
// This test ensures that validator nodes remain in sync after transactions are processed through the bridge.
// The test follows these steps:
//  1. Initializes a test cluster with multiple validators and a bridge.
//  2. Stops one validator to create an out-of-sync scenario.
//  3. Performs multiple deposit transactions through the bridge and wait for transactions to be processed.
//  4. Restarts the stopped validator and waits for it to synchronize with the rest of the network.
//  5. Compares restarted validator with another active validator to ensure they are in sync and have identical state data.
//
// This test validates that validators can correctly resynchronize after being offline

func TestE2E_Bridge_ValidatorSyncExecuted(t *testing.T) {
	const (
		transfersCount        = 5
		numBlockConfirmations = 2
		sprintSize            = uint64(5)
		numberOfBridges       = 1
		numberOfMapTokenEvent = 1
	)

	var (
		bridgeAmount        = ethgo.Ether(2)
		bridgeMessageResult contractsapi.BridgeMessageResultEvent
		options             = &bolt.Options{ReadOnly: true}
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

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithTestRewardToken(),
		framework.WithNumBlockConfirmations(numBlockConfirmations),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBatchThreshold(100),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				// premine receivers, so that they are able to do withdrawals
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))

			tcc.Premine = append(tcc.Premine, receivers...)
			tcc.Premine = append(tcc.Premine, relayerPrivateKey.String())
		}))

	defer cluster.Stop()

	bridgeOne := 0

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	validatorSrv1 := cluster.Servers[0]
	validatorSrvStopped := cluster.Servers[1]

	childEthEndpoint := validatorSrv1.JSONRPC()

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
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
	t.Log("External chain token address:", rootERC20Token)

	// wait for a couple of sprints
	finalBlockNum := 1 * sprintSize
	require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

	validatorSrvStopped.Stop()

	wg := sync.WaitGroup{}

	// DEPOSIT FROM EXTERNAL ERC20 TOKENS
	// send a few transactions to the bridge
	wg.Add(1)

	go func() {
		defer wg.Done()
		require.NoError(t,
			cluster.Bridges[bridgeOne].Deposit(
				common.ERC20,
				rootERC20Token,
				bridgeCfg.ExternalERC20PredicateAddr,
				bridgeHelper.TestAccountPrivKey,
				strings.Join(receivers, ","),
				strings.Join(amounts, ","),
				"",
				cluster.Bridges[bridgeOne].JSONRPCAddr(),
				bridgeHelper.TestAccountPrivKey,
				false,
			))
	}()

	// rootToken represents deposit token (basically native mintable token from the Supernets)
	rootToken := contracts.NativeERC20TokenContract

	// DEPOSIT FROM EXTERNAL ERC20 TOKENS
	// send a few transactions to the bridge

	wg.Add(1)

	go func() {
		defer wg.Done()

		for i, key := range receiverKeys {
			// make sure deposit is successfully executed
			err = cluster.Bridges[bridgeOne].Deposit(
				common.ERC20,
				rootToken,
				bridgeCfg.InternalMintableERC20PredicateAddr,
				key,
				receivers[i],
				amounts[i],
				"",
				validatorSrv1.JSONRPCAddr(),
				"",
				true)
			require.NoError(t, err)
		}
	}()

	wg.Wait()

	// first exit event is mapping child token on a rootchain
	require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
		for i := uint64(1); i <= transfersCount+1; i++ {
			if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
				return false
			}
		}

		return true
	}))

	finalBlockNum = 5 * sprintSize
	// wait for a couple of sprints
	require.NoError(t, cluster.WaitForBlock(finalBlockNum, 4*time.Minute))

	// the bridge transactions are processed and there should be a success state sync events
	logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, finalBlockNum, childEthEndpoint)
	require.NoError(t, err)

	// assert that all deposits are executed successfully
	// because of the token mapping with the first deposit
	assertBridgeEventResultSuccess(t, logs, transfersCount+1)

	// get child token address
	childERC20Token := getChildToken(t, contractsapi.RootERC20Predicate.Abi,
		bridgeCfg.ExternalERC20PredicateAddr, rootERC20Token, externalChainTxRelayer)

	// check receivers balances got increased by deposited amount
	for _, receiver := range receivers {
		balance := erc20BalanceOf(t, types.StringToAddress(receiver), childERC20Token, txRelayer)
		require.Equal(t, bridgeAmount, balance)
	}

	t.Log("Deposits were successfully processed")

	currectBlock, err := childEthEndpoint.BlockNumber()
	require.NoError(t, err)

	validatorSrvStopped.Start()

	require.NoError(t, cluster.WaitForBlock(currectBlock+5, 2*time.Minute))

	validatorSrvStopped.Stop()
	validatorSrv1.Stop()

	db1, err := bolt.Open(filepath.Join(validatorSrv1.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	db2, err := bolt.Open(filepath.Join(validatorSrvStopped.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(100), helperCommon.EncodeUint64ToBytes(chainID.Uint64()), false, true)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(chainID.Uint64()), helperCommon.EncodeUint64ToBytes(100), false, true)
}

// TestE2E_Bridge_ValidatorSyncRollback_E2I is a test that verifies validator synchronization
// in an E2I (External-to-Internal) rollback scenario.
//
// This test simulates a multi-validator blockchain network with a bridge and performs the following steps:
// 1. Initializes a test cluster with configurable epoch and sprint sizes, enabling bridge rollbacks.
// 2. Stops a validator to create an out-of-sync scenario, allowing transactions to proceed without it.
// 3. Simulates multiple deposit transactions from different accounts through the bridge.
// 4. Starts the validator.
// 5. Waits for the validator to synchronize.
// 6. Ensures the validator syncs back correctly after restarting.

func TestE2E_Bridge_ValidatorSyncRollback_E2I(t *testing.T) {
	const (
		transfersCount        = 5
		numOfRollback         = int((transfersCount + 1) / 2)
		numBlockConfirmations = 2
		epochSize             = 40
		sprintSize            = 20
		numberOfAttempts      = 7
		stateSyncedLogsCount  = 2
		numberOfBridges       = 1
		numberOfMapTokenEvent = 1
		bridgeERC1155Amount   = 100
	)

	var (
		bridgeERC20Amount = ethgo.Ether(2)
		options           = &bolt.Options{ReadOnly: true}
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
		framework.WithSprintSize(sprintSize),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.Premine = append(tcc.Premine, receivers...)
		}))

	defer cluster.Stop()

	cluster.WaitForReady(t)

	validatorSrv1 := cluster.Servers[0]
	validatorSrv2 := cluster.Servers[1]

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]
	bridge := cluster.Bridges[0]

	require.NoError(t, err)

	// Default deployer key
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

	validatorSrv1.Stop()

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

	require.NoError(t, cluster.WaitForBlock(sprintSize+10, 4*time.Minute))

	validatorSrv1.Start()

	require.NoError(t, cluster.WaitForBlock(sprintSize+15, 2*time.Minute))

	validatorSrv1.Stop()
	validatorSrv2.Stop()

	db1, err := bolt.Open(filepath.Join(validatorSrv1.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	db2, err := bolt.Open(filepath.Join(validatorSrv2.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(chainID.Uint64()), helperCommon.EncodeUint64ToBytes(100), true, false)
}

// TestE2E_Bridge_ValidatorSyncRollbackI2E is a test that verifies validator synchronization
// in an I2E (Internal-to-External) rollback scenario.
//
// This test simulates a multi-validator blockchain network with a bridge and performs the following steps:
// 1. Initializes a test cluster with configurable epoch and sprint sizes, enabling bridge rollbacks.
// 2. Stops a validator to create an out-of-sync scenario, allowing transactions to proceed without it.
// 3. Simulates multiple deposit transactions from different accounts through the bridge.
// 4. Starts the validator.
// 5. Waits for the validator to synchronize.
// 6. Ensures the validator syncs back correctly after restarting.

func TestE2E_Bridge_ValidatorSyncRollback_I2E(t *testing.T) {
	const (
		transfersCount   = uint64(5)
		numOfRollback    = int((transfersCount + 1) / 2)
		amount           = 100
		epochSize        = 60
		sprintSize       = 30
		numberOfAttempts = 4
		numberOfBridges  = 1
	)

	var (
		depositorKeys = make([]string, transfersCount)
		depositors    = make([]types.Address, transfersCount)
		amounts       = make([]string, transfersCount)
		funds         = make([]*big.Int, transfersCount)
		singleToken   = ethgo.Ether(1)
		options       = &bolt.Options{ReadOnly: true}
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
		framework.WithSprintSize(sprintSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBlockListAdmin(adminAddr),
		framework.WithRollback(framework.I2ERollback),
		framework.WithRelayerPrivateKey(relayerKey),
		framework.WithBlockGasLimit(100000000),
		framework.WithPremine(append(depositors, adminAddr)...))

	defer cluster.Stop()

	cluster.WaitForReady(t)

	bridgeOne := 0
	bridge := cluster.Bridges[bridgeOne]

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	validatorSrv1 := cluster.Servers[0]
	validatorSrv2 := cluster.Servers[1]

	require.NoError(t, validatorSrv1.ExternalChainFundFor(depositors, funds, uint64(bridgeOne)))

	cluster.WaitForReady(t)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(bridge.JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	rootToken := contracts.NativeERC20TokenContract

	validatorSrv1.Stop()

	wg := sync.WaitGroup{}

	for i, key := range depositorKeys {
		wg.Add(1)

		go func() {
			defer wg.Done()

			require.NoError(t, bridge.Deposit(
				common.ERC20,
				rootToken,
				bridgeCfg.InternalMintableERC20PredicateAddr,
				key,
				depositors[i].String(),
				amounts[i],
				"",
				validatorSrv2.JSONRPCAddr(),
				"",
				true))
		}()
	}

	wg.Wait()

	require.NoError(t, cluster.WaitForBlock(sprintSize+10, 3*time.Minute))

	validatorSrv1.Start()

	require.NoError(t, cluster.WaitForBlock(sprintSize+15, 2*time.Minute))

	validatorSrv1.Stop()
	validatorSrv2.Stop()

	db1, err := bolt.Open(filepath.Join(validatorSrv1.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	db2, err := bolt.Open(filepath.Join(validatorSrv2.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(100), helperCommon.EncodeUint64ToBytes(chainID.Uint64()), true, false)
}

// TestE2E_Bridge_ValidatorSyncRollbackE2I is a test that verifies validator synchronization
// in an E2I (External-to-Internal) rollback scenario.
//
// This test simulates a multi-validator blockchain network with a bridge and performs the following steps:
// 1. Initializes a test cluster with configurable epoch and sprint sizes, enabling bridge rollbacks.
// 2. Stops a validator to create an out-of-sync scenario, allowing transactions to proceed without it.
// 3. Simulates multiple deposit transactions from different accounts through the bridge and wait for transactions to be processed.
// 4. Starts the validator.
// 5. Waits for the validator to synchronize.
// 6. Ensures the validator syncs back correctly after restarting.

func TestE2E_Bridge_ValidatorSyncRollbackExecuted_E2I(t *testing.T) {
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
		options           = &bolt.Options{ReadOnly: true}
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

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Servers[0].JSONRPCAddr()))
	require.NoError(t, err)

	validatorSrv1 := cluster.Servers[0]
	validatorSrvStopped := cluster.Servers[1]

	externalRPC, err := jsonrpc.NewEthClient(cluster.Bridges[0].JSONRPCAddr())
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]
	bridge := cluster.Bridges[0]

	require.NoError(t, err)

	evNum := uint64(1)
	startEventNum := func() uint64 { return (evNum-1)*transfersCount + evNum }
	endEventNum := func() uint64 { return (evNum)*transfersCount + evNum }

	// Default deployer key
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

	validatorSrvStopped.Stop()

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

	finalBlockNum = 5 * sprintSize
	require.NoError(t, cluster.WaitForBlock(finalBlockNum, 4*time.Minute))

	// Wait for the rollback to be processed
	require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
		for i := startEventNum(); i <= endEventNum(); i++ {
			if !isEventProcessed(t, bridgeCfg.InternalGatewayAddr, internalChainTxRelayer, i, false) {
				return false
			}
		}

		return true
	}))

	validateBridgeRollbackExternal(
		t,
		cluster,
		0,
		0,
		startEventNum(),
		endEventNum(),
		numOfRollback,
		externalChainTxRelayer,
		bridgeCfg.ExternalGatewayAddr,
		externalRPC)

	blockNumber, err := validatorSrv1.JSONRPC().BlockNumber()
	require.NoError(t, err)

	validatorSrvStopped.Start()

	require.NoError(t, cluster.WaitForBlock(blockNumber+5, 2*time.Minute))

	validatorSrv1.Stop()
	validatorSrvStopped.Stop()

	db1, err := bolt.Open(filepath.Join(validatorSrv1.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	db2, err := bolt.Open(filepath.Join(validatorSrvStopped.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(100), helperCommon.EncodeUint64ToBytes(chainID.Uint64()), false, true)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(chainID.Uint64()), helperCommon.EncodeUint64ToBytes(100), false, true)
}

// TestE2E_Bridge_ValidatorSyncRollbackExecuted_I2E is a test that verifies validator synchronization
// in an I2E (Internal-to-External) rollback scenario.
//
// This test simulates a multi-validator blockchain network with a bridge and performs the following steps:
// 1. Initializes a test cluster with configurable epoch and sprint sizes, enabling bridge rollbacks.
// 2. Stops a validator to create an out-of-sync scenario, allowing transactions to proceed without it.
// 3. Simulates multiple deposit transactions from different accounts through the bridge.
// 4. Starts the validator.
// 5. Waits for the validator to synchronize.
// 6. Ensures the validator syncs back correctly after restarting

func TestE2E_Bridge_ValidatorSyncRollbackExecuted_I2E(t *testing.T) {
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
		options       = &bolt.Options{ReadOnly: true}
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

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	validatorSrv1 := cluster.Servers[0]
	childEthEndpoint := validatorSrv1.JSONRPC()
	validatorSrvStopped := cluster.Servers[1]

	require.NoError(t, validatorSrv1.ExternalChainFundFor(depositors, funds, uint64(bridgeOne)))

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

	rootToken := contracts.NativeERC20TokenContract

	validatorSrvStopped.Stop()

	for i, key := range depositorKeys {
		err = bridge.Deposit(
			common.ERC20,
			rootToken,
			bridgeCfg.InternalMintableERC20PredicateAddr,
			key,
			depositors[i].String(),
			amounts[i],
			"",
			validatorSrv1.JSONRPCAddr(),
			"",
			true)
		require.NoError(t, err)
	}

	t.Log("After deposit")

	require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
		for i := startEventNum(); i <= endEventNum(); i++ {
			if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
				return false
			}
		}

		return true
	}))

	validateBridgeRollbackInternal(t,
		cluster,
		0, // external start block
		0, // internal start block
		startEventNum(),
		endEventNum(),
		numOfRollback,
		bridgeCfg.InternalGatewayAddr,
		internalChainTxRelayer,
		externalRPC)

	t.Log("after validate")

	blockNumber, err := childEthEndpoint.BlockNumber()
	require.NoError(t, err)

	validatorSrvStopped.Start()

	require.NoError(t, cluster.WaitForBlock(blockNumber+5, 2*time.Minute))

	validatorSrvStopped.Stop()
	validatorSrv1.Stop()

	db1, err := bolt.Open(filepath.Join(validatorSrv1.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	db2, err := bolt.Open(filepath.Join(validatorSrvStopped.DataDir(), "consensus", "polybft", "consensusState.db"), 0444, options)
	require.NoError(t, err)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(100), helperCommon.EncodeUint64ToBytes(chainID.Uint64()), false, true)

	compareBucketsFromDBs(t, db1, db2, helperCommon.EncodeUint64ToBytes(chainID.Uint64()), helperCommon.EncodeUint64ToBytes(100), false, true)
}

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
