package governance

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
)

type VoteType uint8
type ProposalState uint8

const (
	Against VoteType = iota
	For
	Abstain
)

const (
	Pending ProposalState = iota
	Active
	Canceled
	Defeated
	Succeeded
	Queued
	Expired
	Executed
)

type GovernanceTest interface {
	Name() string
	Run() error
}

// BaseGovernanceTest represents a base Governance test
type BaseGovernanceTest struct {
	config *GovernanceTestConfig

	testAccountKey *crypto.ECDSAKey
	client         *jsonrpc.EthClient

	txrelayer txrelayer.TxRelayer
}

// NewBaseGovernanceTest creates a new Governance test instance.
func NewBaseGovernanceTest(cfg *GovernanceTestConfig,
	testAccountKey *crypto.ECDSAKey, client *jsonrpc.EthClient) (*BaseGovernanceTest, error) {
	txRelayer, err := txrelayer.NewTxRelayer(
		txrelayer.WithClient(client),
		txrelayer.WithReceiptsTimeout(cfg.ReceiptsTimeout),
	)
	if err != nil {
		return nil, err
	}

	return &BaseGovernanceTest{
		config:         cfg,
		client:         client,
		txrelayer:      txRelayer,
		testAccountKey: testAccountKey,
	}, nil
}

// decodePrivateKey decodes the given private key string.
func decodePrivateKey(privateKeyRaw string) (*crypto.ECDSAKey, error) {
	raw, err := hex.DecodeString(privateKeyRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key string '%s': %w", privateKeyRaw, err)
	}

	return crypto.NewECDSAKeyFromRawPrivECDSA(raw)
}

// sendQueueProposalTransaction sends a queue proposal to the ChildGovernor contract.
func (b *BaseGovernanceTest) sendQueueProposalTransaction(
	senderKey crypto.Key,
	input []byte, description string) error {

	queueFn := contractsapi.QueueChildGovernorFn{
		Targets:         []types.Address{contracts.NetworkParamsContract},
		Calldatas:       [][]byte{input},
		DescriptionHash: crypto.Keccak256Hash([]byte(description)),
		Values:          []*big.Int{big.NewInt(0)},
	}

	input, err := queueFn.EncodeAbi()
	if err != nil {
		return err
	}

	txn := types.NewTx(types.NewLegacyTx(
		types.WithTo(&contracts.ChildGovernorContract),
		types.WithInput(input),
	))

	receipt, err := b.txrelayer.SendTransaction(txn, senderKey)
	if err != nil {
		return err
	}

	if uint64(types.ReceiptSuccess) != receipt.Status {
		return err
	}

	return nil
}

// sendProposalTransaction submits a proposal to the ChildGovernor contract
// and returns the created proposal ID.
func (b *BaseGovernanceTest) sendProposalTransaction(
	senderKey crypto.Key,
	input []byte, description string) (*big.Int, error) {

	proposeFn := &contractsapi.ProposeChildGovernorFn{
		Targets:     []types.Address{contracts.NetworkParamsContract},
		Calldatas:   [][]byte{input},
		Description: description,
		Values:      []*big.Int{big.NewInt(0)},
	}

	input, err := proposeFn.EncodeAbi()
	if err != nil {
		return nil, err
	}

	txn := types.NewTx(types.NewLegacyTx(
		types.WithTo(&contracts.ChildGovernorContract),
		types.WithInput(input),
	))

	receipt, err := b.txrelayer.SendTransaction(txn, senderKey)
	if err != nil {
		return nil, err
	}

	if uint64(types.ReceiptSuccess) != receipt.Status {
		return nil, fmt.Errorf("send proposal transaction receipt status == failed")
	}

	var proposalCreatedEvent contractsapi.ProposalCreatedEvent
	for _, log := range receipt.Logs {
		doesMatch, err := proposalCreatedEvent.ParseLog(log)
		if err != nil {
			return nil, fmt.Errorf("parsing log error:%w", err)
		}

		if doesMatch {
			break
		}
	}

	if reflect.DeepEqual(proposalCreatedEvent, contractsapi.ProposalCreatedEvent{}) {
		return nil, fmt.Errorf("proposal event empty")
	}

	return proposalCreatedEvent.ProposalID, nil
}

// executeSuccessfulProposalCycle runs the full governance proposal lifecycle and
// verifies the updated network parameter.
func (b *BaseGovernanceTest) executeSuccessfulProposalCycle(
	proposalInput []byte, proposerAcc *crypto.ECDSAKey,
	proposalDescription, fieldName string, expectedValue *big.Int) error {
	proposalID, err := b.sendProposalTransaction(proposerAcc,
		proposalInput, proposalDescription)
	if err != nil {
		return err
	}

	if err := waitUntil(3*time.Minute, 2*time.Second, func() (bool, error) {
		proposalState, err := b.getProposalState(proposalID)

		return proposalState == Active, err
	}); err != nil {
		return err
	}

	for _, key := range b.config.ValidatorKeys {
		privKey, err := decodePrivateKey(key)
		if err != nil {
			return err
		}

		if err := b.sendVoteTransaction(proposalID, For, privKey); err != nil {
			return err
		}
	}

	if err := waitUntil(3*time.Minute, 2*time.Second, func() (bool, error) {
		proposalState, err := b.getProposalState(proposalID)

		return proposalState == Succeeded, err
	}); err != nil {
		return err
	}

	if err := b.sendQueueProposalTransaction(
		proposerAcc, proposalInput, proposalDescription); err != nil {
		return err
	}

	if err := waitUntil(3*time.Minute, 2*time.Second, func() (bool, error) {
		proposalState, err := b.getProposalState(proposalID)

		return proposalState == Queued, err
	}); err != nil {
		return err
	}

	currentBlockNumber, err := b.txrelayer.Client().BlockNumber()
	if err != nil {
		return err
	}

	if err := waitForBlock(currentBlockNumber+2, 10*time.Second, b.txrelayer); err != nil {
		return err
	}

	if err := b.sendExecuteProposalTransaction(
		proposerAcc, proposalInput, proposalDescription); err != nil {
		return err
	}

	networkParamsRespons, err := ABICall(b.txrelayer, contractsapi.NetworkParams,
		contracts.NetworkParamsContract, types.ZeroAddress, fieldName)
	if err != nil {
		return err
	}

	paramValueOnNetworkParams, err := common.ParseUint256orHex(&networkParamsRespons)
	if err != nil {
		return err
	}

	if expectedValue.Uint64() != paramValueOnNetworkParams.Uint64() {
		return fmt.Errorf("expected value and param value on network not equal")
	}

	return nil
}

// getProposalState returns the current state of the given proposal by calling the ChildGovernor contract.
func (b *BaseGovernanceTest) getProposalState(proposalID *big.Int) (ProposalState, error) {
	stateFn := &contractsapi.StateChildGovernorFn{
		ProposalID: proposalID,
	}

	input, err := stateFn.EncodeAbi()
	if err != nil {
		return 0, err
	}

	response, err := b.txrelayer.Call(types.ZeroAddress, contracts.ChildGovernorContract, input)
	if err != nil {
		return 0, err
	}

	if "0x" == response {
		return 0, fmt.Errorf("response not equal to 0x")
	}

	converted, err := common.ParseUint64orHex(&response)
	if err != nil {
		return 0, err
	}

	return ProposalState(converted), nil
}

// sendVoteTransaction submits a vote on the given proposal to the ChildGovernor contract.
func (b *BaseGovernanceTest) sendVoteTransaction(proposalID *big.Int, vote VoteType,
	senderKey crypto.Key) error {
	castVoteFn := &contractsapi.CastVoteChildGovernorFn{
		ProposalID: proposalID,
		Support:    uint8(vote),
	}

	input, err := castVoteFn.EncodeAbi()
	if err != nil {
		return err
	}

	txn := types.NewTx(types.NewLegacyTx(
		types.WithTo(&contracts.ChildGovernorContract),
		types.WithInput(input),
	))

	receipt, err := b.txrelayer.SendTransaction(txn, senderKey)
	if err != nil {
		return err
	}

	if uint64(types.ReceiptSuccess) != receipt.Status {
		return fmt.Errorf("send vote transaction receipt status == failed")
	}

	return nil
}

// sendExecuteProposalTransaction executes a successful proposal on the ChildGovernor contract
// using the given calldata and description.
func (b *BaseGovernanceTest) sendExecuteProposalTransaction(
	senderKey crypto.Key, input []byte,
	description string) error {
	executeFn := &contractsapi.ExecuteChildGovernorFn{
		Targets:         []types.Address{contracts.NetworkParamsContract},
		Calldatas:       [][]byte{input},
		DescriptionHash: crypto.Keccak256Hash([]byte(description)),
		Values:          []*big.Int{big.NewInt(0)},
	}

	input, err := executeFn.EncodeAbi()
	if err != nil {
		return err
	}

	txn := types.NewTx(types.NewLegacyTx(
		types.WithTo(&contracts.ChildGovernorContract),
		types.WithInput(input),
	))

	receipt, err := b.txrelayer.SendTransaction(txn, senderKey)
	if err != nil {
		return err
	}

	if uint64(types.ReceiptSuccess) != receipt.Status {
		return fmt.Errorf("execute proposal receipt status failed")
	}

	return nil
}
