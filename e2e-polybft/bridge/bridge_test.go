package bridge

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/Ethernal-Tech/ethgo"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/bridge/common"
	bridgeHelper "github.com/0xPolygon/polygon-edge/command/bridge/helper"
	validatorHelper "github.com/0xPolygon/polygon-edge/command/validator/helper"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	polytypes "github.com/0xPolygon/polygon-edge/consensus/polybft/types"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	helperCommon "github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/state/runtime/addresslist"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
)

// TestE2E_Bridge_ExternalChainTokensTransfers is an end-to-end test for the bridge functionality
// involving external chain token transfers. It performs the following operations:
//   - Sets up a test cluster with a bridge and initializes necessary configurations.
//   - Generates multiple receiver accounts and premines them for testing purposes.
//   - Deploys a RootERC20 token contract on the external chain.
//   - Tests the deposit functionality by transferring ERC20 tokens from the external chain
//     to the child chain via the bridge and verifies the balances on the child chain.
//   - Tests the withdrawal functionality by transferring ERC20 tokens back from the child chain
//     to the external chain and verifies the balances on the external chain.
//   - Tests multiple deposit batches per epoch by sending deposits in subsets and verifying
//     the processing of events and state syncs for each batch.
//
// The test ensures that all deposits and withdrawals are processed successfully, and the
// balances on both chains are updated correctly. It also verifies the proper handling of
// multiple deposit batches within the same epoch.
func TestE2E_Bridge_ExternalChainTokensTransfers(t *testing.T) {
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

	validatorSrv := cluster.Servers[0]

	childEthEndpoint := validatorSrv.JSONRPC()

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

	t.Run("bridge ERC20 tokens", func(t *testing.T) {
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

		finalBlockNum := 10 * sprintSize
		// wait for a couple of sprints
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

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

		// WITHDRAW ERC20 TOKENS
		// send withdraw transaction
		for i, senderKey := range receiverKeys {
			err = cluster.Bridges[bridgeOne].Withdraw(
				common.ERC20,
				senderKey,
				receivers[i],
				amounts[i],
				"",
				validatorSrv.JSONRPCAddr(),
				bridgeCfg.InternalERC20PredicateAddr,
				childERC20Token,
				false)
			require.NoError(t, err)
		}

		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := range receivers {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, uint64(i+1), false) {
					return false
				}
			}

			return true
		}))

		for _, receiver := range receivers {
			// assert that receiver's balance on RootERC20 smart contract is as expected
			balance := erc20BalanceOf(t, types.StringToAddress(receiver), rootERC20Token, externalChainTxRelayer)
			require.True(t, bridgeAmount.Cmp(balance) == 0)
		}
	})

	t.Run("multiple deposit batches per epoch", func(t *testing.T) {
		const (
			depositsSubset = 1
		)

		internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
		require.NoError(t, err)

		lastCommittedIDMethod := contractsapi.BridgeStorage.Abi.GetMethod("lastCommittedE2I")
		lastCommittedIDInput, err := lastCommittedIDMethod.Encode([]interface{}{chainID.Uint64()})
		require.NoError(t, err)

		// check for last committed id
		commitmentIDRaw, err := internalChainTxRelayer.Call(types.ZeroAddress, contracts.BridgeStorageContract, lastCommittedIDInput)
		require.NoError(t, err)

		_, err = helperCommon.ParseUint64orHex(&commitmentIDRaw)
		require.NoError(t, err)

		initialBlockNum, err := childEthEndpoint.BlockNumber()
		require.NoError(t, err)

		// wait for next sprint block as the starting point,
		// in order to be able to make assertions against blocks offsetted by sprints
		initialBlockNum = initialBlockNum + sprintSize - (initialBlockNum % sprintSize)
		require.NoError(t, cluster.WaitForBlock(initialBlockNum, 1*time.Minute))

		// send two transactions to the bridge so that we have a minimal batch
		require.NoError(t, cluster.Bridges[bridgeOne].Deposit(
			common.ERC20,
			rootERC20Token,
			bridgeCfg.ExternalERC20PredicateAddr,
			bridgeHelper.TestAccountPrivKey,
			strings.Join(receivers[:depositsSubset], ","),
			strings.Join(amounts[:depositsSubset], ","),
			"",
			cluster.Bridges[bridgeOne].JSONRPCAddr(),
			bridgeHelper.TestAccountPrivKey,
			false),
		)

		// wait for a few more sprints
		midBlockNumber := initialBlockNum + 2*sprintSize
		require.NoError(t, cluster.WaitForBlock(midBlockNumber, 2*time.Minute))

		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := range receivers[:depositsSubset] {
				if !isEventProcessed(t,
					bridgeCfg.InternalGatewayAddr,
					internalChainTxRelayer,
					// this sum represent minimal value for event id based on earlier events
					uint64(numberOfMapTokenEvent+transfersCount+depositsSubset+i), false) {
					return false
				}
			}

			return true
		}))

		// send some more transactions to the bridge to build another batch in epoch
		require.NoError(t, cluster.Bridges[bridgeOne].Deposit(
			common.ERC20,
			rootERC20Token,
			bridgeCfg.ExternalERC20PredicateAddr,
			bridgeHelper.TestAccountPrivKey,
			strings.Join(receivers[depositsSubset:], ","),
			strings.Join(amounts[depositsSubset:], ","),
			"",
			cluster.Bridges[bridgeOne].JSONRPCAddr(),
			bridgeHelper.TestAccountPrivKey,
			false),
		)

		finalBlockNum := midBlockNumber + 5*sprintSize
		// wait for a few more sprints
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

		require.NoError(t, cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := range receivers[depositsSubset:] {
				if !isEventProcessed(t,
					bridgeCfg.InternalGatewayAddr,
					internalChainTxRelayer,
					// this sum represent minimal value for event id based on earlier events
					uint64(numberOfMapTokenEvent+transfersCount+2*depositsSubset+i), false) {
					return false
				}
			}

			return true
		}))

		finalBlockNum, err = childEthEndpoint.BlockNumber()
		require.NoError(t, err)

		// the transactions are mined and state syncs should be executed by the relayer
		// and there should be a success events
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), initialBlockNum, finalBlockNum, childEthEndpoint)
		require.NoError(t, err)

		// assert that all state syncs are executed successfully
		assertBridgeEventResultSuccess(t, logs, transfersCount)
	})
}

// TestE2E_Bridge_ERC721Transfer is an end-to-end test for ERC721 token transfers
// across a bridge between two blockchain networks. The test performs the following steps:
//   - Initializes a test cluster with a specified number of nodes, epoch size, and bridge configuration.
//   - Generates receiver accounts and token IDs for the transfer.
//   - Deploys a root ERC721 token contract on the external chain.
//   - Deposits ERC721 tokens from the external chain to the child chain via the bridge.
//   - Waits for the transactions to be processed and verifies the success of the deposit events.
//   - Retrieves and validates the child token address on both chains to ensure consistency.
//   - Verifies that the deposited tokens are owned by the expected accounts on the child chain.
//   - Performs withdrawals of the ERC721 tokens from the child chain back to the external chain.
//   - Waits for the withdrawal events to be processed and verifies the success of the withdrawal events.
//   - Asserts that the withdrawn tokens are owned by the expected accounts on the root chain.
//
// This test ensures the correctness of the ERC721 token transfer functionality across the bridge,
// including deposit and withdrawal operations, event processing, and ownership validation.
func TestE2E_Bridge_ERC721Transfer(t *testing.T) {
	const (
		transfersCount       = 4
		epochSize            = 5
		numberOfAttempts     = 4
		stateSyncedLogsCount = 2
		numberOfBridges      = 1
	)

	receiverKeys := make([]string, transfersCount)
	receivers := make([]string, transfersCount)
	receiversAddrs := make([]types.Address, transfersCount)
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

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithEpochSize(epochSize),
		framework.WithPremine(receiversAddrs...),
		framework.WithBridges(numberOfBridges),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}
		}),
	)
	defer cluster.Stop()

	bridgeOne := 0

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[bridgeOne].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	externalChainDeployer, err := bridgeHelper.DecodePrivateKey("")
	require.NoError(t, err)

	deployTx := types.NewTx(&types.LegacyTx{
		BaseTx: &types.BaseTx{
			To:    nil,
			Input: contractsapi.RootERC721.Bytecode,
		},
	})

	// deploy root ERC 721 token
	receipt, err := externalChainTxRelayer.SendTransaction(deployTx, externalChainDeployer)
	require.NoError(t, err)

	rootERC721Addr := types.Address(receipt.ContractAddress)

	// DEPOSIT ERC721 TOKENS
	// send a few transactions to the bridge
	require.NoError(
		t,
		cluster.Bridges[bridgeOne].Deposit(
			common.ERC721,
			rootERC721Addr,
			bridgeCfg.ExternalERC721PredicateAddr,
			bridgeHelper.TestAccountPrivKey,
			strings.Join(receivers, ","),
			"",
			strings.Join(tokenIDs, ","),
			cluster.Bridges[bridgeOne].JSONRPCAddr(),
			bridgeHelper.TestAccountPrivKey,
			false),
	)

	// wait for a few more sprints
	require.NoError(t, cluster.WaitForBlock(50, 4*time.Minute))

	validatorSrv := cluster.Servers[0]
	childEthEndpoint := validatorSrv.JSONRPC()

	// the transactions are processed and there should be a success events
	var bridgeMessageResult contractsapi.BridgeMessageResultEvent

	for i := 0; i < numberOfAttempts; i++ {
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, uint64(50+i*epochSize), childEthEndpoint)
		require.NoError(t, err)

		if len(logs) == stateSyncedLogsCount || i == numberOfAttempts-1 {
			// assert that all deposits are executed successfully.
			// All deposits are sent using a single transaction, so arbitrary message bridge emits two state sync events:
			// MAP_TOKEN_SIG and DEPOSIT_BATCH_SIG state sync events
			assertBridgeEventResultSuccess(t, logs, stateSyncedLogsCount)

			break
		}

		require.NoError(t, cluster.WaitForBlock(uint64(50+(i+1)*epochSize), 1*time.Minute))
	}

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
	require.NoError(t, err)

	// retrieve child token address (from both chains, and assert they are the same)
	externalChildTokenAddr := getChildToken(t, contractsapi.RootERC721Predicate.Abi, bridgeCfg.ExternalERC721PredicateAddr,
		rootERC721Addr, externalChainTxRelayer)
	internalChildTokenAddr := getChildToken(t, contractsapi.ChildERC721Predicate.Abi, bridgeCfg.InternalERC721PredicateAddr,
		rootERC721Addr, txRelayer)

	t.Log("External child token", externalChildTokenAddr)
	t.Log("Internal child token", internalChildTokenAddr)
	require.Equal(t, externalChildTokenAddr, internalChildTokenAddr)

	for i, receiver := range receiversAddrs {
		owner := erc721OwnerOf(t, big.NewInt(int64(i)), internalChildTokenAddr, txRelayer)
		require.Equal(t, receiver, owner)
	}

	t.Log("Deposits were successfully processed")

	// WITHDRAW ERC721 TOKENS
	for i, receiverKey := range receiverKeys {
		// send withdraw transactions
		err = cluster.Bridges[bridgeOne].Withdraw(
			common.ERC721,
			receiverKey,
			receivers[i],
			"",
			tokenIDs[i],
			validatorSrv.JSONRPCAddr(),
			bridgeCfg.InternalERC721PredicateAddr,
			internalChildTokenAddr,
			false)
		require.NoError(t, err)
	}

	currentBlock, err := childEthEndpoint.GetBlockByNumber(jsonrpc.LatestBlockNumber, false)
	require.NoError(t, err)

	currentExtra, err := polytypes.GetIbftExtra(currentBlock.Header.ExtraData)
	require.NoError(t, err)

	t.Logf("Latest block number: %d, epoch number: %d\n", currentBlock.Number(), currentExtra.BlockMetaData.EpochNumber)

	require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
		for i := 1; i <= transfersCount; i++ {
			if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, uint64(i), false) {
				return false
			}
		}

		return true
	}))

	// assert that owners of given token ids are the accounts on the root chain ERC 721 token
	for i, receiver := range receiversAddrs {
		t.Log("ERC721 OWNER", i)
		owner := erc721OwnerOf(t, big.NewInt(int64(i)), rootERC721Addr, externalChainTxRelayer)
		require.Equal(t, receiver, owner)
	}
}

// TestE2E_Bridge_ERC1155Transfer is an end-to-end test for ERC1155 token transfers
// using a bridge between two blockchain networks. The test performs the following steps:
//   - Initializes a test cluster with a specified number of nodes, epoch size, and bridge configuration.
//   - Generates receiver accounts and prepares data for multiple ERC1155 token transfers.
//   - Deploys a RootERC1155 token contract on the external chain.
//   - Deposits ERC1155 tokens from the external chain to the child chain via the bridge.
//   - Waits for the transactions to be processed and verifies the success events emitted by the bridge.
//   - Retrieves the child token address on both L1 and L2 and ensures they match.
//   - Verifies that the balances of the receivers on the child chain have increased by the deposited amounts.
//   - Performs withdrawals of ERC1155 tokens from the child chain back to the external chain.
//   - Waits for the withdrawal events to be processed and verifies the balances of the receivers on the RootERC1155 contract.
//
// The test ensures that the bridge correctly handles ERC1155 token deposits and withdrawals,
// and that the balances on both chains are updated as expected.
func TestE2E_Bridge_ERC1155Transfer(t *testing.T) {
	const (
		transfersCount       = 5
		amount               = 100
		epochSize            = 5
		numberOfAttempts     = 4
		stateSyncedLogsCount = 2
		numberOfBridges      = 1
	)

	receiverKeys := make([]string, transfersCount)
	receivers := make([]string, transfersCount)
	receiversAddrs := make([]types.Address, transfersCount)
	amounts := make([]string, transfersCount)
	tokenIDs := make([]string, transfersCount)

	for i := 0; i < transfersCount; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		rawKey, err := key.MarshallPrivateKey()
		require.NoError(t, err)

		receiverKeys[i] = hex.EncodeToString(rawKey)
		receivers[i] = key.Address().String()
		receiversAddrs[i] = key.Address()
		amounts[i] = fmt.Sprintf("%d", amount)
		tokenIDs[i] = fmt.Sprintf("%d", i+1)

		t.Logf("Receiver#%d=%s\n", i+1, receivers[i])
	}

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithNumBlockConfirmations(0),
		framework.WithEpochSize(epochSize),
		framework.WithPremine(receiversAddrs...),
		framework.WithBridges(numberOfBridges),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}
		}),
	)
	defer cluster.Stop()

	bridgeOne := 0

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[bridgeOne].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	externalChainDeployer, err := bridgeHelper.DecodePrivateKey("")
	require.NoError(t, err)

	deployTx := types.NewTx(&types.LegacyTx{
		BaseTx: &types.BaseTx{
			To:    nil,
			Input: contractsapi.RootERC1155.Bytecode,
		},
	})

	// deploy root ERC 1155 token
	receipt, err := externalChainTxRelayer.SendTransaction(deployTx, externalChainDeployer)
	require.NoError(t, err)

	rootERC1155Addr := types.Address(receipt.ContractAddress)

	// DEPOSIT ERC1155 TOKENS
	// send a few transactions to the bridge
	require.NoError(
		t,
		cluster.Bridges[bridgeOne].Deposit(
			common.ERC1155,
			rootERC1155Addr,
			bridgeCfg.ExternalERC1155PredicateAddr,
			bridgeHelper.TestAccountPrivKey,
			strings.Join(receivers, ","),
			strings.Join(amounts, ","),
			strings.Join(tokenIDs, ","),
			cluster.Bridges[bridgeOne].JSONRPCAddr(),
			bridgeHelper.TestAccountPrivKey,
			false),
	)

	// wait for a few more sprints
	require.NoError(t, cluster.WaitForBlock(50, 4*time.Minute))

	validatorSrv := cluster.Servers[0]
	childEthEndpoint := validatorSrv.JSONRPC()

	// the transactions are processed and there should be a success events
	var bridgeMessageResult contractsapi.BridgeMessageResultEvent

	for i := 0; i < numberOfAttempts; i++ {
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, uint64(50+i*epochSize), childEthEndpoint)
		require.NoError(t, err)

		if len(logs) == stateSyncedLogsCount || i == numberOfAttempts-1 {
			// assert that all deposits are executed successfully.
			// All deposits are sent using a single transaction, so arbitrary message bridge emits two state sync events:
			// MAP_TOKEN_SIG and DEPOSIT_BATCH_SIG state sync events
			assertBridgeEventResultSuccess(t, logs, stateSyncedLogsCount)

			break
		}

		require.NoError(t, cluster.WaitForBlock(uint64(50+(i+1)*epochSize), 1*time.Minute))
	}

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
	require.NoError(t, err)

	// retrieve child token address
	l1ChildTokenAddr := getChildToken(t, contractsapi.RootERC1155Predicate.Abi, bridgeCfg.ExternalERC1155PredicateAddr,
		rootERC1155Addr, externalChainTxRelayer)
	l2ChildTokenAddr := getChildToken(t, contractsapi.ChildERC1155Predicate.Abi, bridgeCfg.InternalERC1155PredicateAddr,
		rootERC1155Addr, txRelayer)

	t.Log("L1 child token", l1ChildTokenAddr)
	t.Log("L2 child token", l2ChildTokenAddr)
	require.Equal(t, l1ChildTokenAddr, l2ChildTokenAddr)

	// check receivers balances got increased by deposited amount
	for i, receiver := range receivers {
		balanceOfFn := &contractsapi.BalanceOfChildERC1155Fn{
			Account: types.StringToAddress(receiver),
			ID:      big.NewInt(int64(i + 1)),
		}

		balanceInput, err := balanceOfFn.EncodeAbi()
		require.NoError(t, err)

		balanceRaw, err := txRelayer.Call(types.ZeroAddress, l2ChildTokenAddr, balanceInput)
		require.NoError(t, err)

		balance, err := helperCommon.ParseUint256orHex(&balanceRaw)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(int64(amount)), balance)
	}

	t.Log("Deposits were successfully processed")

	// WITHDRAW ERC1155 TOKENS
	senderAccount, err := validatorHelper.GetAccountFromDir(cluster.Servers[0].DataDir())
	require.NoError(t, err)

	t.Logf("Withdraw sender: %s\n", senderAccount.Ecdsa.Address())

	for i, receiverKey := range receiverKeys {
		// send withdraw transactions
		err = cluster.Bridges[bridgeOne].Withdraw(
			common.ERC1155,
			receiverKey,
			receivers[i],
			amounts[i],
			tokenIDs[i],
			validatorSrv.JSONRPCAddr(),
			bridgeCfg.InternalERC1155PredicateAddr,
			l2ChildTokenAddr,
			false)
		require.NoError(t, err)
	}

	require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
		for i := 1; i <= transfersCount; i++ {
			if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, uint64(i), false) {
				return false
			}
		}

		return true
	}))

	// assert that receiver's balances on RootERC1155 smart contract are expected
	for i, receiver := range receivers {
		balanceOfFn := &contractsapi.BalanceOfRootERC1155Fn{
			Account: types.StringToAddress(receiver),
			ID:      big.NewInt(int64(i + 1)),
		}

		balanceInput, err := balanceOfFn.EncodeAbi()
		require.NoError(t, err)

		balanceRaw, err := externalChainTxRelayer.Call(types.ZeroAddress, rootERC1155Addr, balanceInput)
		require.NoError(t, err)

		balance, err := helperCommon.ParseUint256orHex(&balanceRaw)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(amount), balance)
	}
}

// TestE2E_Bridge_InternalChainTokensTransfer is a test that ensures the correct behavior of token transfers
// between an external chain and an internal bridge, covering both native tokens (ERC20) and NFTs (ERC721).
// The test includes multiple key steps to validate the system's functionality, such as token deposits, withdrawals,
// blocklist/allowlist manipulations, and verification of token ownership across chains. The test covers:
//   - Deposit and withdrawal of native ERC20 tokens (mintable on the root chain) into the bridge and on the child chain.
//   - ERC721 token minting, deposit, and withdrawal, ensuring the correct ownership of tokens after the operations.
//   - Manipulation of the blocklist and allowlist, ensuring that only eligible accounts can perform deposit and withdrawal operations.
//   - Verifying that the child chain's balances and owners are correctly updated after token transfers.
func TestE2E_Bridge_InternalChainTokensTransfer(t *testing.T) {
	const (
		transfersCount = uint64(4)
		amount         = 100
		// make epoch size long enough, so that all exit events are processed within the same epoch
		epochSize        = 30
		sprintSize       = uint64(5)
		numberOfAttempts = 4
		numberOfBridges  = 1
	)

	// init private keys and amounts
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

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	// setup cluster
	cluster := framework.NewTestCluster(t, 5,
		framework.WithNumBlockConfirmations(0),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBlockListAdmin(adminAddr),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithPremine(append(depositors, adminAddr)...)) //nolint:makezero
	defer cluster.Stop()

	bridgeOne := 0

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]
	childEthEndpoint := validatorSrv.JSONRPC()

	// fund accounts on external
	require.NoError(t, validatorSrv.ExternalChainFundFor(depositors, funds, uint64(bridgeOne)))

	cluster.WaitForReady(t)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[bridgeOne].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
	require.NoError(t, err)

	t.Run("bridge native tokens", func(t *testing.T) {
		// rootToken represents deposit token (basically native mintable token from the Supernets)
		rootToken := contracts.NativeERC20TokenContract

		// block list first depositor
		setAccessListRole(t, cluster, contracts.BlockListBridgeAddr, depositors[0], addresslist.EnabledRole, admin)

		// try sending a single native token deposit transaction
		// it should fail, because first depositor is added to bridge transactions block list
		err = cluster.Bridges[bridgeOne].Deposit(
			common.ERC20,
			rootToken,
			bridgeCfg.InternalMintableERC20PredicateAddr,
			depositorKeys[0],
			depositors[0].String(),
			amounts[0],
			"",
			validatorSrv.JSONRPCAddr(),
			"",
			true)
		require.Error(t, err)

		// remove first depositor from bridge transactions block list
		setAccessListRole(t, cluster, contracts.BlockListBridgeAddr, depositors[0], addresslist.NoRole, admin)

		// allow list each depositor and make sure deposit is successfully executed
		for i, key := range depositorKeys {
			// make sure deposit is successfully executed
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

		// first exit event is mapping child token on a rootchain
		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := uint64(1); i <= transfersCount+1; i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		// retrieve child mintable token address from both chains and make sure they are the same
		l1ChildToken := getChildToken(t, contractsapi.ChildERC20Predicate.Abi, bridgeCfg.ExternalMintableERC20PredicateAddr,
			rootToken, externalChainTxRelayer)
		l2ChildToken := getChildToken(t, contractsapi.RootERC20Predicate.Abi, bridgeCfg.InternalMintableERC20PredicateAddr,
			rootToken, internalChainTxRelayer)

		t.Log("L1 child token", l1ChildToken)
		t.Log("L2 child token", l2ChildToken)
		require.Equal(t, l1ChildToken, l2ChildToken)

		// check that balances on external chain have increased by deposited amounts
		for _, depositor := range depositors {
			balance := erc20BalanceOf(t, depositor, l1ChildToken, externalChainTxRelayer)
			require.Equal(t, big.NewInt(amount), balance)
		}

		balancesBefore := make([]*big.Int, transfersCount)
		for i := uint64(0); i < transfersCount; i++ {
			balancesBefore[i], err = childEthEndpoint.GetBalance(depositors[i], jsonrpc.LatestBlockNumberOrHash)
			require.NoError(t, err)
		}

		// withdraw child token on the external chain
		for i, depositorKey := range depositorKeys {
			err = cluster.Bridges[bridgeOne].Withdraw(
				common.ERC20,
				depositorKey,
				depositors[i].String(),
				amounts[i],
				"",
				cluster.Bridges[bridgeOne].JSONRPCAddr(),
				bridgeCfg.ExternalMintableERC20PredicateAddr,
				l1ChildToken,
				true)
			require.NoError(t, err)
		}

		allSuccessful := false

		for it := 0; it < numberOfAttempts && !allSuccessful; it++ {
			blockNum, err := childEthEndpoint.BlockNumber()
			require.NoError(t, err)

			// wait a couple of sprints to finalize state sync events
			require.NoError(t, cluster.WaitForBlock(blockNum+3*sprintSize, 2*time.Minute))

			allSuccessful = true

			// check that balances on the child chain are correct
			for i, receiver := range depositors {
				balance := erc20BalanceOf(t, receiver, contracts.NativeERC20TokenContract, internalChainTxRelayer)
				t.Log("Attempt", it+1, "Balance before", balancesBefore[i], "Balance after", balance)

				if balance.Cmp(new(big.Int).Add(balancesBefore[i], big.NewInt(amount))) != 0 {
					allSuccessful = false

					break
				}
			}
		}

		require.True(t, allSuccessful)
	})

	t.Run("bridge ERC 721 tokens", func(t *testing.T) {
		erc721DeployTxn := cluster.Deploy(t, admin, contractsapi.RootERC721.Bytecode)
		require.True(t, erc721DeployTxn.Succeed())
		rootERC721Token := types.Address(erc721DeployTxn.Receipt().ContractAddress)

		for _, depositor := range depositors {
			// mint all the depositors in advance
			mintFn := &contractsapi.MintRootERC721Fn{To: depositor}
			mintInput, err := mintFn.EncodeAbi()
			require.NoError(t, err)

			mintTxn := cluster.MethodTxn(t, admin, rootERC721Token, mintInput)
			require.True(t, mintTxn.Succeed())

			// add all depositors to bride block list
			setAccessListRole(t, cluster, contracts.BlockListBridgeAddr, depositor, addresslist.EnabledRole, admin)
		}

		// deposit should fail because depositors are in bridge block list
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
			// remove all depositors from the bridge block list
			setAccessListRole(t, cluster, contracts.BlockListBridgeAddr, depositors[i], addresslist.NoRole, admin)

			// deposit (without minting, as it was already done beforehand)
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

		// first exit event is mapping child token on a rootchain
		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := uint64(1); i <= transfersCount+1; i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		// retrieve child token addresses on both chains and make sure they are the same
		l1ChildToken := getChildToken(t, contractsapi.ChildERC721Predicate.Abi, bridgeCfg.ExternalMintableERC721PredicateAddr,
			rootERC721Token, externalChainTxRelayer)
		l2ChildToken := getChildToken(t, contractsapi.RootERC721Predicate.Abi, bridgeCfg.InternalMintableERC721PredicateAddr,
			rootERC721Token, internalChainTxRelayer)

		t.Log("L1 child token", l1ChildToken)
		t.Log("L2 child token", l2ChildToken)
		require.Equal(t, l1ChildToken, l2ChildToken)

		blockNumber, err := childEthEndpoint.BlockNumber()
		require.NoError(t, err)

		require.NoError(t, cluster.WaitForBlock(blockNumber+epochSize, 2*time.Minute))

		// check owner on the external chain
		for i := uint64(0); i < transfersCount; i++ {
			owner := erc721OwnerOf(t, new(big.Int).SetUint64(i), l1ChildToken, externalChainTxRelayer)
			t.Log("ChildERC721 owner", owner)
			require.Equal(t, depositors[i], owner)
		}

		// withdraw tokens
		for i, depositorKey := range depositorKeys {
			err = cluster.Bridges[bridgeOne].Withdraw(
				common.ERC721,
				depositorKey,
				depositors[i].String(),
				"",
				fmt.Sprintf("%d", i),
				cluster.Bridges[bridgeOne].JSONRPCAddr(),
				bridgeCfg.ExternalMintableERC721PredicateAddr,
				l1ChildToken,
				true)
			require.NoError(t, err)
		}

		allSuccessful := false

		for it := 0; it < numberOfAttempts && !allSuccessful; it++ {
			internalChainBlockNum, err := childEthEndpoint.BlockNumber()
			require.NoError(t, err)

			// wait for commitment execution
			require.NoError(t, cluster.WaitForBlock(internalChainBlockNum+3*sprintSize, 2*time.Minute))

			allSuccessful = true
			// check owners on the child chain
			for i, receiver := range depositors {
				owner := erc721OwnerOf(t, big.NewInt(int64(i)), rootERC721Token, internalChainTxRelayer)
				t.Log("Attempt:", it+1, " Owner:", owner, " Receiver:", receiver)

				if receiver != owner {
					allSuccessful = false

					break
				}
			}
		}

		require.True(t, allSuccessful)
	})
}

// TestE2E_Bridge_Transfers_AccessLists is a test that ensures the functionality of ERC20 token transfers
// on a bridge network, validating both deposit and withdrawal scenarios with access control via allowlists and blocklists.
// The test involves multiple steps including:
//   - Setting up a cluster with specific configurations, such as the number of blocks and the interval between them.
//   - Deploying a Root ERC20 token on an external chain and configuring the test environment to interact with it.
//   - Depositing ERC20 tokens into the bridge from multiple receivers, ensuring the transfers are successful and that balances are updated correctly.
//   - Attempting a withdrawal before and after adding an account to the bridge allowlist, ensuring that access control works as expected.
//   - Verifying that withdrawals are blocked when the account is added to the blocklist.
//   - Validating the correct balances after all operations have been completed, ensuring that all transfers and withdrawals were processed accurately.
func TestE2E_Bridge_Transfers_AccessLists(t *testing.T) {
	var (
		transfersCount = 5
		depositAmount  = ethgo.Ether(5)
		withdrawAmount = ethgo.Ether(1)
		// make epoch size long enough, so that all exit events are processed within the same epoch
		epochSize       = 40
		sprintSize      = uint64(5)
		numberOfBridges = uint64(1)
	)

	receivers := make([]string, transfersCount)
	depositAmounts := make([]string, transfersCount)
	withdrawAmounts := make([]string, transfersCount)

	admin, _ := crypto.GenerateECDSAKey()
	adminAddr := admin.Address()

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithNumBlockConfirmations(0),
		framework.WithEpochSize(epochSize),
		framework.WithTestRewardToken(),
		framework.WithRootTrackerPollInterval(3*time.Second),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeAllowListAdmin(adminAddr),
		framework.WithBridgeBlockListAdmin(adminAddr),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(a []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(a); i++ {
				receivers[i] = a[i].String()
				depositAmounts[i] = fmt.Sprintf("%d", depositAmount)
				withdrawAmounts[i] = fmt.Sprintf("%d", withdrawAmount)

				t.Logf("Receiver#%d=%s\n", i+1, receivers[i])

				// premine access list admin account
				tcc.Premine = append(tcc.Premine, adminAddr.String())
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}
		}),
	)
	defer cluster.Stop()

	bridgeOne := 0

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]
	childEthEndpoint := validatorSrv.JSONRPC()
	relayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(childEthEndpoint))
	require.NoError(t, err)

	senderAccount, err := validatorHelper.GetAccountFromDir(validatorSrv.DataDir())
	require.NoError(t, err)

	// fund admin on external chain
	require.NoError(t, validatorSrv.ExternalChainFundFor([]types.Address{adminAddr}, []*big.Int{ethgo.Ether(1)}, uint64(bridgeOne)))

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[bridgeOne].JSONRPCAddr()))
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

	// deploy root erc20 token
	receipt, err := externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)
	rootERC20Token := types.Address(receipt.ContractAddress)

	t.Run("bridge ERC 20 tokens", func(t *testing.T) {
		// DEPOSIT ERC20 TOKENS
		// send a few transactions to the bridge
		require.NoError(
			t,
			cluster.Bridges[bridgeOne].Deposit(
				common.ERC20,
				rootERC20Token,
				bridgeCfg.ExternalERC20PredicateAddr,
				bridgeHelper.TestAccountPrivKey,
				strings.Join(receivers, ","),
				strings.Join(depositAmounts, ","),
				"",
				cluster.Bridges[bridgeOne].JSONRPCAddr(),
				bridgeHelper.TestAccountPrivKey,
				false),
		)

		finalBlockNum := 10 * sprintSize
		// wait for a couple of sprints
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

		var bridgeMessageResult contractsapi.BridgeMessageResultEvent

		// the transactions are processed and there should be a success events
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, finalBlockNum, childEthEndpoint)
		require.NoError(t, err)

		// assert that all deposits are executed successfully
		// (token mapping and transferCount of deposits)
		assertBridgeEventResultSuccess(t, logs, transfersCount+1)

		// get child token address
		childERC20Token := getChildToken(t, contractsapi.RootERC20Predicate.Abi,
			bridgeCfg.ExternalERC20PredicateAddr, rootERC20Token, externalChainTxRelayer)

		for _, receiver := range receivers {
			balance := erc20BalanceOf(t, types.StringToAddress(receiver), childERC20Token, relayer)
			require.Equal(t, depositAmount, balance)
		}

		t.Log("Deposits were successfully processed")

		oldBalances := map[types.Address]*big.Int{}

		for _, receiver := range receivers {
			balance := erc20BalanceOf(t, types.StringToAddress(receiver), rootERC20Token, externalChainTxRelayer)
			oldBalances[types.StringToAddress(receiver)] = balance
		}

		// WITHDRAW ERC20 TOKENS
		rawKey, err := senderAccount.Ecdsa.MarshallPrivateKey()
		require.NoError(t, err)

		// send withdraw transaction.
		// It should fail because sender is not allow-listed.
		err = cluster.Bridges[bridgeOne].Withdraw(
			common.ERC20,
			hex.EncodeToString(rawKey),
			strings.Join(receivers, ","),
			strings.Join(withdrawAmounts, ","),
			"",
			validatorSrv.JSONRPCAddr(),
			bridgeCfg.InternalERC20PredicateAddr,
			childERC20Token,
			false)
		require.Error(t, err)

		// add account to bridge allow list
		setAccessListRole(t, cluster, contracts.AllowListBridgeAddr, senderAccount.Address(), addresslist.EnabledRole, admin)

		// try to withdraw again
		err = cluster.Bridges[bridgeOne].Withdraw(
			common.ERC20,
			hex.EncodeToString(rawKey),
			strings.Join(receivers, ","),
			strings.Join(withdrawAmounts, ","),
			"",
			validatorSrv.JSONRPCAddr(),
			bridgeCfg.InternalERC20PredicateAddr,
			childERC20Token,
			false)
		require.NoError(t, err)

		// add account to bridge block list
		setAccessListRole(t, cluster, contracts.BlockListBridgeAddr, senderAccount.Address(), addresslist.EnabledRole, admin)

		// it should fail now because sender accont is in the block list
		err = cluster.Bridges[bridgeOne].Withdraw(
			common.ERC20,
			hex.EncodeToString(rawKey),
			strings.Join(receivers, ","),
			strings.Join(withdrawAmounts, ","),
			"",
			validatorSrv.JSONRPCAddr(),
			bridgeCfg.InternalERC20PredicateAddr,
			contracts.NativeERC20TokenContract,
			false)
		require.ErrorContains(t, err, "failed to send withdraw transaction")

		currentBlock, err := childEthEndpoint.GetBlockByNumber(jsonrpc.LatestBlockNumber, false)
		require.NoError(t, err)

		currentExtra, err := polytypes.GetIbftExtra(currentBlock.Header.ExtraData)
		require.NoError(t, err)

		t.Logf("Latest block number: %d, epoch number: %d\n", currentBlock.Number(), currentExtra.BlockMetaData.EpochNumber)

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for i := uint64(1); i <= uint64(transfersCount); i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}))

		// assert that receiver's balances on RootERC20 smart contract are expected
		for _, receiver := range receivers {
			balance := erc20BalanceOf(t, types.StringToAddress(receiver), rootERC20Token, externalChainTxRelayer)
			require.Equal(t, oldBalances[types.StringToAddress(receiver)].Add(
				oldBalances[types.StringToAddress(receiver)], withdrawAmount), balance)
		}
	})
}

// TestE2E_Bridge_NonMintableERC20Token_WithPremine tests the end-to-end bridging functionality between
// a non-mintable ERC20 token and a blockchain bridge. The test involves a series of operations, such as
// premining tokens to specific addresses, checking balances, performing withdrawals, deposits, and ensuring
// the correct execution of bridge events. The test also handles the London fork (EIP-1559) and simulates various
// user scenarios, such as transferring tokens and withdrawing them from the bridge. This test includes multiple
// validations for both validator and non-validator addresses to verify correct token balances and bridge event
// processing over several epochs.
//
// It performs the following actions:
//   - Initializes a test cluster with several configurations, including a non-mintable ERC20 token and premined
//     tokens for specific addresses.
//   - Checks the initial token balances of premined addresses on both root and child chains.
//   - Verifies that validators and non-validators can withdraw premined tokens from the bridge and checks their
//     balances after withdrawal.
//   - Simulates deposits to validator and non-validator addresses and waits for bridge events to process.
//   - Verifies the success or failure of transfers, including edge cases such as transferring more native tokens
//     than the source address holds.
func TestE2E_Bridge_NonMintableERC20Token_WithPremine(t *testing.T) {
	var (
		stateSyncedLogsCount  = 2
		epochSize             = uint64(10)
		numberOfAttempts      = uint64(4)
		numBlockConfirmations = uint64(2)
		bridgeEvents          = uint64(2)
		tokensToTransfer      = ethgo.Gwei(10)
		tenMilionTokens       = ethgo.Ether(10000000)
		bigZero               = big.NewInt(0)
		numberOfBridges       = 1
	)

	nonValidatorKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	nonValidatorKeyRaw, err := nonValidatorKey.MarshallPrivateKey()
	require.NoError(t, err)

	rewardWalletKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	rewardWalletKeyRaw, err := rewardWalletKey.MarshallPrivateKey()
	require.NoError(t, err)

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	relayerPrivateKeyRaw, err := relayerPrivateKey.MarshallPrivateKey()
	require.NoError(t, err)

	// start cluster with default, non-mintable native erc20 root token
	// with london fork enabled
	cluster := framework.NewTestCluster(t, 5,
		framework.WithBridges(uint64(numberOfBridges)),
		framework.WithEpochSize(int(epochSize)),
		framework.WithNumBlockConfirmations(numBlockConfirmations),
		framework.WithNativeTokenConfig(nativeTokenNonMintableConfig),
		framework.WithBridgeBatchThreshold(25),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		// this enables London (EIP-1559) fork
		framework.WithBurnContract(&polycfg.BurnContractInfo{
			BlockNumber: 0,
			Address:     types.StringToAddress("0xBurnContractAddress")}),
		framework.WithSecretsCallback(func(_ []types.Address, tcc *framework.TestClusterConfig) {
			nonValidatorKeyString := hex.EncodeToString(nonValidatorKeyRaw)
			rewardWalletKeyString := hex.EncodeToString(rewardWalletKeyRaw)
			relayerPrivateKeyString := hex.EncodeToString(relayerPrivateKeyRaw)

			// do premine to a non validator address
			tcc.Premine = append(tcc.Premine,
				fmt.Sprintf("%s:%s:%s",
					nonValidatorKey.Address(),
					new(big.Int).Mul(big.NewInt(10), tenMilionTokens).String(),
					nonValidatorKeyString))

			// do premine to reward wallet address
			tcc.Premine = append(tcc.Premine,
				fmt.Sprintf("%s:%s:%s",
					rewardWalletKey.Address(),
					command.DefaultPremineBalance.String(),
					rewardWalletKeyString))

			// do premine to reward wallet address
			tcc.Premine = append(tcc.Premine,
				fmt.Sprintf("%s:%s:%s",
					relayerPrivateKey.Address(),
					command.DefaultPremineBalance.String(),
					relayerPrivateKeyString))
		}),
	)
	defer cluster.Stop()

	bridgeOne := 0

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[bridgeOne].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	childEthEndpoint := cluster.Servers[0].JSONRPC()

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	checkBalancesFn := func(address types.Address, rootExpected, childExpected *big.Int, isValidator bool) {
		offset := ethgo.Gwei(100)
		expectedValue := new(big.Int)

		t.Log("Checking balance of native ERC20 token on root and child", "Address", address,
			"Root expected", rootExpected, "Child Expected", childExpected)

		balance := erc20BalanceOf(t, address,
			bridgeCfg.ExternalNativeERC20Addr, externalChainTxRelayer)
		t.Log("Balance of native ERC20 token on root", balance, "Address", address)
		require.Equal(t, rootExpected, balance)

		balance, err = childEthEndpoint.GetBalance(address, jsonrpc.LatestBlockNumberOrHash)
		require.NoError(t, err)
		t.Log("Balance of native ERC20 token on child", balance, "Address", address)

		if isValidator {
			require.True(t, balance.Cmp(childExpected) >= 0) // because of London fork
		} else {
			// this check is implemented because non-validators incur fees, potentially resulting in a balance lower than anticipated
			require.True(t, balance.Cmp(expectedValue.Sub(childExpected, offset)) >= 0)
		}
	}

	t.Run("check the balances at the beginning", func(t *testing.T) {
		// check the balances on root and child at the beginning to see if they are as expected
		checkBalancesFn(nonValidatorKey.Address(), bigZero, command.DefaultPremineBalance, false)
		checkBalancesFn(rewardWalletKey.Address(), bigZero, command.DefaultPremineBalance, true)

		validatorsExpectedBalance := new(big.Int).Sub(command.DefaultPremineBalance, command.DefaultStake)

		for _, server := range cluster.Servers {
			validatorAccount, err := validatorHelper.GetAccountFromDir(server.DataDir())
			require.NoError(t, err)

			checkBalancesFn(validatorAccount.Address(), bigZero, validatorsExpectedBalance, true)
		}
	})

	// this test case will check first if they can withdraw some of the premined amount of non-mintable token
	t.Run("do a withdraw for premined validator address and premined non-validator address", func(t *testing.T) {
		waitOneBlockFn := func(server *framework.TestServer) {
			currentBlock, err := server.JSONRPC().GetBlockByNumber(jsonrpc.LatestBlockNumber, false)
			require.NoError(t, err)
			require.NoError(t, cluster.WaitForBlock(currentBlock.Header.Number+1, time.Second*5))
		}

		validatorSrv := cluster.Servers[1]
		validatorAcc, err := validatorHelper.GetAccountFromDir(validatorSrv.DataDir())
		require.NoError(t, err)

		validatorRawKey, err := validatorAcc.Ecdsa.MarshallPrivateKey()
		require.NoError(t, err)

		err = cluster.Bridges[bridgeOne].Withdraw(
			common.ERC20,
			hex.EncodeToString(validatorRawKey),
			validatorAcc.Address().String(),
			tokensToTransfer.String(),
			"",
			validatorSrv.JSONRPCAddr(),
			bridgeCfg.InternalERC20PredicateAddr,
			contracts.NativeERC20TokenContract,
			false)
		require.NoError(t, err)

		// Wait for 1 block before getting expected child balance
		waitOneBlockFn(validatorSrv)

		validatorBalanceAfterWithdraw, err := childEthEndpoint.GetBalance(
			validatorAcc.Address(), jsonrpc.LatestBlockNumberOrHash)
		require.NoError(t, err)

		err = cluster.Bridges[bridgeOne].Withdraw(
			common.ERC20,
			hex.EncodeToString(nonValidatorKeyRaw),
			nonValidatorKey.Address().String(),
			tokensToTransfer.String(),
			"",
			validatorSrv.JSONRPCAddr(),
			bridgeCfg.InternalERC20PredicateAddr,
			contracts.NativeERC20TokenContract,
			false)
		require.NoError(t, err)

		// Wait for 1 block before getting expected child balance
		waitOneBlockFn(validatorSrv)

		nonValidatorBalanceAfterWithdraw, err := childEthEndpoint.GetBalance(
			nonValidatorKey.Address(), jsonrpc.LatestBlockNumberOrHash)
		require.NoError(t, err)

		currentBlock, err := childEthEndpoint.GetBlockByNumber(jsonrpc.LatestBlockNumber, false)
		require.NoError(t, err)

		currentExtra, err := polytypes.GetIbftExtra(currentBlock.Header.ExtraData)
		require.NoError(t, err)

		t.Logf("Latest block number: %d, epoch number: %d\n", currentBlock.Number(), currentExtra.BlockMetaData.EpochNumber)

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			for bridgeEventID := uint64(1); bridgeEventID <= bridgeEvents; bridgeEventID++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, bridgeEventID, false) {
					return false
				}
			}

			return true
		}))

		// assert that receiver's balances on RootERC20 smart contract are expected
		checkBalancesFn(validatorAcc.Address(), tokensToTransfer, validatorBalanceAfterWithdraw, true)
		checkBalancesFn(nonValidatorKey.Address(), tokensToTransfer, nonValidatorBalanceAfterWithdraw, false)
	})

	t.Run("do a deposit to some validator and non-validator address", func(t *testing.T) {
		validatorSrv := cluster.Servers[4]
		validatorAcc, err := validatorHelper.GetAccountFromDir(validatorSrv.DataDir())
		require.NoError(t, err)

		require.NoError(t, cluster.Bridges[bridgeOne].Deposit(
			common.ERC20,
			bridgeCfg.ExternalNativeERC20Addr,
			bridgeCfg.ExternalERC20PredicateAddr,
			bridgeHelper.TestAccountPrivKey,
			strings.Join([]string{validatorAcc.Address().String(), nonValidatorKey.Address().String()}, ","),
			strings.Join([]string{tokensToTransfer.String(), tokensToTransfer.String()}, ","),
			"",
			cluster.Bridges[bridgeOne].JSONRPCAddr(),
			bridgeHelper.TestAccountPrivKey,
			false),
		)

		currentBlock, err := childEthEndpoint.GetBlockByNumber(jsonrpc.LatestBlockNumber, false)
		require.NoError(t, err)

		// wait for couple of epoches
		finalBlockNum := currentBlock.Header.Number + 2*epochSize
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

		// the transaction is processed and there should be a success event
		var bridgeMessageResult contractsapi.BridgeMessageResultEvent

		for i := uint64(0); i < numberOfAttempts; i++ {
			logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, finalBlockNum+i*epochSize, childEthEndpoint)
			require.NoError(t, err)

			if len(logs) == stateSyncedLogsCount || i == numberOfAttempts-1 {
				// assert that all deposits are executed successfully
				assertBridgeEventResultSuccess(t, logs, stateSyncedLogsCount)

				break
			}

			require.NoError(t, cluster.WaitForBlock(finalBlockNum+(i+1)*epochSize, time.Minute))
		}
	})

	t.Run("transfer more native tokens than 0x0 balance is", func(t *testing.T) {
		// since bridging native token is essentially minting
		// (i.e. transferring tokens from 0x0 to receiver address using native transfer precompile),
		// this test tries to deposit more tokens than 0x0 address has on its balance
		currentBlock, err := childEthEndpoint.BlockNumber()
		require.NoError(t, err)

		require.NoError(t, cluster.WaitForBlock(currentBlock+epochSize, 2*time.Minute))
		currentBlock += epochSize

		require.NoError(t, cluster.Bridges[bridgeOne].Deposit(
			common.ERC20,
			bridgeCfg.ExternalNativeERC20Addr,
			bridgeCfg.ExternalERC20PredicateAddr,
			bridgeHelper.TestAccountPrivKey,
			strings.Join([]string{nonValidatorKey.Address().String()}, ","),
			strings.Join([]string{tenMilionTokens.String()}, ","),
			"",
			cluster.Bridges[bridgeOne].JSONRPCAddr(),
			bridgeHelper.TestAccountPrivKey,
			false),
		)

		// wait for couple of epoches
		finalBlockNum := currentBlock + 2*epochSize
		require.NoError(t, cluster.WaitForBlock(finalBlockNum, 2*time.Minute))

		// the transaction is processed and there should be a success event
		var bridgeMessageResult contractsapi.BridgeMessageResultEvent

		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), currentBlock, finalBlockNum, childEthEndpoint)
		require.NoError(t, err)

		assertBridgeEventResultNotSuccessful(t, logs, 1)

		require.NoError(t, cluster.WaitForBlock(finalBlockNum+epochSize, time.Minute))

		require.NoError(t, cluster.WaitUntil(time.Minute*3, time.Second*2, func() bool {
			if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, 3, true) {
				return false
			}

			return true
		}))
	})
}

// TestE2E_Bridge_L1OriginatedNativeToken_ERC20StakingToken tests the end-to-end functionality of the bridge
// between an L1 originated native token and an ERC20 staking token. The test simulates a staking scenario
// with two validators, where a specified stake amount is initially added, then another stake is added and
// validated. Finally, the additional stake is unstaked, and the expected outcome is verified. This test
// includes interaction with a test cluster, relayer, minting operations, and validation checks.
//
// It performs the following actions:
// - Initializes a test cluster with multiple validators, setting up the configuration and staking parameters.
// - Mints an ERC20 token and stakes it to the second validator's account.
// - Adds a stake to the second validator, and waits for the epoch blocks to confirm.
// - Verifies the correct amount of stake after the addition.
// - Unstakes the previously added stake and checks if the validator's stake is restored to its initial value.
func TestE2E_Bridge_L1OriginatedNativeToken_ERC20StakingToken(t *testing.T) {
	const (
		epochSize       = 5
		blockTimeout    = 30 * time.Second
		numberOfBridges = 1
	)

	var (
		initialStake   = ethgo.Ether(10)
		addedStake     = ethgo.Ether(1)
		stakeTokenAddr = types.StringToAddress("0x2040")
	)

	minter, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithNumBlockConfirmations(0),
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBladeAdmin(minter.Address().String()),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				tcc.StakeAmounts = append(tcc.StakeAmounts, initialStake)
			}
		}),
		framework.WithNativeTokenConfig(nativeTokenNonMintableConfig),
		framework.WithPredeploy(fmt.Sprintf("%s:RootERC20", stakeTokenAddr)),
	)
	defer cluster.Stop()

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	// first validator server(minter)
	firstValidator := cluster.Servers[0]
	// second validator server
	secondValidator := cluster.Servers[1]

	// validator account from second validator
	validatorAccTwo, err := validatorHelper.GetAccountFromDir(secondValidator.DataDir())
	require.NoError(t, err)

	relayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(firstValidator.JSONRPCAddr()))
	require.NoError(t, err)

	mintFn := &contractsapi.MintRootERC20Fn{
		To:     validatorAccTwo.Address(),
		Amount: addedStake,
	}

	mintInput, err := mintFn.EncodeAbi()
	require.NoError(t, err)

	nonNativeErc20 := polybftCfg.StakeTokenAddr

	mintTx := types.NewTx(types.NewDynamicFeeTx(
		types.WithTo(&nonNativeErc20),
		types.WithInput(mintInput),
	))

	receipt, err := relayer.SendTransaction(mintTx, minter)
	require.NoError(t, err)
	require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

	secondValidatorInfo, err := validatorHelper.GetValidatorInfo(validatorAccTwo.Ecdsa.Address(), relayer)
	require.NoError(t, err)
	require.True(t, secondValidatorInfo.Stake.Cmp(initialStake) == 0)

	require.NoError(t, cluster.WaitForBlock(epochSize*1, blockTimeout))

	require.NoError(t, secondValidator.Stake(polybftCfg.StakeTokenAddr, addedStake))

	require.NoError(t, cluster.WaitForBlock(epochSize*3, blockTimeout))

	secondValidatorInfo, err = validatorHelper.GetValidatorInfo(validatorAccTwo.Ecdsa.Address(), relayer)
	require.NoError(t, err)

	expectedStakeAmount := new(big.Int).Add(initialStake, addedStake)
	require.Equal(t, expectedStakeAmount, secondValidatorInfo.Stake)

	require.NoError(t, secondValidator.Unstake(addedStake))

	require.NoError(t, cluster.WaitForBlock(epochSize*4, blockTimeout))

	secondValidatorInfo, err = validatorHelper.GetValidatorInfo(validatorAccTwo.Ecdsa.Address(), relayer)
	require.NoError(t, err)
	require.True(t, secondValidatorInfo.Stake.Cmp(initialStake) == 0)
}

// TestE2E_Bridge_ValidatorSetChange tests the end-to-end functionality of the bridge
// during a validator set change. It performs the following steps:
//   - Sets up a test cluster with a specified epoch size, sprint size, and number of bridges.
//   - Configures the cluster with a relayer private key and premines accounts for withdrawals.
//   - Retrieves the initial validator set hash from the bridge storage, internal gateway,
//     and external gateway, ensuring they are consistent.
//   - Unstakes a validator and waits for the changes to propagate through the network.
//   - Retrieves the updated validator set hash from the bridge storage, internal gateway,
//     and external gateway, ensuring they are consistent and different from the initial hash.
//   - Verifies that the validator set hash changes are correctly applied across the bridge
//     components after the validator is unstaked.
func TestE2E_Bridge_ValidatorSetChange(t *testing.T) {
	const (
		epochSize       = 10
		sprintSize      = uint64(5)
		numberOfBridges = 1
	)

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 6,
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				// premine receivers, so that they are able to do withdrawals
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))

			tcc.Premine = append(tcc.Premine, relayerPrivateKey.String())
		}))

	defer cluster.Stop()

	cluster.WaitForReady(t)

	validatorSrv := cluster.Servers[1]

	validatorEndpoint := validatorSrv.JSONRPC()

	polycfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	internalTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(validatorEndpoint))
	require.NoError(t, err)

	externalTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalTxRelayer.Client().ChainID()
	require.NoError(t, err)

	getBridgeStorageValidatorSetHash := func() string {
		method := contractsapi.BridgeStorage.Abi.GetMethod("currentValidatorSetHash")
		res, err := internalTxRelayer.Call(types.ZeroAddress, contracts.BridgeStorageContract, method.ID())
		require.NoError(t, err)

		return res
	}

	getInternalGatewayValidatorSetHash := func() string {
		method := contractsapi.BridgeStorage.Abi.GetMethod("currentValidatorSetHash")
		internalGatewayAddress := polycfg.Bridge[chainID.Uint64()].InternalGatewayAddr

		res, err := internalTxRelayer.Call(types.ZeroAddress, internalGatewayAddress, method.ID())
		require.NoError(t, err)

		return res
	}

	getExternalGatewayValidatorSetHash := func() string {
		method := contractsapi.BridgeStorage.Abi.GetMethod("currentValidatorSetHash")
		externalGatewayAddress := polycfg.Bridge[chainID.Uint64()].ExternalGatewayAddr

		res, err := externalTxRelayer.Call(types.ZeroAddress, externalGatewayAddress, method.ID())
		require.NoError(t, err)

		return res
	}

	// validator set hash before unstake
	beforeBridgeStorageValidatorSetHash := getBridgeStorageValidatorSetHash()
	beforeInternalGatewayValidatorSetHash := getInternalGatewayValidatorSetHash()
	beforeExternalGatewayValidatorSetHash := getExternalGatewayValidatorSetHash()

	require.Equal(t, beforeBridgeStorageValidatorSetHash, beforeInternalGatewayValidatorSetHash)
	require.Equal(t, beforeBridgeStorageValidatorSetHash, beforeExternalGatewayValidatorSetHash)

	srv := cluster.Servers[0]
	validatorAcc, err := validatorHelper.GetAccountFromDir(srv.DataDir())
	require.NoError(t, err)

	validatorAddr := validatorAcc.Ecdsa.Address()

	validatorInfo, err := validatorHelper.GetValidatorInfo(validatorAcc.Address(), internalTxRelayer)
	require.NoError(t, err)
	require.True(t, validatorInfo.IsActive)

	initialStake := validatorInfo.Stake

	// unstake validator
	require.NoError(t, srv.Unstake(initialStake))

	currentBlock, err := validatorEndpoint.BlockNumber()
	require.NoError(t, err)

	// waiting for unstake and relayer to apply
	require.NoError(t, cluster.WaitForBlock(currentBlock+2*epochSize, time.Minute))

	// validator set hash after unstake
	afterBridgeStorageValidatorSetHash := getBridgeStorageValidatorSetHash()
	afterInternalGatewayValidatorSetHash := getInternalGatewayValidatorSetHash()
	afterExternalGatewayValidatorSetHash := getExternalGatewayValidatorSetHash()

	t.Logf("Validator unstaked %s\n", validatorAddr.String())

	t.Logf("beforeBridgeStorageValidatorSetHash=%s\n", beforeBridgeStorageValidatorSetHash)
	t.Logf("afterBridgeStorageValidatorSetHash=%s\n", afterBridgeStorageValidatorSetHash)

	t.Logf("beforeInternalGatewayValidatorSetHash=%s\n", beforeInternalGatewayValidatorSetHash)
	t.Logf("afterInternalGatewayValidatorSetHash=%s\n", afterInternalGatewayValidatorSetHash)

	t.Logf("beforeExternalGatewayValidatorSetHash=%s\n", beforeExternalGatewayValidatorSetHash)
	t.Logf("afterExternalGatewayValidatorSetHash=%s\n", afterExternalGatewayValidatorSetHash)

	require.NotEqual(t, beforeBridgeStorageValidatorSetHash, afterBridgeStorageValidatorSetHash)
	require.NotEqual(t, beforeInternalGatewayValidatorSetHash, afterInternalGatewayValidatorSetHash)
	require.NotEqual(t, beforeExternalGatewayValidatorSetHash, afterExternalGatewayValidatorSetHash)
	require.Equal(t, afterBridgeStorageValidatorSetHash, afterInternalGatewayValidatorSetHash)
	require.Equal(t, afterBridgeStorageValidatorSetHash, afterExternalGatewayValidatorSetHash)
}

// Create X ERC20/ERC721/ERC1155 bridge transactions between source and destination chain, where the following is done sequentially:
//   - X/4 bridge transaction execute and settle on the destination chain while
//   - another X/4 bridge transactions fail due to insufficient funds on source chain.
//   - X/4 funding transactions are made on source chain.
//   - the same X/4 bridge transactions are resubmitted, being executed and settled on the destination chain.
//
// Ensure the bridge correctly handles the failed transactions.
// Use X/4 different source addresses.
// The test should be parametrized. X=60.
// The test should be run for Blade as source chain and an external chain as destination chain and vice verse in parallel.
func TestE2E_Bridge_InsufficientFunds(t *testing.T) {
	const (
		// X = 60
		transfersCount  = 5                                   // decreased from 15 for CI
		erc20FirstID    = uint64(1)                           // start from one
		erc20SecondID   = erc20FirstID + transfersCount + 1   // last id + num of transfers + first event for contract
		erc721FirstID   = erc20SecondID + transfersCount      // last id + num of transfers
		erc721SecondID  = erc721FirstID + transfersCount + 1  // last id + num of transfers + first event for contract
		erc1155FirstID  = erc721SecondID + transfersCount     // last id + num of transfers
		erc1155SecondID = erc1155FirstID + transfersCount + 1 // last id + num of transfers + first event for contract
		epochSize       = 10
		sprintSize      = uint64(5)
		numberOfBridges = 1

		amount    = int64(1000)
		amountStr = "1000"
	)

	accountAddrs := make([]types.Address, transfersCount)
	accounts := make([]string, transfersCount)
	accountKeys := make([]string, transfersCount)
	amounts := make([]string, transfersCount)

	for i := 0; i < transfersCount; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		rawKey, err := key.MarshallPrivateKey()
		require.NoError(t, err)

		accountKeys[i] = hex.EncodeToString(rawKey)
		accountAddrs[i] = key.Address()
		accounts[i] = key.Address().String()
		amounts[i] = fmt.Sprintf("%d", amount)

		t.Logf("Receiver#%d=%s\n", i+1, accounts[i])
	}

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))

			tcc.Premine = append(tcc.Premine, accounts...)
			tcc.Premine = append(tcc.Premine, relayerPrivateKey.String())
		}))

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

	var (
		internalERC20Addr types.Address
		externalERC20Addr types.Address
	)

	t.Run("ERC20 token transfer test", func(t *testing.T) {
		// Deployment of ERC20 token on both chains
		{
			// internal ERC20 token
			deployTx := types.NewTx(types.NewLegacyTx(
				types.WithTo(nil),
				types.WithInput(contractsapi.RootERC20.Bytecode),
			))

			receipt, err := internalChainTxRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			internalERC20Addr = types.Address(receipt.ContractAddress)

			// external ERC20 token
			receipt, err = externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			externalERC20Addr = types.Address(receipt.ContractAddress)
		}

		t.Logf("Internal ERC20 token address: %s\n", internalERC20Addr)
		t.Logf("External ERC20 token address: %s\n", externalERC20Addr)

		// mint ERC20 tokens
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
		stopChan := time.After(10 * time.Minute)

		runTest := func(erc20Addr, predicateAddr, destinationGW types.Address,
			rpcAddr, dtype string, sourceRelayer, destinationRelayer txrelayer.TxRelayer) {
			// mint
			for i := range transfersCount {
				if err := mint(erc20Addr, big.NewInt(amount),
					accountAddrs[i], sourceRelayer); err != nil {
					errChan <- err

					return
				}
			}

			t.Logf("ERC20 tokens minted " + dtype)

			// transfer function
			transferFunc := func(shouldThrowError bool) error {
				for i := range transfersCount {
					err := bridge.Deposit(
						common.ERC20,
						erc20Addr,
						predicateAddr,
						accountKeys[i],
						accounts[i],
						amountStr,
						"",
						rpcAddr,
						"",
						false,
					)

					if shouldThrowError && err == nil {
						return fmt.Errorf("expected error but got nil")
					}

					if !shouldThrowError && err != nil {
						return err
					}
				}

				return nil
			}

			// first transfer - should be processed
			if err := transferFunc(false); err != nil {
				errChan <- err
			}

			t.Logf("ERC20 tokens deposited " + dtype)

			// should be processed because there are enough ERC20 funds
			if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
				for i := erc20FirstID; i < erc20FirstID+transfersCount+1; i++ {
					if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
						return false
					}
				}

				return true
			}); err != nil {
				errChan <- err

				return
			}

			t.Logf("All events processed " + dtype)

			// second transfer - should fail because there are not enough ERC20 funds
			if err := transferFunc(true); err != nil {
				errChan <- err

				return
			}

			t.Logf("Second transfer failed " + dtype)

			// mint again to have enough funds
			for i := range transfersCount {
				if err := mint(erc20Addr, big.NewInt(amount),
					accountAddrs[i], sourceRelayer); err != nil {
					errChan <- err

					return
				}
			}

			t.Logf("ERC20 tokens minted again " + dtype)

			// should be processed because there are enough ERC20 funds
			if err := transferFunc(false); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC20 tokens deposited again " + dtype)

			// should be processed because there are enough ERC20 funds
			if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
				for i := erc20SecondID; i < erc20SecondID+transfersCount; i++ {
					if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
						return false
					}
				}

				return true
			}); err != nil {
				errChan <- err

				return
			}

			t.Logf("All events processed " + dtype)

			childTokenAddr := getChildToken(t, contractsapi.RootERC20Predicate.Abi, predicateAddr, erc20Addr, sourceRelayer)

			for i := range transfersCount {
				balance := erc20BalanceOf(t, accountAddrs[i], childTokenAddr, destinationRelayer)
				if balance.Cmp(big.NewInt(amount*2)) != 0 {
					errChan <- fmt.Errorf("balance check failed")

					return
				}
			}

			t.Logf("ERC20 tokens balances checked " + dtype)
			errChan <- nil
		}

		go runTest(internalERC20Addr, bridgeCfg.InternalMintableERC20PredicateAddr, bridgeCfg.ExternalGatewayAddr,
			internalJSONRPCAddr, "I2E", internalChainTxRelayer, externalChainTxRelayer)

		go runTest(externalERC20Addr, bridgeCfg.ExternalERC20PredicateAddr, bridgeCfg.InternalGatewayAddr,
			externalJSONRPCAddr, "E2I", externalChainTxRelayer, internalChainTxRelayer)

		counter := 0

		for {
			select {
			case err := <-errChan:
				if err != nil {
					t.Fatal(err)

					return
				}

				if counter++; counter == 2 {
					t.Logf("Test passed")

					return
				}
			case <-stopChan:
				t.Fatal("timeout")

				return
			}
		}
	})

	t.Run("ERC721 token transfer test", func(t *testing.T) {
		var (
			internalERC721Addr types.Address
			externalERC721Addr types.Address
		)

		// Deployment of ERC721 token on both chains
		{
			deployTx := types.NewTx(types.NewLegacyTx(
				types.WithTo(nil),
				types.WithInput(contractsapi.RootERC721.Bytecode),
			))

			receipt, err := internalChainTxRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			internalERC721Addr = types.Address(receipt.ContractAddress)

			receipt, err = externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			externalERC721Addr = types.Address(receipt.ContractAddress)
		}

		t.Logf("Internal ERC721 token address: %s\n", internalERC721Addr)
		t.Logf("External ERC721 token address: %s\n", externalERC721Addr)

		mint := func(token types.Address, account types.Address, relayer txrelayer.TxRelayer) error {
			mintFn := &contractsapi.MintRootERC721Fn{
				To: account,
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
		stopChan := time.After(10 * time.Minute)

		runTest := func(erc721Addr, predicateAddr, destinationGW types.Address, rpcAddr, dtype string,
			sourceRelayer, destinationRelayer txrelayer.TxRelayer) {
			// mint
			for i := range transfersCount {
				if err := mint(erc721Addr, accountAddrs[i], sourceRelayer); err != nil {
					errChan <- err

					return
				}
			}

			t.Logf("ERC721 tokens minted " + dtype)

			transferFunc := func(startID int, shouldThrowError bool) error {
				for i := range transfersCount {
					err := bridge.Deposit(
						common.ERC721,
						erc721Addr,
						predicateAddr,
						accountKeys[i],
						accounts[i],
						"",
						fmt.Sprintf("%d", startID+i),
						rpcAddr,
						"",
						false,
					)

					if shouldThrowError && err == nil {
						return fmt.Errorf("expected error didn't show")
					}

					if !shouldThrowError && err != nil {
						return err
					}
				}

				return nil
			}

			// transfer with minting, should succeed
			if err := transferFunc(0, false); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC721 tokens all deposited " + dtype)
			// check if all events are processed on destination
			if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
				for i := erc721FirstID; i < erc721FirstID+transfersCount+1; i++ {
					if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
						return false
					}
				}

				return true
			}); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC721 tokens all events processed " + dtype)

			childToken := getChildToken(t, contractsapi.RootERC721Predicate.Abi, predicateAddr, erc721Addr, sourceRelayer)

			for i := range transfersCount {
				owner := erc721OwnerOf(t, big.NewInt(int64(i)), childToken, destinationRelayer)
				if owner != accountAddrs[i] {
					errChan <- fmt.Errorf("owner of %d is not same on source & destination", i)

					return
				}
			}

			t.Logf("ERC721 tokens all ownership checked " + dtype)

			// transfer without minting, should return error
			if err := transferFunc(transfersCount, true); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC721 tokens all deposit failed as expected " + dtype)

			// mint
			for i := range transfersCount {
				if err := mint(erc721Addr, accountAddrs[i], sourceRelayer); err != nil {
					errChan <- err

					return
				}
			}

			t.Logf("ERC721 tokens all minted " + dtype)

			// deposit new tokens again, should succeed
			if err := transferFunc(transfersCount, false); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC721 tokens all deposited again " + dtype)

			// check if all events are processed on destination
			if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
				for i := erc721SecondID; i < erc721SecondID+transfersCount; i++ {
					if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
						return false
					}
				}

				return true
			}); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC721 tokens all events processed " + dtype)

			for i := range transfersCount {
				owner := erc721OwnerOf(t, big.NewInt(int64(i+transfersCount)), childToken, destinationRelayer)
				if owner != accountAddrs[i] {
					errChan <- fmt.Errorf("owner of %d is not same on source & destination", i+transfersCount)

					return
				}
			}

			t.Logf("ERC721 tokens all ownership checked " + dtype)

			errChan <- nil
		}

		go runTest(internalERC721Addr, bridgeCfg.InternalMintableERC721PredicateAddr, bridgeCfg.ExternalGatewayAddr,
			internalJSONRPCAddr, "I2E", internalChainTxRelayer, externalChainTxRelayer)

		go runTest(externalERC721Addr, bridgeCfg.ExternalERC721PredicateAddr, bridgeCfg.InternalGatewayAddr,
			externalJSONRPCAddr, "E2I", externalChainTxRelayer, internalChainTxRelayer)

		counter := 0

		for {
			select {
			case err := <-errChan:
				if err != nil {
					t.Fatalf(err.Error())

					return
				}

				if counter++; counter == 2 {
					t.Logf("Test succeed")

					return
				}
			case <-stopChan:
				t.Fatalf("timeout")

				return
			}
		}
	})

	t.Run("ERC1155 token transfer test", func(t *testing.T) {
		var (
			internalERC1155Addr types.Address
			externalERC1155Addr types.Address
		)

		// Deployment of ERC1155 token on both chains
		{
			deployTx := types.NewTx(types.NewLegacyTx(
				types.WithTo(nil),
				types.WithInput(contractsapi.RootERC1155.Bytecode),
			))

			receipt, err := internalChainTxRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			internalERC1155Addr = types.Address(receipt.ContractAddress)

			receipt, err = externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			externalERC1155Addr = types.Address(receipt.ContractAddress)
		}

		t.Logf("Internal ERC1155 token address: %s\n", internalERC1155Addr)
		t.Logf("External ERC1155 token address: %s\n", externalERC1155Addr)

		mint := func(token types.Address, account types.Address, id *big.Int, relayer txrelayer.TxRelayer) error {
			mintFn := &contractsapi.MintBatchRootERC1155Fn{
				To:      account,
				IDs:     []*big.Int{id},
				Amounts: []*big.Int{big.NewInt(amount)},
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
		stopChan := time.After(10 * time.Minute)

		runTest := func(ERC1155Addr, predicateAddr, destinationGW types.Address, rpcAddr, dtype string,
			sourceRelayer, destinationRelayer txrelayer.TxRelayer) {
			// mint
			for i := range transfersCount {
				if err := mint(ERC1155Addr, accountAddrs[i],
					big.NewInt(int64(i)), sourceRelayer); err != nil {
					errChan <- err

					return
				}
			}

			t.Logf("ERC1155 tokens minted " + dtype)

			transferFunc := func(startID int, shouldThrowError bool) error {
				for i := range transfersCount {
					err := bridge.Deposit(
						common.ERC1155,
						ERC1155Addr,
						predicateAddr,
						accountKeys[i],
						accounts[i],
						amountStr,
						fmt.Sprintf("%d", startID+i),
						rpcAddr,
						"",
						false,
					)

					if shouldThrowError && err == nil {
						return fmt.Errorf("expected error didn't show")
					}

					if !shouldThrowError && err != nil {
						return err
					}
				}

				return nil
			}

			// transfer with minting, should succeed
			if err := transferFunc(0, false); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC1155 tokens all deposited " + dtype)
			// check if all events are processed on destination
			if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
				for i := erc1155FirstID; i < erc1155FirstID+transfersCount+1; i++ {
					if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
						return false
					}
				}

				return true
			}); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC1155 tokens all events processed " + dtype)

			childToken := getChildToken(t, contractsapi.RootERC1155Predicate.Abi, predicateAddr, ERC1155Addr, sourceRelayer)

			checkERC1155Balance := func(account types.Address, id int64) error {
				balanceOfFn := &contractsapi.BalanceOfChildERC1155Fn{
					Account: account,
					ID:      big.NewInt(id),
				}

				balanceInput, err := balanceOfFn.EncodeAbi()
				if err != nil {
					return err
				}

				balanceRaw, err := destinationRelayer.Call(types.ZeroAddress, childToken, balanceInput)
				if err != nil {
					return err
				}

				balance, err := helperCommon.ParseUint256orHex(&balanceRaw)
				if err != nil {
					return err
				}

				validBalance := big.NewInt(amount)

				if validBalance.Cmp(balance) != 0 {
					return fmt.Errorf("balances incompatible %d:%d", balance.Int64(), validBalance.Int64())
				}

				return nil
			}

			for i := range transfersCount {
				if err := checkERC1155Balance(accountAddrs[i], int64(i)); err != nil {
					errChan <- err

					return
				}
			}

			t.Logf("ERC1155 tokens all ownership checked " + dtype)

			// transfer without minting, should return error
			if err := transferFunc(transfersCount, true); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC1155 tokens all deposit failed as expected " + dtype)

			// mint
			for i := range transfersCount {
				if err := mint(ERC1155Addr, accountAddrs[i],
					big.NewInt(int64(i+transfersCount)), sourceRelayer); err != nil {
					errChan <- err

					return
				}
			}

			t.Logf("ERC1155 tokens all minted " + dtype)

			// deposit new tokens again, should succeed
			if err := transferFunc(transfersCount, false); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC1155 tokens all deposited again " + dtype)

			// check if all events are processed on destination
			if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
				for i := erc1155SecondID; i < erc1155SecondID+transfersCount; i++ {
					if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
						return false
					}
				}

				return true
			}); err != nil {
				errChan <- err

				return
			}

			t.Logf("ERC1155 tokens all events processed " + dtype)

			for i := range transfersCount {
				if err := checkERC1155Balance(accountAddrs[i], int64(i+transfersCount)); err != nil {
					errChan <- err

					return
				}
			}

			t.Logf("ERC1155 tokens all ownership checked " + dtype)

			errChan <- nil
		}

		go runTest(internalERC1155Addr, bridgeCfg.InternalMintableERC1155PredicateAddr, bridgeCfg.ExternalGatewayAddr,
			internalJSONRPCAddr, "I2E", internalChainTxRelayer, externalChainTxRelayer)

		go runTest(externalERC1155Addr, bridgeCfg.ExternalERC1155PredicateAddr, bridgeCfg.InternalGatewayAddr,
			externalJSONRPCAddr, "E2I", externalChainTxRelayer, internalChainTxRelayer)

		counter := 0

		for {
			select {
			case err := <-errChan:
				if err != nil {
					t.Fatalf(err.Error())

					return
				}

				if counter++; counter == 2 {
					t.Logf("Test succeed")

					return
				}
			case <-stopChan:
				t.Fatalf("timeout")

				return
			}
		}
	})
}

func TestE2E_Bridge_WithdrawInsufficientFunds(t *testing.T) {
	const (
		// X = 60
		transfersCount  = 5                               // decreased from 15 for CI
		erc20ID1        = uint64(1)                       // start from one
		erc20ID2        = erc20ID1 + transfersCount + 1   // last id + num of transfers + first event for contract
		erc20ID3        = erc20ID2 + transfersCount       // last id + num of transfers
		erc20ID4        = erc20ID3 + transfersCount       // last id + num of transfers
		erc721ID1       = erc20ID4 + transfersCount       // last id + num of transfers
		erc721ID2       = erc721ID1 + transfersCount + 1  // last id + num of transfers + first event for contract
		erc721ID3       = erc721ID2 + transfersCount      // last id + num of transfers
		erc721ID4       = erc721ID3 + transfersCount      // last id + num of transfers
		erc1155ID1      = erc721ID4 + transfersCount      // last id + num of transfers
		erc1155ID2      = erc1155ID1 + transfersCount + 1 // last id + num of transfers + first event for contract
		erc1155ID3      = erc1155ID2 + transfersCount     // last id + num of transfers
		erc1155ID4      = erc1155ID3 + transfersCount     // last id + num of transfers
		epochSize       = 10
		sprintSize      = uint64(5)
		numberOfBridges = 1

		amount    = int64(1000)
		amountStr = "1000"
	)

	var (
		accountAddrs = make([]types.Address, transfersCount)
		accounts     = make([]string, transfersCount)
		accountKeys  = make([]string, transfersCount)
		amounts      = make([]string, transfersCount)
	)

	for i := 0; i < transfersCount; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		rawKey, err := key.MarshallPrivateKey()
		require.NoError(t, err)

		accountKeys[i] = hex.EncodeToString(rawKey)
		accountAddrs[i] = key.Address()
		accounts[i] = key.Address().String()
		amounts[i] = fmt.Sprintf("%d", amount)

		t.Logf("Receiver#%d=%s\n", i+1, accounts[i])
	}

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))

			tcc.Premine = append(tcc.Premine, accounts...)
			tcc.Premine = append(tcc.Premine, relayerPrivateKey.String())
		}))

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

	errChan := make(chan error, 1)

	t.Run("ERC20", func(t *testing.T) {
		runTest := func(sourcePred, destinationPred, sourceGW, destinationGW types.Address,
			sourceRPC, destinationRPC, dtype string, sourceRelayer, destinationRelayer txrelayer.TxRelayer) {
			// deploy ERC20 token
			deployTx := types.NewTx(types.NewLegacyTx(
				types.WithTo(nil),
				types.WithInput(contractsapi.RootERC20.Bytecode),
			))

			receipt, err := sourceRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			tokenAddr := types.Address(receipt.ContractAddress)

			depositFunc := func(startID, contract uint64) error {
				// mint & deposit
				for i := range transfersCount {
					if err := bridge.Deposit(
						common.ERC20,
						tokenAddr,
						sourcePred,
						bridgeHelper.TestAccountPrivKey,
						accounts[i],
						amountStr,
						"",
						sourceRPC,
						bridgeHelper.TestAccountPrivKey,
						false,
					); err != nil {
						return err
					}
				}

				t.Logf("ERC20 tokens deposited %s", dtype)

				// should be processed because there are enough ERC20 funds
				if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
					for i := startID; i < startID+transfersCount+contract; i++ {
						if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
							return false
						}
					}

					return true
				}); err != nil {
					return err
				}

				t.Logf("All deposit events processed %s", dtype)

				return nil
			}

			if err := depositFunc(erc20ID1, 1); err != nil {
				errChan <- err

				return
			}

			childToken := getChildToken(t, contractsapi.RootERC20Predicate.Abi, sourcePred, tokenAddr, sourceRelayer)

			withdrawFunc := func(shouldThrowError bool, startID uint64) error {
				// withdraw - should be processed because there are enough ERC20 funds
				for i, accountKey := range accountKeys {
					err = bridge.Withdraw(
						common.ERC20,
						accountKey,
						accounts[i],
						amountStr,
						"",
						destinationRPC,
						destinationPred,
						childToken,
						false)

					if shouldThrowError && err == nil {
						return fmt.Errorf("expected error but got nil %s", dtype)
					}

					if !shouldThrowError && err != nil {
						return err
					}
				}

				if !shouldThrowError {
					// should be processed because there are enough ERC20 funds
					if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
						for i := startID; i < startID+transfersCount; i++ {
							if !isEventProcessed(t, sourceGW, sourceRelayer, i, false) {
								return false
							}
						}

						return true
					}); err != nil {
						return err
					}

					t.Logf("All withdraws processed %s", dtype)

					return nil
				}

				t.Logf("Withdraw failed as expected %s", dtype)

				return nil
			}

			// successful withdraw
			if err := withdrawFunc(false, erc20ID2); err != nil {
				errChan <- err

				return
			}

			// no funds withdraw
			if err := withdrawFunc(true, 0); err != nil {
				errChan <- err

				return
			}

			// deposit again
			if err := depositFunc(erc20ID3, 0); err != nil {
				errChan <- err

				return
			}

			// successful withdraw
			if err := withdrawFunc(false, erc20ID4); err != nil {
				errChan <- err

				return
			}

			for _, acc := range accountAddrs {
				balance := erc20BalanceOf(t, acc, childToken, destinationRelayer)
				if balance.Cmp(big.NewInt(0)) != 0 {
					errChan <- fmt.Errorf("balance check failed %s for account %s", dtype, acc)

					return
				}
			}

			errChan <- nil
		}

		go runTest(bridgeCfg.InternalMintableERC20PredicateAddr, bridgeCfg.ExternalMintableERC20PredicateAddr,
			bridgeCfg.InternalGatewayAddr, bridgeCfg.ExternalGatewayAddr, internalJSONRPCAddr, externalJSONRPCAddr, "I2E", internalChainTxRelayer, externalChainTxRelayer)

		go runTest(bridgeCfg.ExternalERC20PredicateAddr, bridgeCfg.InternalERC20PredicateAddr,
			bridgeCfg.ExternalGatewayAddr, bridgeCfg.InternalGatewayAddr, externalJSONRPCAddr, internalJSONRPCAddr, "E2I", externalChainTxRelayer, internalChainTxRelayer)

		counter := 0

		for {
			select {
			case err := <-errChan:
				if err != nil {
					t.Fatal(err)

					return
				}

				if counter++; counter == 2 {
					return
				}
			case <-time.After(10 * time.Minute):
				t.Fatalf("timeout")
			}
		}
	})

	t.Run("ERC721", func(t *testing.T) {
		runTest := func(sourcePred, destinationPred, sourceGW, destinationGW types.Address,
			sourceRPC, destinationRPC, dtype string, sourceRelayer, destinationRelayer txrelayer.TxRelayer) {
			// deploy ERC721 token
			deployTx := types.NewTx(types.NewLegacyTx(
				types.WithTo(nil),
				types.WithInput(contractsapi.RootERC721.Bytecode),
			))

			receipt, err := sourceRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			tokenAddr := types.Address(receipt.ContractAddress)

			depositFunc := func(startID, contract, tokenStart uint64) error {
				// mint & deposit
				for i := range transfersCount {
					if err := bridge.Deposit(
						common.ERC721,
						tokenAddr,
						sourcePred,
						bridgeHelper.TestAccountPrivKey,
						accounts[i],
						"",
						fmt.Sprintf("%d", int(tokenStart)+i),
						sourceRPC,
						bridgeHelper.TestAccountPrivKey,
						false,
					); err != nil {
						return err
					}
				}

				t.Logf("ERC721 tokens deposited %s", dtype)

				// should be processed because there are enough ERC721 funds
				if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
					for i := startID; i < startID+transfersCount+contract; i++ {
						if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
							return false
						}
					}

					return true
				}); err != nil {
					return err
				}

				t.Logf("All deposit events processed %s", dtype)

				return nil
			}

			if err := depositFunc(erc721ID1, 1, 0); err != nil {
				errChan <- err

				return
			}

			childToken := getChildToken(t, contractsapi.RootERC721Predicate.Abi, sourcePred, tokenAddr, sourceRelayer)

			for i := range transfersCount {
				owner := erc721OwnerOf(t, big.NewInt(int64(i)), childToken, destinationRelayer)
				if owner != accountAddrs[i] {
					errChan <- fmt.Errorf("owner of %d is not same on source & destination", i)

					return
				}
			}

			withdrawFunc := func(shouldThrowError bool, startID, tokenStart uint64) error {
				// withdraw - should be processed because there are enough ERC721 funds
				for i, accountKey := range accountKeys {
					err = bridge.Withdraw(
						common.ERC721,
						accountKey,
						accounts[i],
						"",
						fmt.Sprintf("%d", int(tokenStart)+i),
						destinationRPC,
						destinationPred,
						childToken,
						false)

					if shouldThrowError && err == nil {
						return fmt.Errorf("expected error but got nil %s", dtype)
					}

					if !shouldThrowError && err != nil {
						return err
					}
				}

				if !shouldThrowError {
					// should be processed because there are enough ERC721 funds
					if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
						for i := startID; i < startID+transfersCount; i++ {
							if !isEventProcessed(t, sourceGW, sourceRelayer, i, false) {
								return false
							}
						}

						return true
					}); err != nil {
						return err
					}

					t.Logf("All withdraws processed %s", dtype)

					return nil
				}

				t.Logf("Withdraw failed as expected %s", dtype)

				return nil
			}

			// successful withdraw
			if err := withdrawFunc(false, erc721ID2, 0); err != nil {
				errChan <- err

				return
			}

			// no funds withdraw
			if err := withdrawFunc(true, 0, 0); err != nil {
				errChan <- err

				return
			}

			// deposit again
			if err := depositFunc(erc721ID3, 0, transfersCount); err != nil {
				errChan <- err

				return
			}

			for i := range transfersCount {
				owner := erc721OwnerOf(t, big.NewInt(int64(i+transfersCount)), childToken, destinationRelayer)
				if owner != accountAddrs[i] {
					errChan <- fmt.Errorf("owner of %d is not same on source & destination", i)

					return
				}
			}

			// successful withdraw
			if err := withdrawFunc(false, erc721ID4, transfersCount); err != nil {
				errChan <- err

				return
			}

			for i := range transfersCount {
				owner1 := erc721OwnerOf(t, big.NewInt(int64(i)), tokenAddr, sourceRelayer)
				owner2 := erc721OwnerOf(t, big.NewInt(int64(i+transfersCount)), tokenAddr, sourceRelayer)

				if owner1 != owner2 {
					errChan <- fmt.Errorf("owner of %d is not same on source & destination", i)

					return
				}
			}

			errChan <- nil
		}

		go runTest(bridgeCfg.InternalMintableERC721PredicateAddr, bridgeCfg.ExternalMintableERC721PredicateAddr,
			bridgeCfg.InternalGatewayAddr, bridgeCfg.ExternalGatewayAddr, internalJSONRPCAddr, externalJSONRPCAddr, "I2E", internalChainTxRelayer, externalChainTxRelayer)

		go runTest(bridgeCfg.ExternalERC721PredicateAddr, bridgeCfg.InternalERC721PredicateAddr,
			bridgeCfg.ExternalGatewayAddr, bridgeCfg.InternalGatewayAddr, externalJSONRPCAddr, internalJSONRPCAddr, "E2I", externalChainTxRelayer, internalChainTxRelayer)

		counter := 0

		for {
			select {
			case err := <-errChan:
				if err != nil {
					t.Fatal(err)

					return
				}

				if counter++; counter == 2 {
					return
				}
			case <-time.After(10 * time.Minute):
				t.Fatalf("timeout")
			}
		}
	})

	t.Run("ERC1155", func(t *testing.T) {
		runTest := func(sourcePred, destinationPred, sourceGW, destinationGW types.Address,
			sourceRPC, destinationRPC, dtype string, sourceRelayer, destinationRelayer txrelayer.TxRelayer) {
			// deploy ERC1155 token
			deployTx := types.NewTx(types.NewLegacyTx(
				types.WithTo(nil),
				types.WithInput(contractsapi.RootERC1155.Bytecode),
			))

			receipt, err := sourceRelayer.SendTransaction(deployTx, deployerKey)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

			tokenAddr := types.Address(receipt.ContractAddress)

			depositFunc := func(startID, contract, tokenStart uint64) error {
				// mint & deposit
				for i := range transfersCount {
					if err := bridge.Deposit(
						common.ERC1155,
						tokenAddr,
						sourcePred,
						bridgeHelper.TestAccountPrivKey,
						accounts[i],
						amountStr,
						fmt.Sprintf("%d", int(tokenStart)+i),
						sourceRPC,
						bridgeHelper.TestAccountPrivKey,
						false,
					); err != nil {
						return err
					}
				}

				t.Logf("ERC1155 tokens deposited %s", dtype)

				// should be processed because there are enough ERC1155 funds
				if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
					for i := startID; i < startID+transfersCount+contract; i++ {
						if !isEventProcessed(t, destinationGW, destinationRelayer, i, false) {
							return false
						}
					}

					return true
				}); err != nil {
					return err
				}

				t.Logf("All deposit events processed %s", dtype)

				return nil
			}

			if err := depositFunc(erc1155ID1, 1, 0); err != nil {
				errChan <- err

				return
			}

			childToken := getChildToken(t, contractsapi.RootERC1155Predicate.Abi, sourcePred, tokenAddr, sourceRelayer)

			for i := range transfersCount {
				balance := erc1155BalanceOf(t, accountAddrs[i], childToken, i, destinationRelayer)
				if balance.Cmp(big.NewInt(amount)) != 0 {
					errChan <- fmt.Errorf("balance check failed %s for account %s", dtype, accounts[i])

					return
				}
			}

			withdrawFunc := func(shouldThrowError bool, startID, tokenStart uint64) error {
				// withdraw - should be processed because there are enough ERC1155 funds
				for i, accountKey := range accountKeys {
					err = bridge.Withdraw(
						common.ERC1155,
						accountKey,
						accounts[i],
						amountStr,
						fmt.Sprintf("%d", int(tokenStart)+i),
						destinationRPC,
						destinationPred,
						childToken,
						false)

					if shouldThrowError && err == nil {
						return fmt.Errorf("expected error but got nil %s", dtype)
					}

					if !shouldThrowError && err != nil {
						return err
					}
				}

				if !shouldThrowError {
					// should be processed because there are enough ERC1155 funds
					if err := cluster.WaitUntil(2*time.Minute, 2*time.Second, func() bool {
						for i := startID; i < startID+transfersCount; i++ {
							if !isEventProcessed(t, sourceGW, sourceRelayer, i, false) {
								return false
							}
						}

						return true
					}); err != nil {
						return err
					}

					t.Logf("All withdraws processed %s", dtype)

					return nil
				}

				t.Logf("Withdraw failed as expected %s", dtype)

				return nil
			}

			// successful withdraw
			if err := withdrawFunc(false, erc1155ID2, 0); err != nil {
				errChan <- err

				return
			}

			// no funds withdraw
			if err := withdrawFunc(true, 0, 0); err != nil {
				errChan <- err

				return
			}

			// deposit again
			if err := depositFunc(erc1155ID3, 0, transfersCount); err != nil {
				errChan <- err

				return
			}

			for i := range transfersCount {
				balance := erc1155BalanceOf(t, accountAddrs[i], childToken, i+transfersCount, destinationRelayer)
				if balance.Cmp(big.NewInt(amount)) != 0 {
					errChan <- fmt.Errorf("balance check failed %s for account %s", dtype, accounts[i])

					return
				}
			}

			// successful withdraw
			if err := withdrawFunc(false, erc1155ID4, transfersCount); err != nil {
				errChan <- err

				return
			}

			for i := range transfersCount {
				balance1 := erc1155BalanceOf(t, accountAddrs[i], childToken, i, destinationRelayer)
				if balance1.Cmp(big.NewInt(0)) != 0 {
					errChan <- fmt.Errorf("balance check failed %s for account %s", dtype, accounts[i])

					return
				}

				balance2 := erc1155BalanceOf(t, accountAddrs[i], childToken, i+transfersCount, destinationRelayer)
				if balance2.Cmp(big.NewInt(0)) != 0 {
					errChan <- fmt.Errorf("balance check failed %s for account %s", dtype, accounts[i])

					return
				}
			}

			errChan <- nil
		}

		go runTest(bridgeCfg.InternalMintableERC1155PredicateAddr, bridgeCfg.ExternalMintableERC1155PredicateAddr,
			bridgeCfg.InternalGatewayAddr, bridgeCfg.ExternalGatewayAddr, internalJSONRPCAddr, externalJSONRPCAddr, "I2E", internalChainTxRelayer, externalChainTxRelayer)

		go runTest(bridgeCfg.ExternalERC1155PredicateAddr, bridgeCfg.InternalERC1155PredicateAddr,
			bridgeCfg.ExternalGatewayAddr, bridgeCfg.InternalGatewayAddr, externalJSONRPCAddr, internalJSONRPCAddr, "E2I", externalChainTxRelayer, internalChainTxRelayer)

		counter := 0

		for {
			select {
			case err := <-errChan:
				if err != nil {
					t.Fatal(err)

					return
				}

				if counter++; counter == 2 {
					return
				}
			case <-time.After(10 * time.Minute):
				t.Fatalf("timeout")
			}
		}
	})
}
