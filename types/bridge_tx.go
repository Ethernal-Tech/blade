package types

import (
	"fmt"
	"math/big"

	"github.com/umbracle/fastrlp"
	"github.com/valyala/fastjson"
)

type BridgeTx struct {
	*BaseTx
	GasPrice *big.Int
}

func NewBridgeTx(options ...TxOption) *BridgeTx {
	bridgeTx := &BridgeTx{BaseTx: &BaseTx{}}

	for _, opt := range options {
		opt(bridgeTx)
	}

	return bridgeTx
}

func (tx *BridgeTx) transactionType() TxType { return BridgeTxType }
func (tx *BridgeTx) chainID() *big.Int       { return deriveChainID(tx.v()) }
func (tx *BridgeTx) gasPrice() *big.Int      { return tx.GasPrice }
func (tx *BridgeTx) gasTipCap() *big.Int     { return tx.GasPrice }
func (tx *BridgeTx) gasFeeCap() *big.Int     { return tx.GasPrice }
func (tx *BridgeTx) effectiveGasPrice(baseFee *big.Int) *big.Int {
	return new(big.Int).Set(tx.gasPrice())
}

func (tx *BridgeTx) accessList() TxAccessList {
	return nil
}

// set methods for transaction fields
func (tx *BridgeTx) setChainID(id *big.Int) {}

func (tx *BridgeTx) setGasPrice(gas *big.Int) {
	tx.GasPrice = gas
}

func (tx *BridgeTx) setGasFeeCap(gas *big.Int) {
	tx.GasPrice = gas
}

func (tx *BridgeTx) setGasTipCap(gas *big.Int) {
	tx.GasPrice = gas
}

func (tx *BridgeTx) setAccessList(accessList TxAccessList) {}

// unmarshalRLPFrom unmarshals a Transaction in RLP format
// Be careful! This function does not de-serialize tx type, it assumes that t.Type is already set
// Hash calculation should also be done from the outside!
// Use UnmarshalRLP in most cases
func (tx *BridgeTx) unmarshalRLPFrom(p *fastrlp.Parser, v *fastrlp.Value) error {
	numOfElems := 10

	var (
		values rlpValues
		err    error
	)

	values, err = v.GetElems()
	if err != nil {
		return err
	}

	if numElems := len(values); numElems != numOfElems {
		return fmt.Errorf("incorrect number of transaction elements, expected %d but found %d", numOfElems, numElems)
	}

	// nonce
	txNonce, err := values.dequeueValue().GetUint64()
	if err != nil {
		return err
	}

	tx.setNonce(txNonce)

	// gasPrice
	txGasPrice := new(big.Int)
	if err = values.dequeueValue().GetBigInt(txGasPrice); err != nil {
		return err
	}

	tx.setGasPrice(txGasPrice)

	// gas
	txGas, err := values.dequeueValue().GetUint64()
	if err != nil {
		return err
	}

	tx.setGas(txGas)

	// to
	if vv, _ := values.dequeueValue().Bytes(); len(vv) == 20 {
		// address
		addr := BytesToAddress(vv)
		tx.setTo(&addr)
	} else {
		// reset To
		tx.setTo(nil)
	}

	// value
	txValue := new(big.Int)
	if err = values.dequeueValue().GetBigInt(txValue); err != nil {
		return err
	}

	tx.setValue(txValue)

	// input
	var txInput []byte

	txInput, err = values.dequeueValue().GetBytes(txInput)
	if err != nil {
		return err
	}

	tx.setInput(txInput)

	// V
	txV := new(big.Int)
	if err = values.dequeueValue().GetBigInt(txV); err != nil {
		return err
	}

	// R
	txR := new(big.Int)
	if err = values.dequeueValue().GetBigInt(txR); err != nil {
		return err
	}

	// S
	txS := new(big.Int)
	if err = values.dequeueValue().GetBigInt(txS); err != nil {
		return err
	}

	tx.setSignatureValues(txV, txR, txS)

	tx.setFrom(ZeroAddress)

	// We need to set From field for state transaction,
	// because we are using unique, predefined address, for sending such transactions
	if vv, err := values.dequeueValue().Bytes(); err == nil && len(vv) == AddressLength {
		// address
		tx.setFrom(BytesToAddress(vv))
	}

	return nil
}

// MarshalRLPWith marshals the transaction to RLP with a specific fastrlp.Arena
// Be careful! This function does not serialize tx type as a first byte.
// Use MarshalRLP/MarshalRLPTo in most cases
func (tx *BridgeTx) marshalRLPWith(arena *fastrlp.Arena) *fastrlp.Value {
	vv := arena.NewArray()

	vv.Set(arena.NewUint(tx.nonce()))
	vv.Set(arena.NewBigInt(tx.gasPrice()))
	vv.Set(arena.NewUint(tx.gas()))
	// Address may be empty
	if tx.to() != nil {
		vv.Set(arena.NewCopyBytes(tx.to().Bytes()))
	} else {
		vv.Set(arena.NewNull())
	}

	vv.Set(arena.NewBigInt(tx.value()))
	vv.Set(arena.NewCopyBytes(tx.input()))

	// signature values
	v, r, s := tx.rawSignatureValues()
	vv.Set(arena.NewBigInt(v))
	vv.Set(arena.NewBigInt(r))
	vv.Set(arena.NewBigInt(s))

	vv.Set(arena.NewCopyBytes(tx.from().Bytes()))

	return vv
}

func (tx *BridgeTx) copy() TxData {
	cpy := NewStateTx()

	if tx.gasPrice() != nil {
		gasPrice := new(big.Int)
		gasPrice.Set(tx.gasPrice())

		cpy.setGasPrice(gasPrice)
	}

	cpy.BaseTx = tx.BaseTx.copy()

	return cpy
}

func (tx *BridgeTx) marshalJSON(a *fastjson.Arena) *fastjson.Value {
	v := a.NewObject()

	tx.BaseTx.marshalJSON(a, v)
	v.Set("type", a.NewString(tx.transactionType().ToHexString()))

	if tx.GasPrice != nil {
		v.Set("gasPrice", a.NewString(fmt.Sprintf("0x%x", tx.GasPrice)))
	}

	return v
}

func (tx *BridgeTx) unmarshalJSON(v *fastjson.Value) error {
	if err := tx.BaseTx.unmarshalJSON(v); err != nil {
		return err
	}

	gasPrice, err := UnmarshalJSONBigInt(v, "gasPrice")
	if err != nil {
		return err
	}

	tx.setGasPrice(gasPrice)

	return nil
}
