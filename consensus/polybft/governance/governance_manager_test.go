package governance

import (
	"math/big"
	"testing"

	"github.com/0xPolygon/polygon-edge/chain"
	polychain "github.com/0xPolygon/polygon-edge/consensus/polybft/blockchain"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/oracle"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/state"
	systemstate "github.com/0xPolygon/polygon-edge/consensus/polybft/system_state"
	"github.com/0xPolygon/polygon-edge/forkmanager"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo/abi"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var networkParamsAbiType = abi.MustNewType(`tuple(uint256 checkpointBlockInterval, uint256 epochSize, uint256 epochReward, uint256 sprintSize, 
	uint256 minValidatorSetSize, uint256 maxValidatorSetSize, uint256 withdrawalWaitPeriod, uint256 blockTime, 
	uint256 blockTimeDrift, uint256 votingDelay, uint256 votingPeriod, uint256 proposalThreshold, uint256 baseFeeChangeDenom)`)

type networkParamsTest struct {
	CheckpointBlockInterval *big.Int `abi:"checkpointBlockInterval"`
	EpochSize               *big.Int `abi:"epochSize"`
	EpochReward             *big.Int `abi:"epochReward"`
	SprintSize              *big.Int `abi:"sprintSize"`
	MinValidatorSetSize     *big.Int `abi:"minValidatorSetSize"`
	MaxValidatorSetSize     *big.Int `abi:"maxValidatorSetSize"`
	WithdrawalWaitPeriod    *big.Int `abi:"withdrawalWaitPeriod"`
	BlockTime               *big.Int `abi:"blockTime"`
	BlockTimeDrift          *big.Int `abi:"blockTimeDrift"`
	VotingDelay             *big.Int `abi:"votingDelay"`
	VotingPeriod            *big.Int `abi:"votingPeriod"`
	ProposalThreshold       *big.Int `abi:"proposalThreshold"`
	BaseFeeChangeDenom      *big.Int `abi:"baseFeeChangeDenom"`
}

func (n *networkParamsTest) Encode() ([]byte, error) {
	return networkParamsAbiType.Encode(n)
}

var forkParamsTestAbiType = abi.MustNewType(`tuple(address feature, uint256 block)[]`)

type forkParamTest struct {
	Feature     types.Hash `abi:"feature"`
	BlockNumber *big.Int   `abi:"block"`
}

type forkParamsTest []*forkParamTest

func (f forkParamsTest) Encode() ([]byte, error) {
	encoded, err := forkParamsTestAbiType.Encode(f)
	if err != nil {
		return nil, err
	}

	offset := make([]byte, 32, 32)
	offset[31] = 0x20

	encoded = append(offset, encoded...)

	return encoded, nil
}

func TestGovernanceManager_PostEpoch(t *testing.T) {
	t.Parallel()

	networkParams := &networkParamsTest{
		CheckpointBlockInterval: big.NewInt(100),
		EpochSize:               big.NewInt(10),
		EpochReward:             big.NewInt(10000),
		SprintSize:              big.NewInt(5),
		MinValidatorSetSize:     big.NewInt(10),
		MaxValidatorSetSize:     big.NewInt(100),
		WithdrawalWaitPeriod:    big.NewInt(100),
		BlockTime:               big.NewInt(2),
		BlockTimeDrift:          big.NewInt(1),
		VotingDelay:             big.NewInt(10),
		VotingPeriod:            big.NewInt(100),
		ProposalThreshold:       big.NewInt(1000),
		BaseFeeChangeDenom:      big.NewInt(100),
	}

	encoded, err := networkParams.Encode()
	require.NoError(t, err)

	providerMock := new(systemstate.ProviderMock)
	providerMock.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(encoded, nil)

	blockchain := new(polychain.BlockchainMock)
	blockchain.On("CurrentHeader").Return(&types.Header{
		Number: 0,
	})

	blockchain.On("GetStateProviderForBlock", mock.Anything).Return(providerMock, nil)

	state := newTestState(t)
	governanceManager := &governanceManager{
		state:      state,
		logger:     hclog.NewNullLogger(),
		blockchain: blockchain,
	}

	// no initial config was saved, so we expect an error
	require.ErrorIs(t, governanceManager.PostEpoch(&oracle.PostEpochRequest{
		NewEpochID:        2,
		FirstBlockOfEpoch: 21,
		Forks:             &chain.Forks{chain.Governance: chain.NewFork(0)},
	}),
		errClientConfigNotFound)

	params := &chain.Params{
		BaseFeeChangeDenom: 8,
		Engine:             map[string]interface{}{polycfg.ConsensusName: createTestPolybftConfig()},
	}

	// insert initial config
	require.NoError(t, state.insertClientConfig(params, nil))

	// PostEpoch will now update config with new epoch reward value
	require.NoError(t, governanceManager.PostEpoch(&oracle.PostEpochRequest{
		NewEpochID:        2,
		FirstBlockOfEpoch: 21,
		Forks:             &chain.Forks{chain.Governance: chain.NewFork(0)},
	}))

	updatedConfig, err := state.getClientConfig(nil)
	require.NoError(t, err)
	require.Equal(t, networkParams.BaseFeeChangeDenom.Uint64(), updatedConfig.BaseFeeChangeDenom)

	pbftConfig, err := polycfg.GetPolyBFTConfig(updatedConfig)
	require.NoError(t, err)

	require.Equal(t, networkParams.EpochReward.Uint64(), pbftConfig.EpochReward)
}

func TestGovernanceManager_PostBlock(t *testing.T) {
	t.Parallel()

	genesisPolybftConfig := createTestPolybftConfig()

	t.Run("Has no new forks", func(t *testing.T) {
		t.Parallel()

		// no governance events in receipts
		req := &oracle.PostBlockRequest{
			FullBlock: &types.FullBlock{Block: &types.Block{Header: &types.Header{Number: 5}},
				Receipts: []*types.Receipt{},
			},
			Epoch: 1,
			Forks: &chain.Forks{chain.Governance: chain.NewFork(0)},
		}

		var forkParamsTest forkParamsTest
		enc, err := forkParamsTest.Encode()
		require.NoError(t, err)

		providerMock := new(systemstate.ProviderMock)
		providerMock.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(enc, nil)

		blockchainMock := new(polychain.BlockchainMock)
		blockchainMock.On("CurrentHeader").Return(&types.Header{
			Number: 0,
		})
		blockchainMock.On("GetStateProviderForBlock", mock.Anything).Return(providerMock, nil)

		chainParams := &chain.Params{Engine: map[string]interface{}{polycfg.ConsensusName: genesisPolybftConfig}}
		gm, err := NewGovernanceManager(chainParams,
			hclog.NewNullLogger(), state.NewTestState(t), blockchainMock, nil)
		require.NoError(t, err)

		require.NoError(t, gm.PostBlock(req))

		governanceManager := gm.(*governanceManager)
		// 13 from chain.AllForksEnabled
		require.Equal(t, 13, len(governanceManager.allForksHashes))
	})

	t.Run("Has new fork", func(t *testing.T) {
		t.Parallel()

		var (
			newForkHash  = types.StringToHash("0xNewForkHash")
			newForkBlock = big.NewInt(5)
			newForkName  = "newFork"
		)

		req := &oracle.PostBlockRequest{
			FullBlock: &types.FullBlock{Block: &types.Block{Header: &types.Header{Number: 5}}},
			Epoch:     1,
			Forks:     &chain.Forks{chain.Governance: chain.NewFork(0)},
		}

		var forkParamsTest forkParamsTest
		enc, err := forkParamsTest.Encode()
		require.NoError(t, err)

		providerMock := new(systemstate.StateProviderMock)
		providerMock.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(enc, nil).Once()

		blockchainMock := new(polychain.BlockchainMock)
		blockchainMock.On("CurrentHeader").Return(&types.Header{
			Number: 0,
		})
		blockchainMock.On("GetStateProviderForBlock", mock.Anything).Return(providerMock, nil)

		chainParams := &chain.Params{Engine: map[string]interface{}{polycfg.ConsensusName: genesisPolybftConfig}}
		gm, err := NewGovernanceManager(chainParams,
			hclog.NewNullLogger(), state.NewTestState(t), blockchainMock, nil)
		require.NoError(t, err)

		governanceManager := gm.(*governanceManager)

		forkParamsTest = append(forkParamsTest, &forkParamTest{
			Feature:     newForkHash,
			BlockNumber: newForkBlock,
		})
		enc, err = forkParamsTest.Encode()
		require.NoError(t, err)

		providerMock.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(enc, nil).Once()
		// this cheats that we have this fork in code
		governanceManager.allForksHashes[newForkHash] = newForkName

		// new fork should not be registered and enabled before PostBlock
		require.False(t, forkmanager.GetInstance().IsForkEnabled(newForkName, newForkBlock.Uint64()))

		require.Equal(t, 14, len(governanceManager.allForksHashes))
		require.NoError(t, governanceManager.PostBlock(req))

		// new fork should be registered and enabled before PostBlock
		require.True(t, forkmanager.GetInstance().IsForkEnabled(newForkName, newForkBlock.Uint64()))
	})
}
