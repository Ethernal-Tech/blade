package stake

import (
	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/contract"
	"github.com/stretchr/testify/mock"
)

type ProviderMock struct {
	mock.Mock
	contract.Provider
}

func (m *ProviderMock) Call(addr ethgo.Address, input []byte, opts *contract.CallOpts) ([]byte, error) {
	args := m.Called(addr, input, opts)

	return args.Get(0).([]byte), args.Error(1)
}

func (m *ProviderMock) Txn(addr ethgo.Address, key ethgo.Key, input []byte) (contract.Txn, error) {
	args := m.Called(addr, key, input)

	return args.Get(0).(contract.Txn), args.Error(1)
}
