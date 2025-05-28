package governance

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Ethernal-Tech/ethgo"
	"github.com/Ethernal-Tech/ethgo/abi"
	"github.com/Ethernal-Tech/ethgo/contract"
	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/mapstructure"
	bolt "go.etcd.io/bbolt"

	"github.com/0xPolygon/polygon-edge/chain"
	polychain "github.com/0xPolygon/polygon-edge/consensus/polybft/blockchain"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/oracle"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/state"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/forkmanager"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/types"
)

var (
	errUnknownGovernanceEvent = errors.New("unknown event from governance")
	stringABIType             = abi.MustNewType("tuple(string)")
)

// GovernanceManager interface provides functions for handling governance events
// and updating client configuration based on executed governance proposals
type GovernanceManager interface {
	state.EventSubscriber
	oracle.ReadOnlyOracle
	GetClientConfig(dbTx *bolt.Tx) (*chain.Params, error)
}

var _ GovernanceManager = (*DummyGovernanceManager)(nil)

// dummyStakeManager is a dummy implementation of GovernanceManager interface
// used only for unit testing
type DummyGovernanceManager struct {
	GetClientConfigFn func() (*chain.Params, error)
}

func (d *DummyGovernanceManager) Close()                                       {}
func (d *DummyGovernanceManager) PostBlock(req *oracle.PostBlockRequest) error { return nil }
func (d *DummyGovernanceManager) PostEpoch(req *oracle.PostEpochRequest) error { return nil }
func (d *DummyGovernanceManager) GetClientConfig(dbTx *bolt.Tx) (*chain.Params, error) {
	if d.GetClientConfigFn != nil {
		return d.GetClientConfigFn()
	}

	return nil, nil
}

// EventSubscriber implementation
func (d *DummyGovernanceManager) GetLogFilters() map[types.Address][]types.Hash {
	return make(map[types.Address][]types.Hash)
}

func (d *DummyGovernanceManager) ProcessLog(header *types.Header, log *ethgo.Log, dbTx *bolt.Tx) error {
	return nil
}

var _ GovernanceManager = (*governanceManager)(nil)

// governanceManager is a struct that saves governance events
// and updates the client configuration based on executed governance proposals
type governanceManager struct {
	logger         hclog.Logger
	state          *GovernanceStore
	allForksHashes map[types.Hash]string
	blockchain     polychain.Blockchain
}

// NewGovernanceManager is a constructor function for governance manager
func NewGovernanceManager(genesisParams *chain.Params,
	logger hclog.Logger,
	state *state.State,
	blockhain polychain.Blockchain,
	dbTx *bolt.Tx) (GovernanceManager, error) {
	store, err := newGovernanceStore(state.DB(), dbTx)
	if err != nil {
		return nil, fmt.Errorf("could not create governance store. Error: %w", err)
	}

	config, err := store.getClientConfig(dbTx)
	if config == nil || errors.Is(err, errClientConfigNotFound) {
		// insert initial config to db if not already inserted
		if err = store.insertClientConfig(genesisParams, dbTx); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// cache all fork name hashes that we have in code
	allForkNameHashes := map[types.Hash]string{}

	for name := range *chain.AllForksEnabled {
		encoded, err := stringABIType.Encode([]interface{}{name})
		if err != nil {
			return nil, fmt.Errorf("could not encode fork name: %s. Error: %w", name, err)
		}

		forkHash := crypto.Keccak256Hash(encoded)
		allForkNameHashes[forkHash] = name
	}

	g := &governanceManager{
		logger:         logger,
		state:          store,
		allForksHashes: allForkNameHashes,
		blockchain:     blockhain,
	}

	// get all features from contract
	featuresFromContract, err := g.getAllFeatures()
	if err != nil {
		return nil, fmt.Errorf("could not activate forks from db on startup. Error: %w", err)
	}

	lastBuiltBlock := blockhain.CurrentHeader().Number

	if err := g.activateNewForks(lastBuiltBlock, featuresFromContract); err != nil {
		return nil, err
	}

	return g, nil
}

// GetClientConfig returns latest client configuration from boltdb
func (g *governanceManager) GetClientConfig(dbTx *bolt.Tx) (*chain.Params, error) {
	return g.state.getClientConfig(dbTx)
}

// Close closes the governance manager
func (g *governanceManager) Close() {}

type networkParams struct {
	CheckpointBlockInterval *big.Int `json:"checkpointBlockInterval"`
	EpochSize               *big.Int `json:"epochSize"`
	EpochReward             *big.Int `json:"epochReward"`
	SprintSize              *big.Int `json:"sprintSize"`
	MinValidatorSetSize     *big.Int `json:"minValidatorSetSize"`
	MaxValidatorSetSize     *big.Int `json:"maxValidatorSetSize"`
	WithdrawalWaitPeriod    *big.Int `json:"withdrawalWaitPeriod"`
	BlockTime               *big.Int `json:"blockTime"`
	BlockTimeDrift          *big.Int `json:"blockTimeDrift"`
	VotingDelay             *big.Int `json:"votingDelay"`
	VotingPeriod            *big.Int `json:"votingPeriod"`
	ProposalThreshold       *big.Int `json:"proposalThreshold"`
	BaseFeeChangeDenom      *big.Int `json:"baseFeeChangeDenom"`
}

func (g *governanceManager) getNetworkParams() (*networkParams, error) {
	provider, err := g.blockchain.GetStateProviderForBlock(g.blockchain.CurrentHeader())
	if err != nil {
		return nil, fmt.Errorf("could not get state provider for current block: %w", err)
	}

	networkParamsContract := contract.NewContract(
		ethgo.Address(contracts.NetworkParamsContract),
		contractsapi.NetworkParams.Abi,
		contract.WithProvider(provider),
	)

	result, err := networkParamsContract.Call("getNetworkParams", ethgo.Latest)
	if err != nil {
		return nil, fmt.Errorf("could not get network params from NetworkParams contract: %w", err)
	}

	var networkParams *networkParams
	if err := mapstructure.Decode(result["0"], &networkParams); err != nil {
		return nil, fmt.Errorf("could not decode NetworkParams contract response: %w", err)
	}

	return networkParams, nil
}

// PostEpoch notifies the governance manager that an epoch has changed
func (g *governanceManager) PostEpoch(req *oracle.PostEpochRequest) error {
	if !req.Forks.IsActive(chain.Governance, req.FirstBlockOfEpoch) {
		// if governance fork is not enabled, do nothing
		return nil
	}

	previousEpoch := req.NewEpochID - 1

	g.logger.Debug("Post epoch - getting events from executed governance proposals...",
		"epoch", previousEpoch)

	networkParams, err := g.getNetworkParams()
	if err != nil {
		return fmt.Errorf("could not get network params: %w", err)
	}

	// get last saved config
	latestChainParams, err := g.state.getClientConfig(req.DBTx)
	if err != nil {
		return err
	}

	latestPolybftConfig, err := config.GetPolyBFTConfig(latestChainParams)
	if err != nil {
		return err
	}

	latestPolybftConfig.CheckpointInterval = networkParams.CheckpointBlockInterval.Uint64()
	latestPolybftConfig.SprintSize = networkParams.SprintSize.Uint64()
	latestPolybftConfig.EpochSize = networkParams.EpochSize.Uint64()
	latestPolybftConfig.EpochReward = networkParams.EpochReward.Uint64()
	latestPolybftConfig.WithdrawalWaitPeriod = networkParams.WithdrawalWaitPeriod.Uint64()
	latestPolybftConfig.BlockTime = common.Duration{Duration: time.Duration(networkParams.BlockTime.Int64()) * time.Second}
	latestPolybftConfig.BlockTimeDrift = networkParams.BlockTimeDrift.Uint64()
	if latestPolybftConfig.GovernanceConfig == nil {
		latestPolybftConfig.GovernanceConfig = &config.Governance{}
	}

	latestPolybftConfig.GovernanceConfig.VotingDelay = networkParams.VotingDelay
	latestPolybftConfig.GovernanceConfig.VotingPeriod = networkParams.VotingPeriod
	latestPolybftConfig.GovernanceConfig.ProposalThreshold = networkParams.ProposalThreshold
	latestPolybftConfig.MinValidatorSetSize = networkParams.MinValidatorSetSize.Uint64()
	latestPolybftConfig.MaxValidatorSetSize = networkParams.MaxValidatorSetSize.Uint64()

	latestChainParams.BaseFeeChangeDenom = networkParams.BaseFeeChangeDenom.Uint64()
	latestChainParams.Engine[config.ConsensusName] = latestPolybftConfig

	// save updated config to db
	return g.state.insertClientConfig(latestChainParams, req.DBTx)
}

func (g *governanceManager) getAllFeatures() (map[types.Hash]*big.Int, error) {
	provider, err := g.blockchain.GetStateProviderForBlock(g.blockchain.CurrentHeader())
	if err != nil {
		return nil, fmt.Errorf("could not get state provider for current block: %w", err)
	}

	forkContract := contract.NewContract(
		ethgo.Address(contracts.ForkParamsContract),
		contractsapi.ForkParams.Abi,
		contract.WithProvider(provider),
	)

	result, err := forkContract.Call("getAllFeatures", ethgo.Latest)
	if err != nil {
		return nil, fmt.Errorf("could not get all forks from ForkParams contract: %w", err)
	}

	type featureInfo struct {
		BlockNumber *big.Int   `json:"blockNumber" mapstructure:"blockNumber"`
		Feature     types.Hash `json:"feature" mapstructure:"feature"`
	}

	var features []*featureInfo

	if err := mapstructure.Decode(result["0"], &features); err != nil {
		return nil, fmt.Errorf("could not decode ForkParams contract response: %w", err)
	}

	allFeatures := make(map[types.Hash]*big.Int, len(features))
	for _, feature := range features {
		allFeatures[feature.Feature] = feature.BlockNumber
	}

	return allFeatures, nil
}

// PostBlock notifies governance manager that a block was finalized
// so that he can extract governance events and save them to bolt db
func (g *governanceManager) PostBlock(req *oracle.PostBlockRequest) error {
	if !req.Forks.IsActive(chain.Governance, req.FullBlock.Block.Number()) {
		// if governance fork is not enabled, do nothing
		return nil
	}

	g.logger.Debug("Post block - Activating any new forks...", "epoch", req.Epoch,
		"block", req.FullBlock.Block.Number())

	currentBlock := req.FullBlock.Block.Number()

	forkEvents, err := g.getAllFeatures()
	if err != nil {
		g.logger.Debug("Post block - Getting fork events failed.", "epoch", req.Epoch,
			"block", currentBlock)

		return err
	}

	g.logger.Debug("Post block - Getting fork events done.", "epoch", req.Epoch,
		"block", currentBlock, "eventsNum", len(forkEvents))

	return g.activateNewForks(currentBlock, forkEvents)
}

// activateNewForks activates forks based on the ForkParams contract events
func (g *governanceManager) activateNewForks(currentBlock uint64,
	forkEvents map[types.Hash]*big.Int) error {
	for forkHash, forkBlock := range forkEvents {
		if err := g.activateSingleFork(currentBlock, forkHash, forkBlock); err != nil {
			return err
		}
	}

	return nil
}

// activateSingleFork registers (if not already registered) and activates fork from specified block
// if given fork does not exist in code (the code version is not up to data to that fork), it will panic
func (g *governanceManager) activateSingleFork(currentBlock uint64, forkHash types.Hash, forkBlock *big.Int) error {
	forkManager := forkmanager.GetInstance()

	// here we add registration and activation of forks
	// based on ForkParams contract
	forkName, exists := g.allForksHashes[forkHash]
	if !exists {
		// current code version does not have this fork
		// so we stop the node
		panic(fmt.Sprintf("current version of code does not have this fork: %v", forkHash)) //nolint:gocritic
	}

	if !forkManager.IsForkRegistered(forkName) {
		// if fork is not already registered, register it
		forkManager.RegisterFork(forkName, nil)
	}

	if forkManager.IsForkEnabled(forkName, currentBlock) {
		// this fork is already enabled and up and running,
		// if an event happens to activate it from some later block
		// we will first deactivate it, and activate it again from the new specified block
		if currentBlock >= forkBlock.Uint64() {
			// fork is already activated, no need to do anything
			return nil
		}

		g.logger.Debug("Gotten event for fork de-activation", "currentBlock", currentBlock, "forkName", forkName,
			"forkHash", forkHash, "forkBlock", forkBlock.Uint64())

		if err := forkManager.DeactivateFork(forkName); err != nil {
			return fmt.Errorf("could not deactivate fork: %s. Error: %w", forkName, err)
		}
	}

	g.logger.Debug("Gotten event for fork activation", "currentBlock", currentBlock, "forkName", forkName,
		"forkHash", forkHash, "forkBlock", forkBlock.Uint64())

	blockNum := forkBlock.Uint64()
	if err := forkManager.ActivateFork(forkName, blockNum); err != nil {
		// activate fork for given number
		return fmt.Errorf("could not activate fork: %s for block: %d based on ForkParams event",
			forkName, blockNum)
	}

	g.logger.Debug("Registered and activated fork", "currentBlock", currentBlock, "forkName", forkName,
		"forkHash", forkHash, "block", forkBlock.Uint64())

	return nil
}

// isForkParamsEvent returns true if given contractsapi event is an event
// from ForkParams contract, and returns its feature hash and blockNumber as well
func isForkParamsEvent(event contractsapi.EventAbi) (types.Hash, *big.Int, bool) {
	switch obj := event.(type) {
	case *contractsapi.NewFeatureEvent:
		return obj.Feature, obj.Block, true
	case *contractsapi.UpdatedFeatureEvent:
		return obj.Feature, obj.Block, true
	default:
		return types.ZeroHash, nil, false
	}
}

// EventSubscriber implementation
func (g *governanceManager) GetLogFilters() map[types.Address][]types.Hash {
	return nil
}

func (g *governanceManager) ProcessLog(header *types.Header, log *ethgo.Log, dbTx *bolt.Tx) error {
	return nil
}

// unmarshalGovernanceEvent unmarshals given raw event to desired type
func unmarshalGovernanceEvent[T contractsapi.EventAbi](rawEvent []byte) (T, error) {
	var event T

	err := json.Unmarshal(rawEvent[types.HashLength:], &event)

	return event, err
}
