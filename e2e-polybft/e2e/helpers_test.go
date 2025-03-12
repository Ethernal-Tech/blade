package e2e

import (
	"math/big"
	"testing"

	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/abi"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/state/runtime/addresslist"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
)

const nativeTokenNonMintableConfig = "Blade:BLD:18:false:1"

func ABICall(relayer txrelayer.TxRelayer, artifact *contracts.Artifact, contractAddress types.Address, senderAddr types.Address, method string, params ...interface{}) (string, error) {
	input, err := artifact.Abi.GetMethod(method).Encode(params)
	if err != nil {
		return "", err
	}

	return relayer.Call(senderAddr, contractAddress, input)
}

func ABITransaction(
	relayer txrelayer.TxRelayer,
	key crypto.Key,
	artifact *contracts.Artifact,
	contractAddress types.Address,
	method string,
	params ...interface{}) (*ethgo.Receipt, error) {
	input, err := artifact.Abi.GetMethod(method).Encode(params)
	if err != nil {
		return nil, err
	}

	tx := types.NewTx(types.NewLegacyTx(
		types.WithTo(&contractAddress),
		types.WithInput(input),
	))

	return relayer.SendTransaction(tx, key)
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

// queryNativeERC20Metadata returns some meta data user requires from native erc20 token
func queryNativeERC20Metadata(t *testing.T, funcName string, abiType *abi.Type, relayer txrelayer.TxRelayer) interface{} {
	t.Helper()

	valueHex, err := ABICall(relayer, contractsapi.NativeERC20Mintable,
		contracts.NativeERC20TokenContract,
		types.ZeroAddress, funcName)
	require.NoError(t, err)

	valueRaw, err := hex.DecodeHex(valueHex)
	require.NoError(t, err)

	var decodedResult map[string]interface{}

	err = abiType.DecodeStruct(valueRaw, &decodedResult)
	require.NoError(t, err)

	return decodedResult["0"]
}
