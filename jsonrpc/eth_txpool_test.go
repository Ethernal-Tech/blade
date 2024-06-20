package jsonrpc

import (
	"math/big"
	"testing"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/stretchr/testify/assert"
)

func TestEth_TxnPool_SendRawTransaction(t *testing.T) {
	store := &mockStoreTxn{}
	eth := newTestEthEndpoint(store)
	txn := types.NewTx(types.NewLegacyTx(
		types.WithFrom(addr0),
		types.WithSignatureValues(big.NewInt(1), nil, nil),
	))
	txn.ComputeHash()

	data := txn.MarshalRLP()
	_, err := eth.SendRawTransaction(data)
	assert.NoError(t, err)
	assert.NotEqual(t, store.txn.Hash(), types.ZeroHash)

	// the hash in the txn pool should match the one we send
	if txn.Hash() != store.txn.Hash() {
		t.Fatal("bad")
	}
}

func TestEth_TxnPool_SignTransaction(t *testing.T) {
	store := &mockStoreTxn{}
	store.AddAccount(addr0)

	eth := newTestEthEndpoint(store)
	txToSend := &txnArgs{
		From:     &addr0,
		To:       &addr1,
		Gas:      argUintPtr(100000),
		GasPrice: argBytesPtr([]byte{0x64}),
		Value:    argBytesPtr([]byte{0x64}),
		Data:     nil,
		Nonce:    argUintPtr(0),
	}

	contractKey, err := crypto.GenerateECDSAKey()
	assert.NoError(t, err)

	eth.ecdsaKey = contractKey

	defaultChainID := uint64(100)
	eth.txSigner = crypto.NewSigner(chain.AllForksEnabled.At(0), defaultChainID)

	res, err := eth.SignTransaction(txToSend)
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}

func TestEth_TxnPool_SendTransaction(t *testing.T) {
	store := &mockStoreTxn{}
	store.AddAccount(addr0)

	eth := newTestEthEndpoint(store)
	txToSend := &txnArgs{
		From:     &addr0,
		To:       &addr1,
		Gas:      argUintPtr(100000),
		GasPrice: argBytesPtr([]byte{0x64}),
		Value:    argBytesPtr([]byte{0x64}),
		Data:     nil,
		Nonce:    argUintPtr(0),
	}

	contractKey, err := crypto.GenerateECDSAKey()
	assert.NoError(t, err)

	eth.ecdsaKey = contractKey

	defaultChainID := uint64(100)
	eth.txSigner = crypto.NewSigner(chain.AllForksEnabled.At(0), defaultChainID)

	hash, err := eth.SendTransaction(txToSend)
	assert.NoError(t, err)
	assert.NotEqual(t, store.txn.Hash(), types.ZeroHash)
	assert.NotNil(t, hash)
}

func TestEth_TxnPool_Sign(t *testing.T) {
	store := &mockStoreTxn{}
	eth := newTestEthEndpoint(store)
	txn := types.NewTx(types.NewLegacyTx(
		types.WithFrom(addr0),
		types.WithSignatureValues(big.NewInt(1), nil, nil),
	))
	txn.ComputeHash()

	contractKey, err := crypto.GenerateECDSAKey()
	assert.NoError(t, err)

	eth.ecdsaKey = contractKey

	defaultChainID := uint64(100)
	eth.txSigner = crypto.NewSigner(chain.AllForksEnabled.At(0), defaultChainID)

	data := txn.MarshalRLP()
	dataSignInterface, err := eth.Sign(addr0, data)
	assert.NoError(t, err)
	assert.NotNil(t, dataSignInterface)

	dataSign := dataSignInterface.(*argBytes)
	assert.Greater(t, (*dataSign)[64], byte(0))
}

type mockStoreTxn struct {
	ethStore
	accounts map[types.Address]*mockAccount
	txn      *types.Transaction
}

func (m *mockStoreTxn) AddTx(tx *types.Transaction) error {
	m.txn = tx

	tx.ComputeHash()

	return nil
}

func (m *mockStoreTxn) GetNonce(addr types.Address) uint64 {
	return 1
}

func (m *mockStoreTxn) GetForksInTime(blockNumber uint64) chain.ForksInTime {
	return chain.AllForksEnabled.At(0)
}

func (m *mockStoreTxn) AddAccount(addr types.Address) *mockAccount {
	if m.accounts == nil {
		m.accounts = map[types.Address]*mockAccount{}
	}

	acct := &mockAccount{
		address: addr,
		account: &Account{},
		storage: make(map[types.Hash][]byte),
	}
	m.accounts[addr] = acct

	return acct
}

func (m *mockStoreTxn) Header() *types.Header {
	return &types.Header{}
}

func (m *mockStoreTxn) GetAccount(root types.Hash, addr types.Address) (*Account, error) {
	acct, ok := m.accounts[addr]
	if !ok {
		return nil, ErrStateNotFound
	}

	return acct.account, nil
}
