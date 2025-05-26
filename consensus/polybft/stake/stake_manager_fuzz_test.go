package stake

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/blockchain"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func FuzzTestStakeManagerUpdateValidatorSet(f *testing.F) {
	const num = 5

	var (
		maxValidatorSetSize = uint64(10)
	)

	type updateValidatorSetF struct {
		EpochID     uint64
		Index       uint64
		VotingPower int64
	}

	accountSet, activeValidators, err := generateValidators(num)
	require.NoError(f, err)

	enc, err := encodeValidators(activeValidators)
	require.NoError(f, err)

	providerMock := new(ProviderMock)
	providerMock.On("Call", mock.Anything, mock.Anything, mock.Anything).Return(enc, nil)

	bcMock := new(blockchain.BlockchainMock)
	bcMock.On("CurrentHeader").Return(&types.Header{Number: 0})
	bcMock.On("GetStateProviderForBlock", mock.Anything).Return(providerMock, nil)

	stakeManager, err := newStakeManager(
		hclog.NewNullLogger(),
		types.StringToAddress("0x0001"),
		bcMock,
		nil,
	)
	require.NoError(f, err)

	seeds := []updateValidatorSetF{
		{
			EpochID:     1,
			Index:       1,
			VotingPower: 30,
		},
		{
			EpochID:     2,
			Index:       4,
			VotingPower: 1,
		},
		{
			EpochID:     3,
			Index:       3,
			VotingPower: 2,
		},
	}

	for _, seed := range seeds {
		data, err := json.Marshal(seed)
		if err != nil {
			return
		}

		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		var data updateValidatorSetF
		if err := json.Unmarshal(input, &data); err != nil {
			t.Skip(err)
		}

		if err := ValidateStruct(data); err != nil {
			t.Skip(err)
		}

		if data.Index > uint64(num-1) {
			t.Skip()
		}

		fullValidatorSet := accountSet.Copy()
		validatorToUpdate := fullValidatorSet[data.Index]
		validatorToUpdate.VotingPower = big.NewInt(data.VotingPower)

		delta, err := stakeManager.UpdateValidatorSet(data.EpochID, maxValidatorSetSize,
			fullValidatorSet[data.Index:])
		require.NoError(t, err)

		require.Equal(t, len(delta.Added), int(data.Index))
		require.Equal(t, len(delta.Updated), 1)
		require.Equal(t, len(delta.Removed), 0)
	})
}

func ValidateStruct(s interface{}) (err error) {
	structType := reflect.TypeOf(s)
	if structType.Kind() != reflect.Struct {
		return errors.New("input param should be a struct")
	}

	structVal := reflect.ValueOf(s)
	fieldNum := structVal.NumField()

	for i := 0; i < fieldNum; i++ {
		field := structVal.Field(i)
		fieldName := structType.Field(i).Name

		isSet := field.IsValid() && !field.IsZero()

		if !isSet {
			err = fmt.Errorf("%w%s is not set; ", err, fieldName)
		}
	}

	return err
}
