package mint

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/0xPolygon/polygon-edge/types"
)

type mintParams struct {
	address          string
	tokens           []string
	amounts          []string
	tokenAddr        string
	minterPrivateKey string
	jsonRPCAddress   string
	txTimeout        time.Duration

	amountValues []*big.Int
	tokensValues []*big.Int
}

func (m *mintParams) validateFlags() error {
	if len(m.tokens) == 0 {
		return errors.New("no tokens provided")
	}

	if len(m.amounts) == 0 {
		return errors.New("no amounts provided")
	}

	if len(m.tokens) != len(m.amounts) {
		return errors.New("tokens (their IDs) and amounts must be equal length")
	}

	if _, err := types.IsValidAddress(m.address, true); err != nil {
		return err
	}

	m.amountValues = make([]*big.Int, len(m.amounts))
	for i, amountRaw := range m.amounts {
		amountValue, err := helper.ParseAmount(amountRaw)
		if err != nil {
			return err
		}

		m.amountValues[i] = amountValue
	}

	m.tokensValues = make([]*big.Int, len(m.tokens))
	for i, tokenRaw := range m.tokens {
		tokenValue, err := helper.ParseID(tokenRaw)
		if err != nil {
			return err
		}

		m.tokensValues[i] = tokenValue
	}

	if _, err := types.IsValidAddress(m.tokenAddr, false); err != nil {
		return fmt.Errorf("invalid erc1155 token address is provided: %w", err)
	}

	return nil
}

type mintResult struct {
	Address types.Address `json:"address"`
	TxHash  types.Hash    `json:"tx_hash"`
}

func (m *mintResult) GetOutput() string {
	var buffer bytes.Buffer

	vals := make([]string, 0, 2)
	vals = append(vals, fmt.Sprintf("Address|%s", m.Address))
	vals = append(vals, fmt.Sprintf("Transaction (hash)|%s", m.TxHash))

	buffer.WriteString("\n[MINT-ERC1155]\n")
	buffer.WriteString(helper.FormatKV(vals))
	buffer.WriteString("\n")

	return buffer.String()
}
