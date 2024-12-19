package stake

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/bls"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/bitmap"
	polychain "github.com/0xPolygon/polygon-edge/consensus/polybft/blockchain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/oracle"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/state"
	polytypes "github.com/0xPolygon/polygon-edge/consensus/polybft/types"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/validator"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/abi"
	"github.com/Ethernal-Tech/ethgo/contract"
	"github.com/hashicorp/go-hclog"
	bolt "go.etcd.io/bbolt"
)

var (
	bigZero          = big.NewInt(0)
	validatorTypeABI = abi.MustNewType("tuple(uint256[4] blsKey," +
		"uint256 stake, bool isWhitelisted, bool isActive)")
	errUnknownStakeManagerEvent = errors.New("unknown event from stake manager contract")
	// error returned if full validator set does not exists in db
	errNoFullValidatorSet = errors.New("full validator set not in db")
)

// StakeManager interface provides functions for handling stake change of validators
// and updating validator set based on changed stake
type StakeManager interface {
	state.EventSubscriber
	oracle.ReadOnlyOracle
	UpdateValidatorSet(epoch uint64, maxValidatorSetSize uint64,
		currentValidatorSet validator.AccountSet) (*validator.ValidatorSetDelta, error)
}

var _ StakeManager = (*DummyStakeManager)(nil)

// DummyStakeManager is a dummy implementation of StakeManager interface
// used only for unit testing
type DummyStakeManager struct{}

func (d *DummyStakeManager) PostBlock(req *oracle.PostBlockRequest) error { return nil }
func (d *DummyStakeManager) PostEpoch(req *oracle.PostEpochRequest) error { return nil }
func (d *DummyStakeManager) GetTransactions(blockInfo oracle.NewBlockInfo) ([]*types.Transaction, error) {
	return nil, nil
}
func (d *DummyStakeManager) VerifyTransactions(blockInfo oracle.NewBlockInfo, txs []*types.Transaction) error {
	return nil
}
func (d *DummyStakeManager) Close() {}

func (d *DummyStakeManager) UpdateValidatorSet(epoch uint64, maxValidatorSetSize uint64,
	currentValidatorSet validator.AccountSet) (*validator.ValidatorSetDelta, error) {
	return &validator.ValidatorSetDelta{}, nil
}

// EventSubscriber implementation
func (d *DummyStakeManager) GetLogFilters() map[types.Address][]types.Hash {
	return make(map[types.Address][]types.Hash)
}

func (d *DummyStakeManager) ProcessLog(header *types.Header, log *ethgo.Log, dbTx *bolt.Tx) error {
	return nil
}

var _ StakeManager = (*stakeManager)(nil)

// stakeManager saves transfer events that happened in each block
// and calculates updated validator set based on changed stake
type stakeManager struct {
	logger                   hclog.Logger
	state                    *stakeStore
	stakeManagerContractAddr types.Address
	polybftBackend           polytypes.Polybft
	blockchain               polychain.Blockchain
}

// NewStakeManager returns a new instance of stake manager
func NewStakeManager(
	logger hclog.Logger,
	state *state.State,
	stakeManagerAddr types.Address,
	blockchain polychain.Blockchain,
	polybftBackend polytypes.Polybft,
	dbTx *bolt.Tx,
) (StakeManager, error) {
	stakeStore, err := newStakeStore(state.DB(), dbTx)
	if err != nil {
		return nil, err
	}

	return newStakeManager(logger, stakeStore, stakeManagerAddr, blockchain, polybftBackend, dbTx)
}

func newStakeManager(logger hclog.Logger,
	stakeStore *stakeStore,
	stakeManagerAddr types.Address,
	blockchain polychain.Blockchain,
	polybftBackend polytypes.Polybft,
	dbTx *bolt.Tx,
) (StakeManager, error) {
	sm := &stakeManager{
		logger:                   logger,
		state:                    stakeStore,
		stakeManagerContractAddr: stakeManagerAddr,
		polybftBackend:           polybftBackend,
		blockchain:               blockchain,
	}

	validatorSet, err := sm.getOrInitValidatorSet(dbTx)
	if err != nil {
		return nil, err
	}

	sm.logger.Debug("stake manager validator set initialized",
		"last saved", validatorSet.BlockNumber,
		"last updated", validatorSet.UpdatedAtBlockNumber)

	return sm, nil
}

// Close closes the oracle
func (s *stakeManager) Close() {}

// PostEpoch posts new epoch to the oracle
func (s *stakeManager) PostEpoch(req *oracle.PostEpochRequest) error {
	return nil
}

// PostBlock is called on every insert of finalized block (either from consensus or syncer)
// It will update the fullValidatorSet in db to the current block number
// Note that EventSubscriber - AddLog will get all the transfer events that happened in block
func (s *stakeManager) PostBlock(req *oracle.PostBlockRequest) error {
	fullValidatorSet, err := s.getOrInitValidatorSet(req.DBTx)
	if err != nil {
		return err
	}

	blockNumber := req.FullBlock.Block.Number()

	s.logger.Debug("Stake manager on post block",
		"block", blockNumber,
		"last saved", fullValidatorSet.BlockNumber,
		"last updated", fullValidatorSet.UpdatedAtBlockNumber)

	// we should save new state even if number of events is zero
	// because otherwise next time we will process more blocks
	fullValidatorSet.EpochID = req.Epoch
	fullValidatorSet.BlockNumber = blockNumber

	return s.state.insertFullValidatorSet(fullValidatorSet, req.DBTx)
}

func (s *stakeManager) getOrInitValidatorSet(dbTx *bolt.Tx) (validator.ValidatorSetState, error) {
	validatorSet, err := s.state.getFullValidatorSet(dbTx)
	if err != nil {
		if !errors.Is(err, errNoFullValidatorSet) {
			return validator.ValidatorSetState{}, err
		}

		validators, err := s.polybftBackend.GetValidatorsWithTx(0, nil, dbTx)
		if err != nil {
			return validator.ValidatorSetState{}, err
		}

		validatorSet = validator.ValidatorSetState{
			BlockNumber:          0,
			EpochID:              0,
			UpdatedAtBlockNumber: 0,
			Validators:           validator.NewValidatorStakeMap(validators),
		}

		if err = s.state.insertFullValidatorSet(validatorSet, dbTx); err != nil {
			return validator.ValidatorSetState{}, err
		}
	}

	return validatorSet, nil
}

func (s *stakeManager) updateWithReceipts(
	fullValidatorSet *validator.ValidatorSetState,
	events []contractsapi.EventAbi,
	blockNumber uint64) error {
	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		switch stakeEvent := event.(type) {
		case *contractsapi.StakeAddedEvent:
			s.logger.Debug("Stake added event", "to", stakeEvent.Validator, "amount", stakeEvent.Amount)

			fullValidatorSet.Validators.AddStake(stakeEvent.Validator, stakeEvent.Amount)
		case *contractsapi.StakeRemovedEvent:
			s.logger.Debug("Stake removed event", "from", stakeEvent.Validator, "value", stakeEvent.Amount)

			fullValidatorSet.Validators.RemoveStake(stakeEvent.Validator, stakeEvent.Amount)
		default:
			// this should not happen, but lets log it if it does
			s.logger.Warn("Found a stake event that represents neither stake nor unstake")
		}
	}

	for addr, data := range fullValidatorSet.Validators {
		if data.BlsKey == nil {
			blsKey, err := s.getBlsKey(data.Address)
			if err != nil {
				s.logger.Warn("Could not get info for new validator",
					"block", blockNumber, "address", addr)
			}

			data.BlsKey = blsKey
		}

		data.IsActive = data.VotingPower.Cmp(bigZero) > 0
	}

	// mark on which block validator set has been updated
	fullValidatorSet.UpdatedAtBlockNumber = blockNumber

	s.logger.Debug("Full validator set after", "block", blockNumber, "data", fullValidatorSet.Validators)

	return nil
}

// UpdateValidatorSet returns an updated validator set
// based on stake change (transfer) events from ValidatorSet contract
func (s *stakeManager) UpdateValidatorSet(epoch uint64, maxValidatorSetSize uint64,
	oldValidatorSet validator.AccountSet) (*validator.ValidatorSetDelta, error) {
	s.logger.Info("Calculating validators set update...", "epoch", epoch)

	fullValidatorSet, err := s.state.getFullValidatorSet(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get full validators set. Epoch: %d. Error: %w", epoch, err)
	}

	// stake map that holds stakes for all validators
	stakeMap := fullValidatorSet.Validators

	// slice of all validator set
	newValidatorSet := stakeMap.GetSorted(int(maxValidatorSetSize))
	// set of all addresses that will be in next validator set
	addressesSet := make(map[types.Address]struct{}, len(newValidatorSet))

	for _, si := range newValidatorSet {
		addressesSet[si.Address] = struct{}{}
	}

	removedBitmap := bitmap.Bitmap{}
	updatedValidators := validator.AccountSet{}
	addedValidators := validator.AccountSet{}
	oldActiveMap := make(map[types.Address]*validator.ValidatorMetadata)

	for i, validator := range oldValidatorSet {
		oldActiveMap[validator.Address] = validator
		// remove existing validators from validator set if they did not make it to the set
		if _, exists := addressesSet[validator.Address]; !exists {
			removedBitmap.Set(uint64(i))
		}
	}

	for _, newValidator := range newValidatorSet {
		// check if its already in existing validator set
		if oldValidator, exists := oldActiveMap[newValidator.Address]; exists {
			if oldValidator.VotingPower.Cmp(newValidator.VotingPower) != 0 {
				updatedValidators = append(updatedValidators, newValidator)
			}
		} else {
			if newValidator.BlsKey == nil {
				newValidator.BlsKey, err = s.getBlsKey(newValidator.Address)
				if err != nil {
					return nil, fmt.Errorf("could not retrieve validator data. Address: %v. Error: %w",
						newValidator.Address, err)
				}
			}

			addedValidators = append(addedValidators, newValidator)
		}
	}

	s.logger.Info("Calculating validators set update finished.", "epoch", epoch)

	delta := &validator.ValidatorSetDelta{
		Added:   addedValidators,
		Updated: updatedValidators,
		Removed: removedBitmap,
	}

	if s.logger.IsDebug() {
		newValidatorSet, err := oldValidatorSet.Copy().ApplyDelta(delta)
		if err != nil {
			return nil, err
		}

		s.logger.Debug("New validator set", "validatorSet", newValidatorSet)
	}

	return delta, nil
}

// getBlsKey returns bls key for validator from the supernet contract
func (s *stakeManager) getBlsKey(address types.Address) (*bls.PublicKey, error) {
	provider, err := s.blockchain.GetStateProviderForBlock(s.blockchain.CurrentHeader())
	if err != nil {
		return nil, err
	}

	stakeManagerContractContract := contract.NewContract(
		ethgo.Address(s.stakeManagerContractAddr),
		contractsapi.StakeManager.Abi, contract.WithProvider(provider),
	)

	rawResult, err := stakeManagerContractContract.Call("getValidator", ethgo.Latest, address)
	if err != nil {
		return nil, err
	}

	validatorData, ok := rawResult["0"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not collect validator's data (%s) from StakeManager", address)
	}

	blsKey, ok := validatorData["blsKey"].([4]*big.Int)
	if !ok {
		return nil, fmt.Errorf("failed to decode blskey")
	}

	pubKey, err := bls.UnmarshalPublicKeyFromBigInt(blsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BLS public key: %w", err)
	}

	return pubKey, nil
}

// EventSubscriber implementation

// GetLogFilters returns a map of log filters for getting desired events,
// where the key is the address of contract that emits desired events,
// and the value is a slice of signatures of events we want to get.
// This function is the implementation of EventSubscriber interface
func (s *stakeManager) GetLogFilters() map[types.Address][]types.Hash {
	return map[types.Address][]types.Hash{
		s.stakeManagerContractAddr: {
			types.Hash(new(contractsapi.StakeAddedEvent).Sig()),
			types.Hash(new(contractsapi.StakeRemovedEvent).Sig()),
		},
	}
}

// ProcessLog is the implementation of EventSubscriber interface,
// used to handle a log defined in GetLogFilters, provided by event provider
func (s *stakeManager) ProcessLog(header *types.Header, log *ethgo.Log, dbTx *bolt.Tx) error {
	var (
		stakeAddedEvent   contractsapi.StakeAddedEvent
		stakeRemovedEvent contractsapi.StakeRemovedEvent
		stakeEvents       = make([]contractsapi.EventAbi, 1)
	)

	switch log.Topics[0] {
	case stakeAddedEvent.Sig():
		doesMatch, err := stakeAddedEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if !doesMatch {
			return nil
		}

		stakeEvents[0] = &stakeAddedEvent
	case stakeRemovedEvent.Sig():
		doesMatch, err := stakeRemovedEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if !doesMatch {
			return nil
		}

		stakeEvents[0] = &stakeRemovedEvent
	default:
		return errUnknownStakeManagerEvent
	}

	fullValidatorSet, err := s.getOrInitValidatorSet(dbTx)
	if err != nil {
		return err
	}

	if err := s.updateWithReceipts(&fullValidatorSet, stakeEvents, header.Number); err != nil {
		return err
	}

	return s.state.insertFullValidatorSet(fullValidatorSet, dbTx)
}