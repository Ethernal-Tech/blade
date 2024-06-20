package crypto

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"

	"github.com/0xPolygon/polygon-edge/types"
)

func TestLondonSignerSender(t *testing.T) {
	t.Parallel()

	recipient := types.StringToAddress("1")

	tcs := []struct {
		name    string
		chainID *big.Int
		txType  types.TxType
	}{
		{
			"mainnet",
			big.NewInt(1),
			types.LegacyTxType,
		},
		{
			"expanse mainnet",
			big.NewInt(2),
			types.DynamicFeeTxType,
		},
		{
			"ropsten",
			big.NewInt(3),
			types.DynamicFeeTxType,
		},
		{
			"rinkeby",
			big.NewInt(4),
			types.AccessListTxType,
		},
		{
			"goerli",
			big.NewInt(5),
			types.AccessListTxType,
		},
		{
			"kovan",
			big.NewInt(42),
			types.StateTxType,
		},
		{
			"geth private",
			big.NewInt(1337),
			types.StateTxType,
		},
		{
			"mega large",
			big.NewInt(0).Exp(big.NewInt(2), big.NewInt(20), nil), // 2**20
			types.AccessListTxType,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key, err := GenerateECDSAPrivateKey()
			require.NoError(t, err, "unable to generate private key")

			var txn *types.Transaction

			switch tc.txType {
			case types.AccessListTxType:
				txn = types.NewTx(types.NewAccessListTx(
					types.WithGasPrice(big.NewInt(5)),
					types.WithChainID(tc.chainID),
					types.WithTo(&recipient),
					types.WithValue(big.NewInt(1)),
				))
			case types.LegacyTxType:
				txn = types.NewTx(types.NewLegacyTx(
					types.WithGasPrice(big.NewInt(5)),
					types.WithTo(&recipient),
					types.WithValue(big.NewInt(1)),
				))
			case types.StateTxType:
				txn = types.NewTx(types.NewStateTx(
					types.WithGasPrice(big.NewInt(5)),
					types.WithTo(&recipient),
					types.WithValue(big.NewInt(1)),
				))
			case types.DynamicFeeTxType:
				txn = types.NewTx(types.NewDynamicFeeTx(
					types.WithChainID(tc.chainID),
					types.WithTo(&recipient),
					types.WithValue(big.NewInt(1)),
				))
			}

			chainID := tc.chainID.Uint64()
			signer := NewLondonSigner(chainID)

			signedTx, err := signer.SignTx(txn, key)
			require.NoError(t, err, "unable to sign transaction")

			sender, err := signer.Sender(signedTx)
			require.NoError(t, err, "failed to recover sender")

			require.Equal(t, sender, PubKeyToAddress(&key.PublicKey))
		})
	}
}

func TestTxSigner_SignCanonical(t *testing.T) {
	t.Parallel()

	key, err := GenerateECDSAPrivateKey()
	require.NoError(t, err, "unable to generate private key")

	to := types.StringToAddress("0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF")

	cases := []struct {
		name          string
		txn           *types.Transaction
		signer        TxSigner
		errorExpected bool
	}{
		{
			name: "LondonSigner - legacy tx",
			txn: types.NewTx(types.NewLegacyTx(
				types.WithGasPrice(big.NewInt(10000)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(1)),
				types.WithNonce(1),
			)),
			signer: NewLondonSigner(100),
		},
		{
			name: "LondonSigner - dynamic tx",
			txn: types.NewTx(types.NewDynamicFeeTx(
				types.WithGasFeeCap(ethgo.Gwei(10)),
				types.WithGasTipCap(ethgo.Gwei(1)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(1)),
				types.WithNonce(1),
			)),
			signer: NewLondonSigner(100),
		},
		{
			name: "BerlinSigner - legacy tx",
			txn: types.NewTx(types.NewLegacyTx(
				types.WithGasPrice(big.NewInt(500)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(11)),
				types.WithNonce(11),
			)),
			signer: NewBerlinSigner(100),
		},
		{
			name: "BerlinSigner - dynamic tx",
			txn: types.NewTx(types.NewDynamicFeeTx(
				types.WithGasFeeCap(ethgo.Gwei(110)),
				types.WithGasTipCap(ethgo.Gwei(121)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(21)),
				types.WithNonce(1),
			)),
			signer:        NewBerlinSigner(100),
			errorExpected: true,
		},
		{
			name: "EIP155Signer - legacy tx",
			txn: types.NewTx(types.NewLegacyTx(
				types.WithGasPrice(big.NewInt(300)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(1311)),
				types.WithNonce(1781),
			)),
			signer: NewEIP155Signer(100),
		},
		{
			name: "EIP155Signer - dynamic tx",
			txn: types.NewTx(types.NewDynamicFeeTx(
				types.WithGasFeeCap(ethgo.Gwei(210)),
				types.WithGasTipCap(ethgo.Gwei(221)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(221)),
				types.WithNonce(11),
			)),
			signer:        NewEIP155Signer(100),
			errorExpected: true,
		},
		{
			name: "HomesteadSigner - legacy tx",
			txn: types.NewTx(types.NewLegacyTx(
				types.WithGasPrice(big.NewInt(700)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(151)),
				types.WithNonce(111),
			)),
			signer: NewHomesteadSigner(),
		},
		{
			name: "HomesteadSigner - dynamic tx",
			txn: types.NewTx(types.NewDynamicFeeTx(
				types.WithGasFeeCap(ethgo.Gwei(310)),
				types.WithGasTipCap(ethgo.Gwei(321)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(321)),
				types.WithNonce(111),
			)),
			signer:        NewHomesteadSigner(),
			errorExpected: true,
		},
		{
			name: "FrontierSigner - legacy tx",
			txn: types.NewTx(types.NewLegacyTx(
				types.WithGasPrice(big.NewInt(1200)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(121)),
				types.WithNonce(113),
			)),
			signer: NewFrontierSigner(),
		},
		{
			name: "FrontierSigner - dynamic tx",
			txn: types.NewTx(types.NewDynamicFeeTx(
				types.WithGasFeeCap(ethgo.Gwei(410)),
				types.WithGasTipCap(ethgo.Gwei(521)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(421)),
				types.WithNonce(1111),
			)),
			signer:        NewFrontierSigner(),
			errorExpected: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sig, err := tc.signer.SignCanonical(tc.txn, key)
			if tc.errorExpected {
				require.Error(t, err, "expected error")
			} else {
				require.NoError(t, err, "unable to sign transaction")
				require.NotEmpty(t, sig)
				require.Equal(t, 65, len(sig))
			}
		})
	}
}

func Test_LondonSigner_Sender(t *testing.T) {
	t.Parallel()

	signer := NewLondonSigner(100)

	to := types.StringToAddress("0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF")

	r, ok := big.NewInt(0).SetString("102623819621514684481463796449525884981685455700611671612296611353030973716382", 10)
	require.True(t, ok)

	s, ok := big.NewInt(0).SetString("52694559292202008915948760944211702951173212957828665318138448463580296965840", 10)
	require.True(t, ok)

	testTable := []struct {
		name   string
		tx     *types.Transaction
		sender types.Address
	}{
		{
			name: "sender is 0x85dA99c8a7C2C95964c8EfD687E95E632Fc533D6",
			tx: types.NewTx(types.NewDynamicFeeTx(
				types.WithChainID(big.NewInt(100)),
				types.WithGasTipCap(ethgo.Gwei(1)),
				types.WithGasFeeCap(ethgo.Gwei(10)),
				types.WithGas(21000),
				types.WithTo(&to),
				types.WithValue(big.NewInt(100000000000000)),
				types.WithSignatureValues(big.NewInt(0), r, s),
			)),
			sender: types.StringToAddress("0x85dA99c8a7C2C95964c8EfD687E95E632Fc533D6"),
		},
	}

	for _, tt := range testTable {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sender, err := signer.Sender(tt.tx)
			require.NoError(t, err)
			require.Equal(t, tt.sender, sender)
		})
	}
}
