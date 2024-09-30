package deploy

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/testutil"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/bridge/helper"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/validator"
	"github.com/0xPolygon/polygon-edge/types"
)

func TestDeployContracts_NoPanics(t *testing.T) {
	t.Parallel()

	server := testutil.DeployTestServer(t, nil)
	t.Cleanup(func() {
		if err := os.RemoveAll(params.genesisPath); err != nil {
			t.Fatal(err)
		}
	})

	client, err := jsonrpc.NewEthClient(server.HTTPAddr())
	require.NoError(t, err)

	testKey, err := helper.DecodePrivateKey("")
	require.NoError(t, err)

	receipt, err := server.Fund(ethgo.Address(testKey.Address()))
	require.NoError(t, err)
	require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

	outputter := command.InitializeOutputter(GetCommand())
	params.proxyContractsAdmin = "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"
	consensusCfg = polycfg.PolyBFT{
		NativeTokenConfig: &polycfg.Token{
			Name:       "Test",
			Symbol:     "TST",
			Decimals:   18,
			IsMintable: false,
		},
	}

	chainCfg := &chain.Chain{
		Params: &chain.Params{
			ChainID: 1,
			Engine: map[string]interface{}{
				"polybft": consensusCfg,
			},
		},
		Genesis: &chain.Genesis{
			Alloc: make(map[types.Address]*chain.GenesisAccount),
		},
	}

	require.NotPanics(t, func() {
		_, err = deployContracts(outputter, client, big.NewInt(2), chainCfg, []*validator.GenesisValidator{}, context.Background())
	})
	require.NoError(t, err)
}
