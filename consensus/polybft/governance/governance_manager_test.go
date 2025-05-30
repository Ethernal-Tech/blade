package governance

import (
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/chain"
	polychain "github.com/0xPolygon/polygon-edge/consensus/polybft/blockchain"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/oracle"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/state"
	systemstate "github.com/0xPolygon/polygon-edge/consensus/polybft/system_state"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/validator"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/forkmanager"
	"github.com/0xPolygon/polygon-edge/helper/common"
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

	governanceManager := &governanceManager{
		state:      nil,
		logger:     hclog.NewNullLogger(),
		blockchain: blockchain,
	}

	// no initial config was saved, so we expect an error
	require.ErrorIs(t, governanceManager.PostEpoch(&oracle.PostEpochRequest{
		NewEpochID:        2,
		FirstBlockOfEpoch: 21,
		Forks:             &chain.Forks{chain.Governance: chain.NewFork(0)},
	}),
		errStateNil)

	params := &chain.Params{
		BaseFeeChangeDenom: 8,
		Engine:             map[string]interface{}{polycfg.ConsensusName: createTestPolybftConfig()},
	}

	governanceManager.state = params

	// PostEpoch will now update config with new epoch reward value
	require.NoError(t, governanceManager.PostEpoch(&oracle.PostEpochRequest{
		NewEpochID:        2,
		FirstBlockOfEpoch: 21,
		Forks:             &chain.Forks{chain.Governance: chain.NewFork(0)},
	}))

	updatedConfig := governanceManager.state
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

		providerMock := new(systemstate.ProviderMock)
		providerMock.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(enc, nil).Times(2)

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

func createTestPolybftConfig() *polycfg.PolyBFT {
	return &polycfg.PolyBFT{
		InitialValidatorSet: []*validator.GenesisValidator{
			{
				Address: types.BytesToAddress([]byte{0, 1, 2}),
				Stake:   big.NewInt(100),
			},
			{
				Address: types.BytesToAddress([]byte{3, 4, 5}),
				Stake:   big.NewInt(100),
			},
			{
				Address: types.BytesToAddress([]byte{6, 7, 8}),
				Stake:   big.NewInt(100),
			},
			{
				Address: types.BytesToAddress([]byte{9, 10, 11}),
				Stake:   big.NewInt(100),
			},
		},
		Bridge: map[uint64]*polycfg.Bridge{0: {
			ExternalGatewayAddr:                  types.StringToAddress("0xGatewayAddr"),
			ExternalERC20PredicateAddr:           types.StringToAddress("0xRootERC20PredicateAddr"),
			ExternalMintableERC20PredicateAddr:   types.StringToAddress("0xChildMintableERC20PredicateAddr"),
			ExternalERC721PredicateAddr:          types.StringToAddress("0xRootERC721PredicateAddr"),
			ExternalMintableERC721PredicateAddr:  types.StringToAddress("0xChildMintableERC721PredicateAddr"),
			ExternalERC1155PredicateAddr:         types.StringToAddress("0xRootERC1155PredicateAddr"),
			ExternalMintableERC1155PredicateAddr: types.StringToAddress("0xChildMintableERC1155PredicateAddr"),
			ExternalERC20Addr:                    types.StringToAddress("0xChildERC20Addr"),
			ExternalERC721Addr:                   types.StringToAddress("0xChildERC721Addr"),
			ExternalERC1155Addr:                  types.StringToAddress("0xChildERC1155Addr"),
			BLSAddress:                           types.StringToAddress("0xBLSAddress"),
			BN256G2Address:                       types.StringToAddress("0xBN256G2Address"),
			JSONRPCEndpoint:                      "http://mumbai-rpc.com",
			EventTrackerStartBlocks: map[types.Address]uint64{
				types.StringToAddress("SomeRootAddress"): 365_000,
			}},
		},
		EpochSize:           10,
		EpochReward:         1000,
		SprintSize:          5,
		BlockTime:           common.Duration{Duration: 2 * time.Second},
		MinValidatorSetSize: 4,
		MaxValidatorSetSize: 100,
		CheckpointInterval:  900,
		BlockTimeDrift:      10,
		Governance:          types.ZeroAddress,
		NativeTokenConfig: &polycfg.Token{
			Name:     "Polygon_MATIC",
			Symbol:   "MATIC",
			Decimals: 18,
		},
		InitialTrieRoot:      types.ZeroHash,
		WithdrawalWaitPeriod: 1,
		RewardConfig: &polycfg.Rewards{
			TokenAddress:  types.StringToAddress("0xRewardTokenAddr"),
			WalletAddress: types.StringToAddress("0xRewardWalletAddr"),
			WalletAmount:  big.NewInt(1_000_000),
		},
		GovernanceConfig: &polycfg.Governance{
			VotingDelay:              big.NewInt(1000),
			VotingPeriod:             big.NewInt(10_0000),
			ProposalThreshold:        big.NewInt(1000),
			ProposalQuorumPercentage: 67,
			ChildGovernorAddr:        contracts.ChildGovernorContract,
			ChildTimelockAddr:        contracts.ChildTimelockContract,
			NetworkParamsAddr:        contracts.NetworkParamsContract,
			ForkParamsAddr:           contracts.ForkParamsContract,
		},
	}
}
