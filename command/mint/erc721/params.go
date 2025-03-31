package mint

import (
	"bytes"
	"fmt"
	"time"

	rootHelper "github.com/0xPolygon/polygon-edge/command/bridge/helper"
	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/0xPolygon/polygon-edge/types"
)

type mintParams struct {
	addresses        []string
	tokenAddr        string
	minterPrivateKey string
	jsonRPCAddress   string
	txTimeout        time.Duration
}

func (m *mintParams) validateFlags() error {
	if len(m.addresses) == 0 {
		return rootHelper.ErrNoAddressesProvided
	}

	for _, addr := range m.addresses {
		if _, err := types.IsValidAddress(addr, true); err != nil {
			return err
		}
	}

	if _, err := types.IsValidAddress(m.tokenAddr, false); err != nil {
		return fmt.Errorf("invalid erc721 token address is provided: %w", err)
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

	buffer.WriteString("\n[MINT-ERC721]\n")
	buffer.WriteString(helper.FormatKV(vals))
	buffer.WriteString("\n")

	return buffer.String()
}
