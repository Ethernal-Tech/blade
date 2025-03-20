package bridge

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/abi"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/state/runtime/addresslist"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
)

const nativeTokenNonMintableConfig = "Blade:BLD:18:false:1"

// checkBridgeMessageResultLogs is helper function which parses given BridgeMessageResultEvent event's logs,
// extracts status topic value and makes assertions against it.
func checkBridgeMessageResultLogs(
	t *testing.T,
	logs []*ethgo.Log,
	expectedCount int,
	handler func(*testing.T, contractsapi.BridgeMessageResultEvent),
) {
	t.Helper()
	require.Equal(t, expectedCount, len(logs))

	var bridgeMessageResultEvent contractsapi.BridgeMessageResultEvent

	for _, log := range logs {
		doesMatch, err := bridgeMessageResultEvent.ParseLog(log)
		require.NoError(t, err)
		require.True(t, doesMatch)

		t.Logf("Block Number=%d, Decoded Log=%+v\n", log.BlockNumber, bridgeMessageResultEvent)

		if handler != nil {
			handler(t, bridgeMessageResultEvent)
		}
	}
}

// assertBridgeEventResultSuccess asserts that:
// 1. there are required amount of logs,
// 2. they are of contractsapi.BridgeMessageResult type
// 3. status is true, meaning that the state syncs were executed successfully
func assertBridgeEventResultSuccess(
	t *testing.T,
	logs []*ethgo.Log,
	expectedCount int) {
	t.Helper()
	checkBridgeMessageResultLogs(t, logs, expectedCount,
		func(t *testing.T, ssre contractsapi.BridgeMessageResultEvent) {
			t.Helper()

			require.True(t, ssre.Status)
		})
}

// assertBridgeEventResultNotSuccessfull asserts that:
// 1. there are required amount of logs,
// 2. they are of contractsapi.BridgeMessageResult type
// 3. status is true, meaning that the state syncs were executed successfully
func assertBridgeEventResultNotSuccessful(
	t *testing.T,
	logs []*ethgo.Log,
	expectedCount int) {
	t.Helper()

	numberOfFailed := 0

	var bridgeMessage contractsapi.BridgeMessageResultEvent

	for _, log := range logs {
		doesMatch, err := bridgeMessage.ParseLog(log)
		require.NoError(t, err)
		require.True(t, doesMatch)

		if !bridgeMessage.Status {
			numberOfFailed++
		}
	}

	require.Equal(t, expectedCount, numberOfFailed)
}

// assertBridgeEventResultNotSuccessfull asserts that:
// 1. there are required amount of logs,
// 2. they are of contractsapi.BridgeMessageResult type
// 3. status is true, meaning that the state syncs were executed successfully
func assertBridgeEventResultSuccessful(
	t *testing.T,
	logs []*ethgo.Log,
	expectedCount int) {
	t.Helper()

	numberOfSuccessful := 0

	var bridgeMessage contractsapi.BridgeMessageResultEvent

	for _, log := range logs {
		doesMatch, err := bridgeMessage.ParseLog(log)
		require.NoError(t, err)
		require.True(t, doesMatch)

		if bridgeMessage.Status {
			numberOfSuccessful++
		}
	}

	require.Equal(t, expectedCount, numberOfSuccessful)
}

// setAccessListRole sets access list role to appropriate access list precompile
func setAccessListRole(t *testing.T, cluster *framework.TestCluster, precompile, account types.Address,
	role addresslist.Role, aclAdmin *crypto.ECDSAKey) {
	t.Helper()

	var updateRoleFn *abi.Method

	switch role {
	case addresslist.AdminRole:
		updateRoleFn = addresslist.SetAdminFunc
	case addresslist.EnabledRole:
		updateRoleFn = addresslist.SetEnabledFunc
	case addresslist.NoRole:
		updateRoleFn = addresslist.SetNoneFunc
	}

	input, err := updateRoleFn.Encode([]interface{}{account})
	require.NoError(t, err)

	enableSetTxn := cluster.MethodTxn(t, aclAdmin, precompile, input)
	require.True(t, enableSetTxn.Succeed())

	expectRole(t, cluster, precompile, account, role)
}

func expectRole(t *testing.T, cluster *framework.TestCluster, contract types.Address, addr types.Address, role addresslist.Role) {
	t.Helper()
	out := cluster.Call(t, contract, addresslist.ReadAddressListFunc, addr)

	num, ok := out["0"].(*big.Int)
	if !ok {
		t.Fatal("unexpected")
	}

	require.Equal(t, role.Uint64(), num.Uint64())
}

// getFilteredLogs retrieves Ethereum logs, described by event signature within the block range
func getFilteredLogs(eventSig ethgo.Hash, startBlock, endBlock uint64,
	ethEndpoint *jsonrpc.EthClient) ([]*ethgo.Log, error) {
	filter := &ethgo.LogFilter{Topics: [][]*ethgo.Hash{{&eventSig}}}

	filter.SetFromUint64(startBlock)
	filter.SetToUint64(endBlock)

	return ethEndpoint.GetLogs(filter)
}

// erc20BalanceOf returns balance of given account on ERC 20 token
func erc20BalanceOf(t *testing.T, account types.Address, tokenAddr types.Address, relayer txrelayer.TxRelayer) *big.Int {
	t.Helper()

	balanceOfFn := &contractsapi.BalanceOfRootERC20Fn{Account: account}
	balanceOfInput, err := balanceOfFn.EncodeAbi()
	require.NoError(t, err)

	balanceRaw, err := relayer.Call(types.ZeroAddress, tokenAddr, balanceOfInput)
	require.NoError(t, err)
	balance, err := common.ParseUint256orHex(&balanceRaw)
	require.NoError(t, err)

	return balance
}

// erc721OwnerOf returns owner of given ERC 721 token
func erc721OwnerOf(t *testing.T, tokenID *big.Int, tokenAddr types.Address, relayer txrelayer.TxRelayer) types.Address {
	t.Helper()

	ownerOfFn := &contractsapi.OwnerOfChildERC721Fn{TokenID: tokenID}
	ownerOfInput, err := ownerOfFn.EncodeAbi()
	require.NoError(t, err)

	ownerRaw, err := relayer.Call(types.ZeroAddress, tokenAddr, ownerOfInput)
	require.NoError(t, err)

	return types.StringToAddress(ownerRaw)
}

// getChildToken queries child token address for provided root token on the target predicate
func getChildToken(t *testing.T, predicateABI *abi.ABI, predicateAddr types.Address,
	rootToken types.Address, relayer txrelayer.TxRelayer) types.Address {
	t.Helper()

	sourceToDestTokenMapFn, exists := predicateABI.Methods["sourceTokenToDestinationToken"]
	require.True(t, exists, "rootTokenToChildToken function is not found in the provided predicate ABI definition")

	input, err := sourceToDestTokenMapFn.Encode([]interface{}{rootToken})
	require.NoError(t, err)

	childTokenRaw, err := relayer.Call(types.ZeroAddress, predicateAddr, input)
	require.NoError(t, err)

	return types.StringToAddress(childTokenRaw)
}

func isEventProcessed(t *testing.T, gatewayAddr types.Address,
	relayer txrelayer.TxRelayer, bridgeEventID uint64, isRollback bool) bool {
	t.Helper()

	processedEventsFn := contractsapi.Gateway.Abi.Methods["processedEvents"]

	if isRollback {
		processedEventsFn = contractsapi.Gateway.Abi.Methods["processedEventsRollback"]
	}

	input, err := processedEventsFn.Encode([]interface{}{bridgeEventID})
	require.NoError(t, err)

	isProcessedRaw, err := relayer.Call(types.ZeroAddress, gatewayAddr, input)
	require.NoError(t, err)

	isProcessedAsNumber, err := common.ParseUint64orHex(&isProcessedRaw)
	require.NoError(t, err)

	return isProcessedAsNumber == 1
}

// Waits for number of blocks specified from current on external
// returns last block number
func waitForBlocksOnExternal(t *testing.T, numberOfBlocks uint64,
	externalRPC *jsonrpc.EthClient, timeToWait time.Duration) uint64 {
	t.Helper()

	latest, err := externalRPC.BlockNumber()
	require.NoError(t, err)

	ticker := time.NewTicker(time.Second)
	timer := time.NewTimer(timeToWait)

	waitFor := latest + numberOfBlocks

	for {
		select {
		case <-ticker.C:
			latest, err := externalRPC.BlockNumber()
			require.NoError(t, err)

			if latest >= waitFor {
				return latest
			}
		case <-timer.C:
			t.Fatalf("External chain didn't get to %d at time", waitFor)
		}
	}
}

// compareBucketsFromDBs compares a bucket from db1 with a bucket from db2
func compareBucketsFromDBs(t *testing.T, db1, db2 *bbolt.DB, sourceChainID, destinationChainID []byte, isRollback bool) {
	// Open read transactions for both databases
	require.NoError(t, db1.View(func(tx1 *bbolt.Tx) error {
		require.NoError(t, db2.View(func(tx2 *bbolt.Tx) error {
			bridgeMessageBucket1 := tx1.Bucket([]byte("bridgeMessageEvents"))
			bridgeMessageBucket2 := tx2.Bucket([]byte("bridgeMessageEvents"))

			if bridgeMessageBucket1 == nil || bridgeMessageBucket2 == nil {
				return fmt.Errorf("one or both buckets do not exist 1")
			}

			bridgeMessageChainIDBucket1 := bridgeMessageBucket1.Bucket(sourceChainID)
			bridgeMessageChainIDBucket2 := bridgeMessageBucket2.Bucket(sourceChainID)

			if bridgeMessageChainIDBucket1 == nil || bridgeMessageChainIDBucket2 == nil {
				return fmt.Errorf("one or both buckets do not exist 2")
			}

			bridgeMessageChainIDBucket1External := bridgeMessageChainIDBucket1.Bucket(destinationChainID)
			bridgeMessageChainIDBucket2External := bridgeMessageChainIDBucket2.Bucket(destinationChainID)
			if bridgeMessageChainIDBucket1External == nil || bridgeMessageChainIDBucket2External == nil {
				return fmt.Errorf("one or both buckets do not exist 3")
			}

			var (
				finalBridgeMessageBucket1 *bbolt.Bucket
				finalBridgeMessageBucket2 *bbolt.Bucket
			)

			if isRollback {
				finalBridgeMessageBucket1 = bridgeMessageChainIDBucket1External.Bucket([]byte("rollback"))
				finalBridgeMessageBucket2 = bridgeMessageChainIDBucket2External.Bucket([]byte("rollback"))
				if finalBridgeMessageBucket1 == nil || finalBridgeMessageBucket2 == nil {
					return fmt.Errorf("one or both buckets do not exist 4")
				}
			} else {
				finalBridgeMessageBucket1 = bridgeMessageChainIDBucket1External.Bucket([]byte("ordinary"))
				finalBridgeMessageBucket2 = bridgeMessageChainIDBucket2External.Bucket([]byte("ordinary"))
				if finalBridgeMessageBucket1 == nil || finalBridgeMessageBucket2 == nil {
					return fmt.Errorf("one or both buckets do not exist 4")
				}
			}

			// Compare keys and values in db1 -> db2
			err := finalBridgeMessageBucket1.ForEach(func(k, v1 []byte) error {
				v2 := finalBridgeMessageBucket2.Get(k)
				if v2 == nil {
					return fmt.Errorf("Key %s is missing in %s (DB2)\n", k, "ordinary bucket")
				} else if string(v1) != string(v2) {
					return fmt.Errorf("Key %s has different values: %s (DB1) vs %s (DB2)\n", k, v1, v2)
				}
				return nil
			})
			if err != nil {
				return err
			}

			// Check for extra keys in db2 -> db1
			err = finalBridgeMessageBucket2.ForEach(func(k, _ []byte) error {
				if finalBridgeMessageBucket1.Get(k) == nil {
					return fmt.Errorf("Key %s is missing in %s (DB1)\n", k, "ordinary bucket")
				}
				return nil
			})
			return err
		}))

		return nil
	}))
}
