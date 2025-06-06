package e2e

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/command/bridge/helper"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/wallet"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/stretchr/testify/require"
)

type confirmedBatch struct {
	BatchID    *big.Int `abi:"batchID"`
	Bitmap     *big.Int `abi:"bitmap"`
	RawTx      []byte   `abi:"rawTx"`
	Signatures [][]byte `abi:"signatures"`
}

func newConfirmedBatch(mp map[string]interface{}) *confirmedBatch {
	return &confirmedBatch{
		BatchID:    mp["batchID"].(*big.Int),
		Bitmap:     mp["bitmap"].(*big.Int),
		RawTx:      mp["rawTx"].([]byte),
		Signatures: mp["signatures"].([][]byte),
	}
}

func (cb confirmedBatch) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ID      = %s\n", cb.BatchID))
	sb.WriteString(fmt.Sprintf("Bmp     = %s\n", cb.Bitmap))
	sb.WriteString(fmt.Sprintf("RawTx = %s\n", hex.EncodeToString(cb.RawTx)))

	for i, x := range cb.Signatures {
		sb.WriteString(fmt.Sprintf("Sign(%d) = %s\n", i, hex.EncodeToString(x)))
	}

	return sb.String()
}

func TestE2E_ApexBridge_TestPerformance(t *testing.T) {
	admin, err := wallet.GenerateAccount()
	require.NoError(t, err)

	// default premined account for geth in --dev mode
	privateKey, err := crypto.HexToECDSA(helper.TestAccountPrivKey)
	require.NoError(t, err)

	admin.Ecdsa = crypto.NewECDSAKey(privateKey)

	fmt.Printf("Admin address: %s\n", admin.Address())

	cluster := framework.NewTestCluster(
		t, 4,
		framework.WithBlockGasLimit(16_000_000),
		framework.WithBladeAdmin(admin.Address().String()))

	cluster.WaitForReady(t)

	defer func() {
		cluster.Stop()
	}()

	testPerformance(
		t, "blade pebble", cluster.Servers[0].JSONRPC(), admin,
		filepath.Join(cluster.Config.TmpDir, "test-chain-1", "trie"), false)

	cluster.Stop()

	cluster = framework.NewTestCluster(
		t, 4,
		framework.WithBlockGasLimit(16_000_000),
		framework.WithBladeAdmin(admin.Address().String()),
		framework.WithDBEngine("leveldb"))

	cluster.WaitForReady(t)

	testPerformance(
		t, "blade leveldb", cluster.Servers[0].JSONRPC(), admin,
		filepath.Join(cluster.Config.TmpDir, "test-chain-1", "trie"), false)

	cluster.Stop()

	testBridge, err := framework.NewTestBridge(t, cluster.Config)
	require.NoError(t, err)

	testBridgeClient, err := jsonrpc.NewEthClient(testBridge.JSONRPCAddr())
	require.NoError(t, err)

	testBridge.Start()

	defer testBridge.Stop()

	rootChainDirectoryBinding, err := helper.ReadRootchainDirectoryBinding()
	require.NoError(t, err)

	fmt.Printf("rootchain directory: %s\n", rootChainDirectoryBinding)

	testPerformance(
		t, "geth", testBridgeClient, admin, rootChainDirectoryBinding, true)
}

func testPerformance(
	t *testing.T, testName string, client *jsonrpc.EthClient,
	admin *wallet.Account, triePath string, geth bool,
) {
	t.Helper()

	const (
		validatorsCount                    = 4
		quorumCnt                          = 4
		checkBatchID                       = true
		deleteTemporaryMappingsAfterQuorum = true
		batchesCount                       = 6
		txSize                             = 8192
		waitForConsolidation               = time.Second * 60 * 5
	)

	validators := make([]*wallet.Account, validatorsCount)

	for i := range validators {
		acc, err := wallet.GenerateAccount()
		require.NoError(t, err)

		validators[i] = acc
	}

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(client))
	require.NoError(t, err)

	if geth {
		amount := ethgo.Ether(100)

		for _, acc := range append([]*wallet.Account{admin}, validators...) {
			receipt, err := txRelayer.SendTransactionLocal(
				types.NewTx(types.NewLegacyTx(
					types.WithTo(acc.Address().Ptr()),
					types.WithValue(amount),
				)))
			require.NoError(t, err)
			require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)
		}
	}

	input, err := contractsapi.TestPerformance.Abi.Constructor.Inputs.Encode([]interface{}{
		big.NewInt(quorumCnt), checkBatchID, deleteTemporaryMappingsAfterQuorum,
	})
	require.NoError(t, err)

	receipt, err := txRelayer.SendTransaction(
		types.NewTx(types.NewLegacyTx(
			types.WithFrom(admin.Ecdsa.Address()),
			types.WithInput(append(contractsapi.TestPerformance.Bytecode, input...)),
			types.WithGas(8_242_880),
		)),
		admin.Ecdsa)
	require.NoError(t, err)
	require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

	contractAddr := types.Address(receipt.ContractAddress)

	getTotalTrieSize := func(t *testing.T) (total int64) {
		t.Helper()

		if geth {
			return 0
		}

		err := filepath.Walk(triePath, func(_ string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				total += info.Size()
			}

			return nil
		})

		require.NoError(t, err)

		return total
	}

	rndBytes := func(size int) []byte {
		token := make([]byte, size)
		_, _ = rand.Read(token)

		return token
	}

	submitBatch := func(t *testing.T, validatorID uint8, batchID uint64, rawTx []byte, signature []byte) {
		t.Helper()

		signedBatch := []any{
			new(big.Int).SetUint64(batchID),
			new(big.Int).SetUint64(uint64(validatorID)),
			rawTx,
			signature,
		}
		validatorAcc := validators[validatorID-1]

		fn := contractsapi.TestPerformance.Abi.GetMethod("submitSignedBatch")
		input, err := fn.Encode([]interface{}{signedBatch})
		require.NoError(t, err)

		txn := types.NewTx(types.NewLegacyTx(
			types.WithFrom(validatorAcc.Address()),
			types.WithTo(&contractAddr),
			types.WithInput(input),
			types.WithGas(8_000_000),
		))

		receipt, err = txRelayer.SendTransaction(txn, validatorAcc.Ecdsa)
		require.NoError(t, err)
		require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)
	}

	getConfirmedBatch := func(t *testing.T) *confirmedBatch {
		t.Helper()

		fn := contractsapi.TestPerformance.Abi.GetMethod("getConfirmedBatch")
		input, err := fn.Encode([]interface{}{})
		require.NoError(t, err)

		response, err := txRelayer.Call(types.ZeroAddress, contractAddr, input)
		require.NoError(t, err)

		byteResponse, err := hex.DecodeHex(response)
		require.NoError(t, err)

		decoded, err := fn.Outputs.Decode(byteResponse)
		require.NoError(t, err)

		base := decoded.(map[string]interface{})
		if len(base) == 0 {
			return nil
		}

		item := base["0"].(map[string]interface{})

		return newConfirmedBatch(item)
	}

	getHashesCount := func(t *testing.T) uint64 {
		t.Helper()

		fn := contractsapi.TestPerformance.Abi.GetMethod("getHashesCount")
		input, err := fn.Encode([]interface{}{})
		require.NoError(t, err)

		response, err := txRelayer.Call(types.ZeroAddress, contractAddr, input)
		require.NoError(t, err)

		result, err := common.ParseUint64orHex(&response)
		require.NoError(t, err)

		return result
	}

	getLastBatchID := func(t *testing.T) uint64 {
		t.Helper()

		fn := contractsapi.TestPerformance.Abi.GetMethod("getLastBatchID")
		input, err := fn.Encode([]interface{}{})
		require.NoError(t, err)

		response, err := txRelayer.Call(types.ZeroAddress, contractAddr, input)
		require.NoError(t, err)

		result, err := common.ParseUint64orHex(&response)
		require.NoError(t, err)

		return result
	}

	require.Equal(t, uint64(0), getLastBatchID(t))

	wg := sync.WaitGroup{}

	fmt.Printf("%s (tx size = %d) trie size = %d\n", testName, txSize, getTotalTrieSize(t))

	for i := uint64(1); i <= batchesCount; i++ {
		txRaw := rndBytes(txSize)

		fmt.Printf("processing %s (tx size = %d) %d/%d.\n", testName, txSize, i, batchesCount)

		for j := uint8(1); j <= validatorsCount; j++ {
			wg.Add(1)

			go func(validatorID uint8, batchID uint64) {
				defer wg.Done()

				submitBatch(t, validatorID, batchID, txRaw, rndBytes(64))
			}(j, i)
		}

		wg.Wait()

		confirmedBatch := getConfirmedBatch(t)

		require.Equal(t, new(big.Int).SetUint64(i), confirmedBatch.BatchID)
		require.Equal(t, txRaw, confirmedBatch.RawTx)
		require.Equal(t, i, getLastBatchID(t))

		fmt.Printf("trie size = %d\n", getTotalTrieSize(t))
	}

	require.Equal(t, uint64(batchesCount), getHashesCount(t))

	<-time.After(waitForConsolidation)

	fmt.Printf("%s (tx size = %d) after consolidation trie size = %d\n", testName, txSize, getTotalTrieSize(t))
}
