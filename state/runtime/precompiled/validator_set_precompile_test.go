package precompiled

import (
	"errors"
	"math/big"
	"testing"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/validator"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/state/runtime"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_ValidatorSetPrecompile_gas(t *testing.T) {
	assert.Equal(t, uint64(240000), (&validatorSetPrecompile{}).gas(nil, types.Address{}, nil))
}

func Test_ValidatorSetPrecompile_run_BackendNotSet(t *testing.T) {
	addr := types.StringToAddress("aaff")
	host := newDummyHost(t)
	host.context = &runtime.TxContext{
		Number: 100,
	}

	p := &validatorSetPrecompile{}
	_, err := p.run(common.PadLeftOrTrim(addr.Bytes(), 32), types.Address{}, host)

	assert.ErrorIs(t, err, errValidatorSetPrecompileNotEnabled)
}

func Test_ValidatorSetPrecompile_run_GetValidatorsForBlockError(t *testing.T) {
	desiredErr := errors.New("aaabbb")
	addr := types.StringToAddress("aaff")
	host := newDummyHost(t)
	host.context = &runtime.TxContext{
		Number:     100,
		NonPayable: true,
	}
	backendMock := &validatorSetBackendMock{}

	backendMock.On("GetValidatorsForBlock", uint64(host.context.Number)).Return((validator.AccountSet)(nil), desiredErr)

	p := &validatorSetPrecompile{
		backend: backendMock,
	}
	_, err := p.run(common.PadLeftOrTrim(addr.Bytes(), 32), types.Address{}, host)

	assert.ErrorIs(t, err, desiredErr)
}

func Test_ValidatorSetPrecompile_run_IsValidator(t *testing.T) {
	accounts := getDummyAccountSet()
	addrBad := types.StringToAddress("1")
	host := newDummyHost(t)
	host.context = &runtime.TxContext{
		Number:     100,
		NonPayable: false,
	}
	backendMock := &validatorSetBackendMock{}

	backendMock.On("GetValidatorsForBlock", uint64(host.context.Number-1)).Return(accounts, error(nil))

	p := &validatorSetPrecompile{
		backend: backendMock,
	}

	for _, x := range accounts {
		v, err := p.run(common.PadLeftOrTrim(x.Address[:], 32), types.Address{}, host)
		require.NoError(t, err)
		assert.Equal(t, abiBoolTrue, v)
	}

	v, err := p.run(common.PadLeftOrTrim(addrBad.Bytes(), 32), types.Address{}, host)
	require.NoError(t, err)
	assert.Equal(t, abiBoolFalse, v)
}

func Test_ValidatorSetPrecompile_run_HasQuorum(t *testing.T) {
	accounts := getDummyAccountSet()
	addrGood := []types.Address{
		accounts[0].Address,
		accounts[1].Address,
		accounts[3].Address,
	}
	addrBad1 := []types.Address{
		accounts[0].Address,
	}
	addrBad2 := []types.Address{
		accounts[0].Address,
		types.StringToAddress("0"),
		accounts[3].Address,
	}
	host := newDummyHost(t)
	host.context = &runtime.TxContext{
		Number:     200,
		NonPayable: true,
	}
	backendMock := &validatorSetBackendMock{}

	backendMock.On("GetValidatorsForBlock", uint64(host.context.Number)).Return(accounts, error(nil))
	backendMock.On("GetMaxValidatorSetSize").Return(uint64(100), error(nil)).Twice()
	backendMock.On("GetMaxValidatorSetSize").Return(uint64(0), error(nil)).Once()

	p := &validatorSetPrecompile{
		backend: backendMock,
	}

	v, err := p.run(abiEncodeAddresses(addrGood), types.Address{}, host)
	require.NoError(t, err)
	assert.Equal(t, abiBoolTrue, v)

	v, err = p.run(abiEncodeAddresses(addrBad1), types.Address{}, host)
	require.NoError(t, err)
	assert.Equal(t, abiBoolFalse, v)

	v, err = p.run(abiEncodeAddresses(addrBad2), types.Address{}, host)
	require.NoError(t, err)
	assert.Equal(t, abiBoolFalse, v)
}

func Test_abiDecodeAddresses_Error(t *testing.T) {
	dummy1 := [31]byte{}
	dummy2 := [62]byte{}
	dummy3 := [64]byte{}
	dummy4 := [96]byte{}
	dummy4[31] = 32
	dummy4[63] = 10

	_, err := abiDecodeAddresses(dummy1[:], 0)
	require.ErrorIs(t, err, runtime.ErrInvalidInputData)

	_, err = abiDecodeAddresses(dummy2[:], 0)
	require.ErrorIs(t, err, runtime.ErrInvalidInputData)

	_, err = abiDecodeAddresses(dummy3[:], 0)
	require.ErrorIs(t, err, runtime.ErrInvalidInputData)

	_, err = abiDecodeAddresses(dummy4[:], 0)
	require.ErrorIs(t, err, runtime.ErrInvalidInputData)
}

type validatorSetBackendMock struct {
	mock.Mock
}

func (m *validatorSetBackendMock) GetValidatorsForBlock(blockNumber uint64) (validator.AccountSet, error) {
	call := m.Called(blockNumber)

	return call.Get(0).(validator.AccountSet), call.Error(1)
}

func (m *validatorSetBackendMock) GetMaxValidatorSetSize() (uint64, error) {
	call := m.Called()

	return call.Get(0).(uint64), call.Error(1)
}

func getDummyAccountSet() validator.AccountSet {
	v, _ := new(big.Int).SetString("1000000000000000000000", 10)

	return validator.AccountSet{
		&validator.ValidatorMetadata{
			Address:     types.StringToAddress("0xd29f66FEd147B26925DE44Ba468670e921012B6f"),
			VotingPower: new(big.Int).Set(v),
			IsActive:    true,
		},
		&validator.ValidatorMetadata{
			Address:     types.StringToAddress("0x72aB93bbbc38E90d962dcbf41d973f8C434978e1"),
			VotingPower: new(big.Int).Set(v),
			IsActive:    true,
		},
		&validator.ValidatorMetadata{
			Address:     types.StringToAddress("0x3b5c82720835d3BAA42eB34E3e4acEE6042A830D"),
			VotingPower: new(big.Int).Set(v),
			IsActive:    true,
		},
		&validator.ValidatorMetadata{
			Address:     types.StringToAddress("0x87558A1abE10E41a328086474c93449e8A1F8f4e"),
			VotingPower: new(big.Int).Set(v),
			IsActive:    true,
		},
	}
}
