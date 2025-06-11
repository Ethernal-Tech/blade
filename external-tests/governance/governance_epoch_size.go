package governance

import (
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus/polybft"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/types"
)

type ChangeEpochSize struct {
	*BaseGovernanceTest
}

func NewEpochSizeTest(cfg *GovernanceTestConfig,
	client *jsonrpc.EthClient) (*ChangeEpochSize, error) {
	base, err := NewBaseGovernanceTest(cfg, client)
	if err != nil {
		return nil, err
	}

	return &ChangeEpochSize{
		BaseGovernanceTest: base,
	}, nil
}

func (t *ChangeEpochSize) Name() string {
	return "Epoch Size test"
}

func (t *ChangeEpochSize) Run() error {
	fmt.Println("Running", t.Name())
	defer fmt.Println("Finished", t.Name())

	networkParamsResponse, err := ABICall(t.txRelayer,
		contractsapi.NetworkParams, contracts.NetworkParamsContract,
		types.ZeroAddress, "epochSize")
	if err != nil {
		return err
	}

	oldEpochSize, err := common.ParseUint256orHex(&networkParamsResponse)
	if err != nil {
		return err
	}

	var (
		newEpochSize = new(big.Int).SetUint64(t.BaseGovernanceTest.config.EpochSize)
	)

	privKey, err := decodePrivateKey(t.config.ValidatorKeys[0])
	if err != nil {
		return err
	}

	setEpochSize := func(newEpochSizeBig, oldEpochSizeBig *big.Int) error {
		oldEpochSize := oldEpochSizeBig.Uint64()
		newEpochSize := newEpochSizeBig.Uint64()

		// propose a new epoch size
		setNewEpochSizeFn := &contractsapi.SetNewEpochSizeNetworkParamsFn{
			NewEpochSize: newEpochSizeBig,
		}

		proposalInput, err := setNewEpochSizeFn.EncodeAbi()
		if err != nil {
			return err
		}

		proposalDescription := fmt.Sprintf("Change epoch size from %d to %d",
			oldEpochSize, newEpochSize)

		t.executeSuccessfulProposalCycle(proposalInput,
			privKey,
			proposalDescription,
			"epochSize",
			newEpochSizeBig)

		currentBlockNumber, err := t.txRelayer.Client().BlockNumber()
		if err != nil {
			return err
		}

		endOfPreviousEpoch := (currentBlockNumber/oldEpochSize + 1) * oldEpochSize
		endOfNewEpoch := endOfPreviousEpoch + newEpochSize

		if err := waitForBlock(endOfNewEpoch, 3*time.Minute, t.txRelayer); err != nil {
			return err
		}

		block, err := t.txRelayer.Client().GetBlockByNumber(
			jsonrpc.BlockNumber(endOfPreviousEpoch), false)
		if err != nil {
			return err
		}

		extra, err := polybft.GetIbftExtra(block.Header.ExtraData)
		if err != nil {
			return err
		}

		oldEpoch := extra.Checkpoint.EpochNumber

		block, err = t.txRelayer.Client().GetBlockByNumber(
			jsonrpc.BlockNumber(endOfNewEpoch), false)
		if err != nil {
			return err
		}

		extra, err = polybft.GetIbftExtra(block.Header.ExtraData)
		if err != nil {
			return err
		}

		newEpoch := extra.Checkpoint.EpochNumber

		if newEpoch != oldEpoch+1 {
			return fmt.Errorf("next epoch is bigger than 1")
		}

		if endOfNewEpoch-endOfPreviousEpoch != newEpochSize {
			return fmt.Errorf("epoch size didn't changed")
		}

		return nil
	}

	if err := setEpochSize(newEpochSize, oldEpochSize); err != nil {
		return err
	}

	if err := setEpochSize(oldEpochSize, newEpochSize); err != nil {
		return err
	}

	return nil
}
