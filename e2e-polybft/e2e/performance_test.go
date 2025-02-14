package e2e

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/wallet"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/stretchr/testify/require"
)

type confirmedBatch struct {
	BatchID    *big.Int `abi:"batchID"`
	Bitmap     *big.Int `abi:"bitmap"`
	Counter    *big.Int `abi:"counter"`
	Signatures [][]byte `abi:"signatures"`
}

func newConfirmedBatch(mp map[string]interface{}) *confirmedBatch {
	return &confirmedBatch{
		BatchID:    mp["batchID"].(*big.Int),    //nolint
		Bitmap:     mp["bitmap"].(*big.Int),     //nolint
		Counter:    mp["counter"].(*big.Int),    //nolint
		Signatures: mp["signatures"].([][]byte), //nolint
	}
}

func (cb confirmedBatch) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ID      = %s\n", cb.BatchID))
	sb.WriteString(fmt.Sprintf("Bmp     = %s\n", cb.Bitmap))
	sb.WriteString(fmt.Sprintf("Counter = %s\n", cb.Counter))

	for i, x := range cb.Signatures {
		sb.WriteString(fmt.Sprintf("Sign(%d) = %s\n", i, hex.EncodeToString(x)))
	}

	return sb.String()
}

func TestE2E_ApexBridge_TestCardanoVerifySignaturePrecompile(t *testing.T) {
	const quorumCnt = 5

	admin, err := wallet.GenerateAccount()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(
		t, 4, framework.WithBladeAdmin(admin.Address().String()))

	defer cluster.Stop()

	cluster.WaitForReady(t)

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(cluster.Servers[0].JSONRPC()))
	require.NoError(t, err)

	input, err := contractsapi.TestPerformance.Abi.Constructor.Inputs.Encode([]interface{}{big.NewInt(quorumCnt)})
	require.NoError(t, err)

	// deploy contract
	receipt, err := txRelayer.SendTransaction(
		types.NewTx(types.NewLegacyTx(
			types.WithFrom(admin.Ecdsa.Address()),
			types.WithInput(append(contractsapi.TestPerformance.Bytecode, input...)),
		)),
		admin.Ecdsa)
	require.NoError(t, err)

	contractAddr := types.Address(receipt.ContractAddress)

	rndBytes := func(size int) []byte {
		token := make([]byte, size)
		_, _ = rand.Read(token)

		return token
	}

	submitBatch := func(t *testing.T, validatorID uint8, batchID uint64, counter uint64, signature []byte) {
		t.Helper()

		signedBatch := []interface{}{
			new(big.Int).SetUint64(batchID),
			new(big.Int).SetUint64(counter),
			new(big.Int).SetUint64(uint64(validatorID)),
			signature,
		}

		fn := contractsapi.TestPerformance.Abi.GetMethod("submitSignedBatch")
		input, err := fn.Encode([]interface{}{signedBatch})
		require.NoError(t, err)

		txn := types.NewTx(types.NewLegacyTx(
			types.WithFrom(admin.Address()),
			types.WithTo(&contractAddr),
			types.WithInput(input),
		))

		receipt, err = txRelayer.SendTransaction(txn, admin.Ecdsa)
		require.NoError(t, err)
		require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)
	}

	getConfirmedBatches := func(t *testing.T) []*confirmedBatch {
		t.Helper()

		fn := contractsapi.TestPerformance.Abi.GetMethod("getConfirmedBatches")
		input, err := fn.Encode([]interface{}{})
		require.NoError(t, err)

		response, err := txRelayer.Call(types.ZeroAddress, contractAddr, input)
		require.NoError(t, err)

		byteResponse, err := hex.DecodeHex(response)
		require.NoError(t, err)

		decoded, err := fn.Outputs.Decode(byteResponse)
		require.NoError(t, err)

		base := decoded.(map[string]interface{}) //nolint
		if len(base) == 0 {
			return nil
		}

		items := base["0"].([]map[string]interface{}) //nolint
		result := make([]*confirmedBatch, len(items))

		for i, x := range items {
			result[i] = newConfirmedBatch(x)
		}

		return result
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

	submitBatch(t, 1, 10, 100, rndBytes(64))
	submitBatch(t, 2, 10, 100, rndBytes(64))
	submitBatch(t, 3, 10, 100, rndBytes(64))
	submitBatch(t, 4, 10, 100, rndBytes(64))
	submitBatch(t, 5, 10, 100, rndBytes(64))

	submitBatch(t, 1, 2, 100, rndBytes(64))
	submitBatch(t, 2, 2, 100, rndBytes(64))
	submitBatch(t, 3, 2, 100, rndBytes(64))
	submitBatch(t, 4, 2, 100, rndBytes(64))
	submitBatch(t, 5, 2, 100, rndBytes(64))

	submitBatch(t, 5, 12, 100, rndBytes(64))
	submitBatch(t, 3, 42, 200, rndBytes(64))

	confirmedBatches := getConfirmedBatches(t)

	require.Len(t, confirmedBatches, 2)
	require.Equal(t, uint64(4), getHashesCount(t))

	for _, x := range confirmedBatches {
		fmt.Println(x)
	}
}
