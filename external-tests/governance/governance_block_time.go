package governance

import (
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/types"
)

type ChangeBlockTime struct {
	*BaseGovernanceTest
}

func NewBlockTimeTest(cfg *GovernanceTestConfig,
	testAccountKey *crypto.ECDSAKey, client *jsonrpc.EthClient) (*ChangeBlockTime, error) {
	base, err := NewBaseGovernanceTest(cfg, testAccountKey, client)
	if err != nil {
		return nil, err
	}

	return &ChangeBlockTime{
		BaseGovernanceTest: base,
	}, nil
}

func (t *ChangeBlockTime) Name() string {
	return "Block Time Test"
}

func (t *ChangeBlockTime) Run() error {
	fmt.Println("Running", t.Name())
	defer fmt.Println("Finished", t.Name())

	setTime := func(newBlockTime *big.Int) error {
		setNewBlockTime := contractsapi.SetNewBlockTimeNetworkParamsFn{
			NewBlockTime: newBlockTime,
		}

		proposalInput, err := setNewBlockTime.EncodeAbi()
		if err != nil {
			return err
		}

		proposalDescription := fmt.Sprintf("Change block time from 2s to %ds", newBlockTime)

		privKey, err := decodePrivateKey(t.config.ValidatorKeys[0])
		if err != nil {
			return err
		}

		executeSuccssfulProposalCycle(t.config,
			t.txrelayer, proposalInput,
			privKey, proposalDescription,
			"blockTime", newBlockTime)

		networkParamsResponse, err := ABICall(t.txrelayer,
			contractsapi.NetworkParams, contracts.NetworkParamsContract,
			types.ZeroAddress, "epochSize")
		if err != nil {
			return err
		}

		epochSize, err := common.ParseUint256orHex(&networkParamsResponse)
		if err != nil {
			return err
		}

		currentBlockNumber, err := t.txrelayer.Client().BlockNumber()
		if err != nil {
			return err
		}

		currentBlockNumber--

		endOfEpoch := (currentBlockNumber/epochSize.Uint64() + 1) * epochSize.Uint64()

		if err := waitForBlock(int64(endOfEpoch)+5, 3*time.Minute); err != nil {
			return err
		}

		blockToGet := endOfEpoch + 5
		headerOne, err := t.txrelayer.Client().GetHeaderByNumber(jsonrpc.BlockNumber(blockToGet))
		if err != nil {
			return err
		}

		headerTwo, err := t.txrelayer.Client().GetHeaderByNumber(jsonrpc.BlockNumber(blockToGet - 1))
		if err != nil {
			return err
		}

		blockTime := headerOne.Timestamp - headerTwo.Timestamp
		if blockTime < newBlockTime.Uint64() {
			return fmt.Errorf("block time didn't changed")
		}

		return nil
	}

	newBlockTime := big.NewInt(5) // 5 seconds

	if err := setTime(newBlockTime); err != nil {
		return err
	}

	// it's neccessary to set old block time on test network.
	oldBlockTime := big.NewInt(2) // 2 seconds

	if err := setTime(oldBlockTime); err != nil {
		return err
	}

	return nil
}
