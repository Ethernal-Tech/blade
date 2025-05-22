package stake

import (
	"math/big"
	"testing"

	"github.com/0xPolygon/polygon-edge/bls"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/blockchain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/validator"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/abi"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStakeManager_UpdateValidatorSet(t *testing.T) {
	const num = 5

	var (
		epoch               = uint64(1)
		maxValidatorSetSize = uint64(10)
	)

	blsKeys, err := bls.CreateRandomBlsKeys(num)
	require.NoError(t, err)

	abiType := abi.MustNewType("tuple(address addr,uint256[4] blsKey,uint256 stake)[]")

	type ActiveValidator struct {
		Address types.Address `abi:"addr"`
		BlsKey  [4]*big.Int   `abi:"blsKey"`
		Stake   *big.Int      `abi:"stake"`
	}

	activeValidators := make([]ActiveValidator, num)
	validatorsMetadata := make([]*validator.ValidatorMetadata, num)

	for i := 0; i < num; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		addr := key.Address()
		activeValidators[i] = ActiveValidator{
			Address: addr,
			BlsKey:  blsKeys[i].PublicKey().ToBigInt(),
			Stake:   big.NewInt(10),
		}

		validatorsMetadata[i] = &validator.ValidatorMetadata{
			Address:     addr,
			BlsKey:      blsKeys[i].PublicKey(),
			VotingPower: big.NewInt(10),
			IsActive:    true,
		}
	}

	enc, err := abiType.Encode(activeValidators)
	require.NoError(t, err)

	offset := make([]byte, 32, 32)
	offset[31] = 0x20

	enc = append(offset, enc...)

	providerMock := new(ProviderMock)
	providerMock.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(enc, nil)

	bcMock := new(blockchain.BlockchainMock)
	bcMock.On("CurrentHeader").Return(&types.Header{Number: 0}, true)
	bcMock.On("GetStateProviderForBlock", mock.Anything).Return(providerMock, nil)

	stakeManager, err := newStakeManager(
		hclog.NewNullLogger(),
		types.StringToAddress("0x0001"),
		bcMock,
		nil,
	)
	require.NoError(t, err)

	accountSet := validator.AccountSet(validatorsMetadata)

	t.Run("UpdateValidatorSet - only update", func(t *testing.T) {
		accSet := accountSet.Copy()
		validatorToUpdate := accSet[0]
		validatorToUpdate.VotingPower = big.NewInt(11)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch, maxValidatorSetSize,
			accSet)
		require.NoError(t, err)

		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 1)
		require.Len(t, updateDelta.Removed, 0)
		require.Equal(t, updateDelta.Updated[0].Address, validatorToUpdate.Address)
	})

	t.Run("UpdateValidatorSet - one unstake", func(t *testing.T) {
		accSet := accountSet.Copy()

		added := []*validator.ValidatorMetadata{
			{
				Address:     types.StringToAddress("0x0002"),
				BlsKey:      blsKeys[1].PublicKey(),
				VotingPower: big.NewInt(10),
				IsActive:    true,
			},
		}

		accSet, err := accSet.ApplyDelta(&validator.ValidatorSetDelta{
			Added: added,
		})
		require.NoError(t, err)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch, maxValidatorSetSize,
			accSet)

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 0)
		require.Len(t, updateDelta.Removed, 1)
	})

	t.Run("UpdateValidatorSet - one new validator", func(t *testing.T) {
		accSet := accountSet.Copy()[1:]

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch, maxValidatorSetSize,
			accSet)

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 1)
		require.Len(t, updateDelta.Updated, 0)
		require.Len(t, updateDelta.Removed, 0)
		require.Equal(t, accountSet[0].Address, updateDelta.Added[0].Address)
		require.Equal(t, accountSet[0].VotingPower, updateDelta.Added[0].VotingPower)
	})
	t.Run("UpdateValidatorSet - remove some stake", func(t *testing.T) {
		accSet := accountSet.Copy()
		validatorToUpdate := accSet[2]
		validatorToUpdate.VotingPower = big.NewInt(15)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch+3, maxValidatorSetSize,
			accSet)

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 1)
		require.Len(t, updateDelta.Removed, 0)
		require.Equal(t, updateDelta.Updated[0].Address, validatorToUpdate.Address)
	})
	t.Run("UpdateValidatorSet - remove validator", func(t *testing.T) {
		accSet := accountSet.Copy()
		accSet, err = accSet.ApplyDelta(&validator.ValidatorSetDelta{
			Added: []*validator.ValidatorMetadata{
				{
					Address:     types.StringToAddress("0x0002"),
					BlsKey:      blsKeys[1].PublicKey(),
					VotingPower: big.NewInt(10),
					IsActive:    true,
				},
			},
		})
		require.NoError(t, err)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch+4, maxValidatorSetSize,
			accSet)

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 0)
		require.Len(t, updateDelta.Removed, 1)
	})
}

func TestStakeCounter_ShouldBeDeterministic(t *testing.T) {
	t.Parallel()

	const timesToExecute = 100

	stakes := [][]uint64{
		{103, 102, 101, 51, 50, 30, 10},
		{100, 100, 100, 50, 50, 30, 10},
		{103, 102, 101, 51, 50, 30, 10},
		{100, 100, 100, 50, 50, 30, 10},
	}
	maxValidatorSetSizes := []int{1000, 1000, 5, 6}

	for ind, stake := range stakes {
		maxValidatorSetSize := maxValidatorSetSizes[ind]

		aliases := []string{"A", "B", "C", "D", "E", "F", "G"}
		validators := validator.NewTestValidatorsWithAliases(t, aliases, stake)

		test := func() []*validator.ValidatorMetadata {
			stakeCounter := validator.NewValidatorStakeMap(validators.GetPublicIdentities("A", "B", "C", "D", "E"))

			return stakeCounter.GetSorted(maxValidatorSetSize)
		}

		initialSlice := test()

		// stake counter and stake map should always be deterministic
		for i := 0; i < timesToExecute; i++ {
			currentSlice := test()

			require.Len(t, currentSlice, len(initialSlice))

			for i, si := range currentSlice {
				initialSi := initialSlice[i]
				require.Equal(t, si.Address, initialSi.Address)
				require.Equal(t, si.VotingPower.Uint64(), initialSi.VotingPower.Uint64())
			}
		}
	}
}

var _ txrelayer.TxRelayer = (*dummyStakeTxRelayer)(nil)

type dummyStakeTxRelayer struct {
	mock.Mock
	callback func() *validator.ValidatorMetadata
	t        *testing.T
}

func newDummyStakeTxRelayer(t *testing.T, callback func() *validator.ValidatorMetadata) *dummyStakeTxRelayer {
	t.Helper()

	return &dummyStakeTxRelayer{
		t:        t,
		callback: callback,
	}
}

func (d *dummyStakeTxRelayer) Call(from types.Address, to types.Address, input []byte) (string, error) {
	args := d.Called(from, to, input)

	if d.callback != nil {
		validatorMetaData := d.callback()
		encoded, err := validatorTypeABI.Encode(map[string]interface{}{
			"blsKey":        validatorMetaData.BlsKey.ToBigInt(),
			"stake":         validatorMetaData.VotingPower,
			"isWhitelisted": true,
			"isActive":      true,
		})

		require.NoError(d.t, err)

		return hex.EncodeToHex(encoded), nil
	}

	return args.String(0), args.Error(1)
}

func (d *dummyStakeTxRelayer) SendTransaction(transaction *types.Transaction, key crypto.Key) (*ethgo.Receipt, error) {
	args := d.Called(transaction, key)

	return args.Get(0).(*ethgo.Receipt), args.Error(1)
}

// SendTransactionLocal sends non-signed transaction (this is only for testing purposes)
func (d *dummyStakeTxRelayer) SendTransactionLocal(txn *types.Transaction) (*ethgo.Receipt, error) {
	args := d.Called(txn)

	return args.Get(0).(*ethgo.Receipt), args.Error(1)
}

func (d *dummyStakeTxRelayer) Client() *jsonrpc.EthClient {
	return nil
}

func (d *dummyStakeTxRelayer) GetTxnHashes() []types.Hash {
	return nil
}
