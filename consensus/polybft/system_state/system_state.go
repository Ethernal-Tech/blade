package systemstate

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/state"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/contract"
)

var (
	errSendTxnUnsupported = errors.New("system state does not support send transactions")
)

type ChainType int

const (
	I2E ChainType = iota // Internal = 0
	E2I                  // External = 1
)

// SystemState is an interface to interact with the consensus system contracts in the chain
type SystemState interface {
	// GetEpoch retrieves current epoch number from the smart contract
	GetEpoch() (uint64, error)
	// GetNextCommittedIndex retrieves next committed bridge message index, based on the chain type
	GetNextCommittedIndex(chainID uint64, chainType ChainType) (uint64, error)
	// GetBridgeBatchByNumber returns bridge batch by number
	GetBridgeBatchByNumber(numberOfBatch *big.Int) (*contractsapi.SignedBridgeMessageBatch, error)
	// GetValidatorSetByNumber returns validator set by number
	GetValidatorSetByNumber(numberOfValidatorSet *big.Int) (*contractsapi.SignedValidatorSet, error)
	// GetBatchCommitCounter returns the commit counter for the batch with the given batch hash.
	// The base hash includes only messages, the source, and the destination chain, while other
	// fields are set to 0. See the buildBridgeMessage method of [BridgeManager] for a concrete
	// example.
	GetBatchCommitCounter(batchHash types.Hash) (*big.Int, error)
	// GetConfirmedRollbackedI2E returns whether the I2E rollback bridge message is committed.
	GetConfirmedRollbackedI2E(chainID uint64, id *big.Int) (bool, error)
	// GetConfirmedRollbackedE2I returns whether the E2I rollback bridge message is committed.
	GetConfirmedRollbackedE2I(chainID uint64, id *big.Int) (bool, error)
}

var _ SystemState = &SystemStateImpl{}

// SystemStateImpl is implementation of SystemState interface
type SystemStateImpl struct {
	validatorContract     *contract.Contract
	bridgeStorageContract *contract.Contract
}

// NewSystemState initializes new instance of systemState which abstracts smart contracts functions
func NewSystemState(
	valSetAddr types.Address,
	bridgeStorageAddr types.Address,
	provider contract.Provider) *SystemStateImpl {
	s := &SystemStateImpl{}
	s.validatorContract = contract.NewContract(
		ethgo.Address(valSetAddr),
		contractsapi.EpochManager.Abi, contract.WithProvider(provider),
	)
	s.bridgeStorageContract = contract.NewContract(
		ethgo.Address(bridgeStorageAddr),
		contractsapi.BridgeStorage.Abi,
		contract.WithProvider(provider),
	)

	return s
}

// GetEpoch retrieves current epoch number from the smart contract
func (s *SystemStateImpl) GetEpoch() (uint64, error) {
	rawResult, err := s.validatorContract.Call("currentEpochId", ethgo.Latest)
	if err != nil {
		return 0, err
	}

	epochNumber, isOk := rawResult["0"].(*big.Int)
	if !isOk {
		return 0, fmt.Errorf("failed to decode epoch")
	}

	return epochNumber.Uint64(), nil
}

// GetNextCommittedIndexExternal retrieves next committed external bridge message index
func (s *SystemStateImpl) GetNextCommittedIndex(chainID uint64, chainType ChainType) (uint64, error) {
	var funcName string

	switch chainType {
	case I2E:
		funcName = "lastCommittedI2E"
	case E2I:
		funcName = "lastCommittedE2I"
	default:
		return 0, fmt.Errorf("unsupported chain type: %d", chainType)
	}

	rawResult, err := s.bridgeStorageContract.Call(funcName, ethgo.Latest, new(big.Int).SetUint64(chainID))
	if err != nil {
		return 0, err
	}

	lastCommittedIndex, isOk := rawResult["0"].(*big.Int)
	if !isOk {
		return 0, fmt.Errorf("failed to decode last committed index")
	}

	return lastCommittedIndex.Uint64() + 1, nil
}

func (s *SystemStateImpl) GetBridgeBatchByNumber(batchID *big.Int) (*contractsapi.SignedBridgeMessageBatch, error) {
	funcName := "getCommittedBatch"

	getCommittedBatchFn := s.bridgeStorageContract.GetABI().GetMethod(funcName)
	if getCommittedBatchFn == nil {
		return nil, fmt.Errorf("failed to resolve %s function", funcName)
	}

	rawResult, err := s.bridgeStorageContract.CallInternal(getCommittedBatchFn, ethgo.Latest, batchID)
	if err != nil {
		return nil, err
	}

	sbmb := &contractsapi.SignedBridgeMessageBatch{}
	// Skip the first 32 bytes (most probably due to the dynamic types in the tuple/struct)
	if err := sbmb.DecodeAbi(rawResult[types.StorageSlotSize:]); err != nil {
		return nil, err
	}

	return sbmb, nil
}

func (s *SystemStateImpl) GetValidatorSetByNumber(validatorSetID *big.Int) (*contractsapi.SignedValidatorSet, error) {
	funcName := "getCommittedValidatorSet"

	getCommittedValSetFn := s.bridgeStorageContract.GetABI().GetMethod(funcName)
	if getCommittedValSetFn == nil {
		return nil, fmt.Errorf("failed to resolve %s function", funcName)
	}

	rawResult, err := s.bridgeStorageContract.CallInternal(getCommittedValSetFn, ethgo.Latest, validatorSetID)
	if err != nil {
		return nil, err
	}

	svs := &contractsapi.SignedValidatorSet{}
	// Skip the first 32 bytes (most probably due to the dynamic types in the tuple/struct)
	if err := svs.DecodeAbi(rawResult[types.StorageSlotSize:]); err != nil {
		return nil, err
	}

	return svs, nil
}

func (s *SystemStateImpl) GetBatchCommitCounter(hash types.Hash) (*big.Int, error) {
	funcName := "batchCommitCounter"

	rawResult, err := s.bridgeStorageContract.Call(funcName, ethgo.Latest, hash.Bytes())
	if err != nil {
		return nil, err
	}

	num, isOk := rawResult["0"].(*big.Int)
	if !isOk {
		return nil, fmt.Errorf("failed to decode batch commit counter")
	}

	return num, nil
}

func (s *SystemStateImpl) GetConfirmedRollbackedI2E(chainID uint64, id *big.Int) (bool, error) {
	funcName := "getConfirmedRollbackedI2E"

	rawResult, err := s.bridgeStorageContract.Call(funcName, ethgo.Latest, new(big.Int).SetUint64(chainID), id)
	if err != nil {
		return false, err
	}

	committed, isOk := rawResult["0"].(bool)
	if !isOk {
		return false, fmt.Errorf("failed to decode batch commit counter")
	}

	return committed, nil
}

func (s *SystemStateImpl) GetConfirmedRollbackedE2I(chainID uint64, id *big.Int) (bool, error) {
	funcName := "getConfirmedRollbackedE2I"

	rawResult, err := s.bridgeStorageContract.Call(funcName, ethgo.Latest, new(big.Int).SetUint64(chainID), id)
	if err != nil {
		return false, err
	}

	committed, isOk := rawResult["0"].(bool)
	if !isOk {
		return false, fmt.Errorf("failed to decode batch commit counter")
	}

	return committed, nil
}

var _ contract.Provider = &stateProvider{}

type stateProvider struct {
	transition *state.Transition
}

// NewStateProvider initializes EVM against given state and chain config and returns stateProvider instance
// which is an abstraction for smart contract calls
func NewStateProvider(transition *state.Transition) contract.Provider {
	return &stateProvider{transition: transition}
}

// Call implements the contract.Provider interface to make contract calls directly to the state
func (s *stateProvider) Call(addr ethgo.Address, input []byte, opts *contract.CallOpts) ([]byte, error) {
	caller := contracts.SystemCaller
	if opts != nil && opts.From != ethgo.Address(types.ZeroAddress) {
		caller = types.Address(opts.From)
	}

	result := s.transition.Call2(
		caller,
		types.Address(addr),
		input,
		big.NewInt(0),
		10000000,
	)
	if result.Failed() {
		return nil, result.Err
	}

	return result.ReturnValue, nil
}

// Txn is part of the contract.Provider interface to make Ethereum transactions. We disable this function
// since the system state does not make any transaction
func (s *stateProvider) Txn(_ ethgo.Address, _ ethgo.Key, _ []byte) (contract.Txn, error) {
	return nil, errSendTxnUnsupported
}
