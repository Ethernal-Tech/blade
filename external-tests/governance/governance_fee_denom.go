package governance

import (
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
)

type ChangeBaseFeeDenom struct {
	*BaseGovernanceTest
}

func NewSprintSizeTest(cfg *GovernanceTestConfig,
	testAccountKey *crypto.ECDSAKey, client *jsonrpc.EthClient) (*ChangeBaseFeeDenom, error) {
	base, err := NewBaseGovernanceTest(cfg, testAccountKey, client)
	if err != nil {
		return nil, err
	}

	return &ChangeBaseFeeDenom{
		BaseGovernanceTest: base,
	}, nil
}

func (t *ChangeBaseFeeDenom) Name() string {
	return "Sprint Size test"
}

func (t *ChangeBaseFeeDenom) Run() error {
	fmt.Println("Running", t.Name())
	defer fmt.Println("Finished", t.Name())

	newBaseFeeDenom := big.NewInt(215)

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

	return nil
}
