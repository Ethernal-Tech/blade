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

var activeValidatorAbiType = abi.MustNewType("tuple(address addr,uint256[4] blsKey,uint256 stake)[]")

type activeValidator struct {
	Address types.Address `abi:"addr"`
	BlsKey  [4]*big.Int   `abi:"blsKey"`
	Stake   *big.Int      `abi:"stake"`
}

func (a *activeValidator) Copy() *activeValidator {
	return &activeValidator{
		Address: types.Address(a.Address[:]),
		BlsKey:  a.BlsKey,
		Stake:   new(big.Int).Set(a.Stake),
	}
}

type activeValidatorSet []*activeValidator

func (a activeValidatorSet) Copy() activeValidatorSet {
	copied := make(activeValidatorSet, len(a))
	for i, v := range a {
		copied[i] = v.Copy()
	}

	return copied
}

func addValidatorsToMock(t *testing.T, providerMock *ProviderMock, activeValidators activeValidatorSet) {
	t.Helper()

	enc, err := encodeValidators(activeValidators)
	require.NoError(t, err)

	t.Logf("active validators: %v", activeValidators)
	providerMock.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(enc, nil).Once()
}

func TestStakeManager_UpdateValidatorSet(t *testing.T) {
	const num = 5

	var (
		epoch               = uint64(1)
		maxValidatorSetSize = uint64(10)
	)

	accountSet, activeValidators, err := generateValidators(num)
	require.NoError(t, err)

	providerMock := new(ProviderMock)

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

	t.Run("UpdateValidatorSet - only update", func(t *testing.T) {
		fullValidatorSet := activeValidators.Copy()
		validatorToUpdate := fullValidatorSet[0]
		validatorToUpdate.Stake = big.NewInt(11)

		addValidatorsToMock(t, providerMock, fullValidatorSet)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch, maxValidatorSetSize,
			accountSet)
		require.NoError(t, err)

		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 1)
		require.Len(t, updateDelta.Removed, 0)
		require.Equal(t, updateDelta.Updated[0].Address, validatorToUpdate.Address)
		require.Equal(t, updateDelta.Updated[0].VotingPower.Uint64(), validatorToUpdate.Stake.Uint64())
	})

	t.Run("UpdateValidatorSet - one unstake", func(t *testing.T) {
		fullValidatorSet := activeValidators.Copy()[1:]

		addValidatorsToMock(t, providerMock, fullValidatorSet)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch+1, maxValidatorSetSize,
			accountSet)

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 0)
		require.Len(t, updateDelta.Removed, 1)
	})

	t.Run("UpdateValidatorSet - one new validator", func(t *testing.T) {
		addValidatorsToMock(t, providerMock, activeValidators)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch+2, maxValidatorSetSize,
			accountSet[1:])

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 1)
		require.Len(t, updateDelta.Updated, 0)
		require.Len(t, updateDelta.Removed, 0)
		require.Equal(t, accountSet[0].Address, updateDelta.Added[0].Address)
		require.Equal(t, accountSet[0].VotingPower, updateDelta.Added[0].VotingPower)
	})

	t.Run("UpdateValidatorSet - remove some stake", func(t *testing.T) {
		fullValidatorSet := activeValidators.Copy()
		validatorToUpdate := fullValidatorSet[2]
		validatorToUpdate.Stake = big.NewInt(5)

		addValidatorsToMock(t, providerMock, fullValidatorSet)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch+3, maxValidatorSetSize,
			accountSet)

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 1)
		require.Len(t, updateDelta.Removed, 0)
		require.Equal(t, updateDelta.Updated[0].Address, validatorToUpdate.Address)
		require.Equal(t, updateDelta.Updated[0].VotingPower.Uint64(), validatorToUpdate.Stake.Uint64())
	})

	t.Run("UpdateValidatorSet - remove entire stake", func(t *testing.T) {
		fullValidatorSet := activeValidators.Copy()
		validatorsToUpdate := fullValidatorSet[4]
		validatorsToUpdate.Stake = bigZero

		addValidatorsToMock(t, providerMock, fullValidatorSet)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch+5, maxValidatorSetSize,
			accountSet)

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 0)
		require.Len(t, updateDelta.Removed, 1)
	})

	t.Run("UpdateValidatorSet - voting power negative", func(t *testing.T) {
		fullValidatorSet := activeValidators.Copy()
		validatorsToUpdate := fullValidatorSet[4]
		validatorsToUpdate.Stake = bigZero

		addValidatorsToMock(t, providerMock, fullValidatorSet)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch+5, maxValidatorSetSize,
			accountSet)
		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 0)
		require.Len(t, updateDelta.Updated, 0)
		require.Len(t, updateDelta.Removed, 1)
	})

	t.Run("UpdateValidatorSet - max validator set size reached", func(t *testing.T) {
		// because we now have 5 validators, and the new validator has more stake
		fullValidatorSet := activeValidators.Copy()
		validatorToAdd := fullValidatorSet[0]
		validatorToAdd.Stake = big.NewInt(11)

		addValidatorsToMock(t, providerMock, fullValidatorSet)

		updateDelta, err := stakeManager.UpdateValidatorSet(epoch+6, 4,
			accountSet[1:])

		require.NoError(t, err)
		require.Len(t, updateDelta.Added, 1)
		require.Len(t, updateDelta.Updated, 0)
		require.Len(t, updateDelta.Removed, 1)
		require.Equal(t, validatorToAdd.Address, updateDelta.Added[0].Address)
		require.Equal(t, validatorToAdd.Stake.Uint64(), updateDelta.Added[0].VotingPower.Uint64())
	})
}

func encodeValidators(activeValidator activeValidatorSet) ([]byte, error) {
	enc, err := activeValidatorAbiType.Encode(activeValidator)
	if err != nil {
		return nil, err
	}

	offset := make([]byte, 32, 32)
	offset[31] = 0x20

	enc = append(offset, enc...)

	return enc, nil
}

func generateValidators(num int) (validator.AccountSet, activeValidatorSet, error) {
	activeValidators := make([]*activeValidator, num)
	validatorsMetadata := make([]*validator.ValidatorMetadata, num)

	blsKeys, err := bls.CreateRandomBlsKeys(num)
	if err != nil {
		return nil, nil, err
	}

	for i := 0; i < num; i++ {
		key, err := crypto.GenerateECDSAKey()
		if err != nil {
			return nil, nil, err
		}

		addr := key.Address()
		activeValidators[i] = &activeValidator{
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

	return validator.AccountSet(validatorsMetadata), activeValidators, nil
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
