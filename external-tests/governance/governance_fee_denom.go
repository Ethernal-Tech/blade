package governance

import (
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/types"
)

type ChangeBaseFeeDenom struct {
	*BaseGovernanceTest
}

func NewBaseFeeDenomTest(cfg *GovernanceTestConfig,
	client *jsonrpc.EthClient) (*ChangeBaseFeeDenom, error) {
	base, err := NewBaseGovernanceTest(cfg, client)
	if err != nil {
		return nil, err
	}

	return &ChangeBaseFeeDenom{
		BaseGovernanceTest: base,
	}, nil
}

func (t *ChangeBaseFeeDenom) Name() string {
	return "Base Fee Denom test"
}

func (t *ChangeBaseFeeDenom) Run() error {
	fmt.Println("Running", t.Name())
	defer fmt.Println("Finished", t.Name())

	newBaseFeeDenom := big.NewInt(215)

	networkParamsResponse, err := ABICall(t.txRelayer,
		contractsapi.NetworkParams, contracts.NetworkParamsContract,
		types.ZeroAddress, "baseFeeChangeDenom")
	if err != nil {
		return err
	}

	oldBaseFeeDenom, err := common.ParseUint256orHex(&networkParamsResponse)
	if err != nil {
		return err
	}

	setNewBaseFeeDenomFn := &contractsapi.SetNewBaseFeeChangeDenomNetworkParamsFn{
		NewBaseFeeChangeDenom: newBaseFeeDenom,
	}

	proposalInput, err := setNewBaseFeeDenomFn.EncodeAbi()
	if err != nil {
		return err
	}

	proposalDescription := fmt.Sprintf("Change base fee denom")

	privKey, err := decodePrivateKey(t.config.ValidatorKeys[0])
	if err != nil {
		return err
	}

	t.executeSuccessfulProposalCycle(proposalInput, privKey, proposalDescription, "baseFeeChangeDenom", newBaseFeeDenom)

	setOldBaseFeeDenomFn := &contractsapi.SetNewBaseFeeChangeDenomNetworkParamsFn{
		NewBaseFeeChangeDenom: oldBaseFeeDenom,
	}

	proposalInput, err = setOldBaseFeeDenomFn.EncodeAbi()
	if err != nil {
		return err
	}

	t.executeSuccessfulProposalCycle(proposalInput, privKey, proposalDescription, "baseFeeChangeDenom", oldBaseFeeDenom)

	return nil
}
