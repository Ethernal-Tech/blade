package governance

import (
	"encoding/hex"
	"fmt"
	"math/big"
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

// NewBaseGovernanceTest creates a new Governance
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

func sendQueueProposalTransaction(
	txRelayer txrelayer.TxRelayer, senderKey crypto.Key,
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

	receipt, err := txRelayer.SendTransaction(txn, senderKey)
	if err != nil {
		return err
	}

	if uint64(types.ReceiptSuccess) != receipt.Status {
		return err
	}

	return nil
}

func sendProposalTransaction(txRelayer txrelayer.TxRelayer,
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

	receipt, err := txRelayer.SendTransaction(txn, senderKey)
	if err != nil {
		return nil, err
	}

	if uint64(types.ReceiptSuccess) == receipt.Status {
		return nil, fmt.Errorf("receipt status failed")
	}

	var proposalCreatedEvent *contractsapi.ProposalCreatedEvent
	for _, log := range receipt.Logs {
		doesMatch, err := proposalCreatedEvent.ParseLog(log)
		if err != nil {
			return nil, err
		}

		if doesMatch {
			break
		}
	}

	if proposalCreatedEvent == nil {
		return nil, fmt.Errorf("proposal event is nil")
	}

	return proposalCreatedEvent.ProposalID, nil
}

func executeSuccssfulProposalCycle(cfg *GovernanceTestConfig, relayer txrelayer.TxRelayer,
	proposalInput []byte, proposerAcc *crypto.ECDSAKey,
	proposalDescription, fieldname string, expectedValue *big.Int) error {
	proposalID, err := sendProposalTransaction(relayer, proposerAcc,
		proposalInput, proposalDescription)
	if err != nil {
		return err
	}

	if err := waitUntil(3*time.Minute, 2*time.Second, func() (bool, error) {
		proposalState, err := getProposalState(proposalID, relayer)

		return proposalState == Active, err
	}); err != nil {
		return err
	}

	for _, key := range cfg.ValidatorKeys {
		privKey, err := decodePrivateKey(key)
		if err != nil {
			return err
		}

		if err := sendVoteTransaction(proposalID, For, relayer, privKey); err != nil {
			return err
		}
	}

	if err := waitUntil(3*time.Minute, 2*time.Second, func() (bool, error) {
		proposalState, err := getProposalState(proposalID, relayer)

		return proposalState == Succeeded, err
	}); err != nil {
		return err
	}

	if err := sendQueueProposalTransaction(relayer,
		proposerAcc, proposalInput, proposalDescription); err != nil {
		return err
	}

	if err := waitUntil(3*time.Minute, 2*time.Second, func() (bool, error) {
		proposalState, err := getProposalState(proposalID, relayer)

		return proposalState == Queued, err
	}); err != nil {
		return err
	}

	currentBlockNumber, err := relayer.Client().BlockNumber()
	if err != nil {
		return err
	}

	if err := waitForBlock(int64(currentBlockNumber+2), 10*time.Second); err != nil {
		return err
	}

	if err := sendExecuteProposalTransaction(
		relayer, proposerAcc, proposalInput, proposalDescription); err != nil {
		return err
	}

	networkParamsRespons, err := ABICall(relayer, contractsapi.NetworkParams,
		contracts.NetworkParamsContract, types.ZeroAddress, fieldname)
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

func getProposalState(proposalID *big.Int, txRelayer txrelayer.TxRelayer) (ProposalState, error) {

	stateFn := &contractsapi.StateChildGovernorFn{
		ProposalID: proposalID,
	}

	input, err := stateFn.EncodeAbi()
	if err != nil {
		return 0, err
	}

	response, err := txRelayer.Call(types.ZeroAddress, contracts.ChildGovernorContract, input)
	if err != nil {
		return 0, err
	}

	if "0x" != response {
		return 0, fmt.Errorf("response not equal to 0x")
	}

	converted, err := common.ParseUint64orHex(&response)
	if err != nil {
		return 0, err
	}

	return ProposalState(converted), nil
}

func sendVoteTransaction(proposalID *big.Int, vote VoteType,
	txRelayer txrelayer.TxRelayer, senderKey crypto.Key) error {
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

	receipt, err := txRelayer.SendTransaction(txn, senderKey)
	if err != nil {
		return err
	}

	if uint64(types.ReceiptSuccess) != receipt.Status {
		return fmt.Errorf("receipt failed")
	}

	return nil
}

func sendExecuteProposalTransaction(txRelayer txrelayer.TxRelayer,
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

	receipt, err := txRelayer.SendTransaction(txn, senderKey)
	if err != nil {
		return err
	}

	if uint64(types.ReceiptSuccess) != receipt.Status {
		return fmt.Errorf("receipt status failed")
	}

	return nil
}
