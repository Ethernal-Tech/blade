package precompiled

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/validator"
	"github.com/0xPolygon/polygon-edge/state/runtime"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
)

const (
	addrOffset                   = 32 - types.AddressLength
	defaultMaxHasQuorumAddresses = 100
)

var (
	errValidatorSetPrecompileNotEnabled = errors.New("validator set precompile is not enabled")
)

// ValidatorSetPrecompileBackend is an interface defining the contract for a precompile backend
// responsible for retrieving validators (current account set) for a specific block number
type ValidatorSetPrecompileBackend interface {
	GetValidatorsForBlock(blockNumber uint64) (validator.AccountSet, error)
	GetMaxValidatorSetSize() (uint64, error)
}

// validatorSetPrecompile is a concrete implementation of the contract interface.
// The struct implements two functionalities through the `run` method:
// - isValidator(address addr) bool: Returns true if addr is the address of a validator.
// - hasQuorum(address[] addrs) bool: Returns true if the array of validators is sufficient to constitute a quorum
// It encapsulates a backend that provides the functionality to retrieve validators for a specific block number
type validatorSetPrecompile struct {
	backend ValidatorSetPrecompileBackend
}

// gas returns the gas required to execute the pre-compiled contract
func (c *validatorSetPrecompile) gas(_ []byte, _ types.Address, config *chain.ForksInTime) uint64 {
	return 240000
}

// Run runs the precompiled contract with the given input.
// There are two functions:
// isValidator(address addr) bool
// hasQuorum(address[] addrs) bool
// Input must be ABI encoded: address or (address[])
// Output could be an error or ABI encoded "bool" value
func (c *validatorSetPrecompile) run(input []byte, caller types.Address, host runtime.Host) ([]byte, error) {
	// if its payable tx we need to look for validator in previous block
	blockNumber := uint64(host.GetTxContext().Number)
	if !host.GetTxContext().NonPayable {
		blockNumber--
	}

	// isValidator case
	if len(input) == 32 {
		validatorSet, err := createValidatorSet(blockNumber, c.backend) // we are calling validators for previous block
		if err != nil {
			return nil, err
		}

		addr := types.BytesToAddress(input[0:32])

		if validatorSet.Includes(addr) {
			return abiBoolTrue, nil
		}

		return abiBoolFalse, nil
	}

	maxAddressesCount, err := c.backend.GetMaxValidatorSetSize()
	if err != nil {
		return nil, err
	}

	if maxAddressesCount == 0 {
		maxAddressesCount = defaultMaxHasQuorumAddresses
	}

	addresses, err := abiDecodeAddresses(input, uint32(maxAddressesCount))
	if err != nil {
		return nil, err
	}

	validatorSet, err := createValidatorSet(blockNumber, c.backend)
	if err != nil {
		return nil, err
	}

	signers := make(map[types.Address]struct{}, len(addresses))
	for _, x := range addresses {
		signers[x] = struct{}{}
	}

	if validatorSet.HasQuorum(blockNumber, signers) {
		return abiBoolTrue, nil
	}

	return abiBoolFalse, nil
}

func createValidatorSet(blockNumber uint64, backend ValidatorSetPrecompileBackend) (validator.ValidatorSet, error) {
	if backend == nil {
		return nil, errValidatorSetPrecompileNotEnabled
	}

	accounts, err := backend.GetValidatorsForBlock(blockNumber)
	if err != nil {
		return nil, err
	}

	return validator.NewValidatorSet(accounts, hclog.NewNullLogger()), nil
}

func abiDecodeAddresses(input []byte, maxCount uint32) ([]types.Address, error) {
	if len(input) < 64 || len(input)%32 != 0 {
		return nil, runtime.ErrInvalidInputData
	}

	// abi.encode encodes addresses[] with slice of bytes where initial 31 bytes
	// are set to 0, and 32nd is 32
	// then goes length of slice (32 bytes)
	// then each address is 32 bytes
	dummy := [32]byte{}
	dummy[31] = 32

	if !bytes.Equal(dummy[:], input[:32]) {
		return nil, runtime.ErrInvalidInputData
	}

	size := binary.BigEndian.Uint32(input[60:64])
	if uint32(len(input)) != size*32+64 {
		return nil, runtime.ErrInvalidInputData
	}
	// sanitize the input if it contains too many addresses.
	if maxCount != 0 && size > maxCount {
		return nil, runtime.ErrInvalidInputData
	}

	res := make([]types.Address, size)
	for i, offset := 0, 64; offset < len(input); i, offset = i+1, offset+32 {
		res[i] = types.Address(input[offset+addrOffset : offset+32])
	}

	return res, nil
}

func abiEncodeAddresses(addrs []types.Address) []byte {
	res := make([]byte, len(addrs)*32+64)
	res[31] = 32

	binary.BigEndian.PutUint32(res[60:64], uint32(len(addrs))) // 60 == 32 + 28

	offset := 64

	for _, addr := range addrs {
		copy(res[offset+addrOffset:offset+32], addr.Bytes())
		offset += 32
	}

	return res
}
