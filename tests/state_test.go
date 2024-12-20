package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/state"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

// Currently used test cases suite version is v10.4.
// It does not include Merge hardfork test cases.

const (
	stateTestsDir = "tests/GeneralStateTests"
)

var (
	ripemd = types.StringToAddress("0000000000000000000000000000000000000003")
)

func TestState(t *testing.T) {
	t.Parallel()

	long := []string{
		"static_Call50000",
		"static_Return50000",
		"static_Call1MB",
		"stQuadraticComplexityTest",
		"stTimeConsuming",
		"modexp",
		"CallRecursiveBomb",
		"ContractCreationSpam",
		"badOpcodes",
		"Opcodes_TransactionInit",
	}

	skip := []string{
		"RevertPrecompiledTouch",
		"RevertPrecompiledTouch_storage",
		"loopMul",
		"CALLBlake2f_MaxRounds",
	}

	files, err := listFiles(stateTestsDir, ".json")
	require.NoError(t, err)

	for _, file := range files {
		if contains(long, file) && testing.Short() {
			t.Logf("Long test '%s' is skipped in short mode\n", file)

			continue
		}

		if contains(skip, file) {
			t.Logf("Test '%s' is skipped\n", file)

			continue
		}

		file := file
		t.Run(file, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(file)
			require.NoError(t, err)

			var testCases map[string]testCase
			if err = json.Unmarshal(data, &testCases); err != nil {
				t.Fatalf("failed to unmarshal %s: %v", file, err)
			}

			for _, tc := range testCases {
				for fork, postState := range tc.Post {
					forks, exists := Forks[fork]
					if !exists {
						t.Logf("%s fork is not supported, skipping test case.", fork)
						continue
					}

					fc := &forkConfig{name: fork, forks: forks}

					for idx, postStateEntry := range postState {
						start := time.Now()
						err := runSpecificTestCase(t, file, tc, fc, idx, postStateEntry)

						t.Logf("'%s' executed. Fork: %s. Case: %d, Duration=%v\n", file, fork, idx, time.Since(start))

						require.NoError(t, tc.checkError(fork, idx, err))
					}
				}
			}
		})
	}
}

func runSpecificTestCase(t *testing.T, file string, c testCase, fc *forkConfig, index int, p postEntry) error {
	t.Helper()

	testName := getTestName(file)

	env := c.Env.ToEnv(t)
	forks := fc.forks

	var baseFee *big.Int

	if forks.IsActive(chain.London, 0) {
		if c.Env.BaseFee != "" {
			baseFee = stringToBigIntT(t, c.Env.BaseFee)
		} else {
			// Retesteth uses `10` for genesis baseFee. Therefore, it defaults to
			// parent - 2 : 0xa as the basefee for 'this' context.
			baseFee = big.NewInt(testGenesisBaseFee)
		}
	}

	msg, err := c.Transaction.At(p.Indexes, baseFee)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	s, snapshot, pastRoot, err := buildState(c.Pre)
	if err != nil {
		return err
	}

	currentForks := forks.At(uint64(env.Number))

	// Try to recover tx with current signer
	if len(p.TxBytes) != 0 {
		ttx := &types.Transaction{}
		err := ttx.UnmarshalRLP(p.TxBytes)
		if err != nil {
			return err
		}

		signer := crypto.NewSigner(currentForks, 1)

		if _, err := signer.Sender(ttx); err != nil {
			return err
		}
	}

	executor := state.NewExecutor(&chain.Params{
		Forks:   forks,
		ChainID: 1,
		BurnContract: map[uint64]types.Address{
			0: types.ZeroAddress,
		},
	}, s, hclog.NewNullLogger())

	executor.PostHook = func(t *state.Transition) {
		if testName == "failed_tx_xcf416c53" {
			// create the account
			t.Txn().TouchAccount(ripemd)
			// now remove it
			t.Txn().Suicide(ripemd)
		}
	}
	executor.GetHash = func(*types.Header) func(i uint64) types.Hash {
		return vmTestBlockHash
	}

	transition, err := executor.BeginTxn(pastRoot, c.Env.ToHeader(t), env.Coinbase)
	if err != nil {
		return err
	}
	_, err = transition.Apply(msg)
	if err != nil {
		return err
	}

	txn := transition.Txn()

	// mining rewards
	txn.AddSealingReward(env.Coinbase, big.NewInt(0))

	objs, err := txn.Commit(currentForks.EIP155)
	if err != nil {
		return err
	}

	_, root, err := snapshot.Commit(objs)
	if err != nil {
		return err
	}

	// Check block root
	if !bytes.Equal(root, p.Root.Bytes()) {
		t.Fatalf(
			"root mismatch (%s %s case#%d): expected %s but found %s",
			file,
			fc.name,
			index,
			p.Root.String(),
			hex.EncodeToHex(root),
		)
	}

	// Check transaction logs
	if logs := rlpHashLogs(txn.Logs()); logs != p.Logs {
		t.Fatalf(
			"logs mismatch (%s %s case#%d): expected %s but found %s",
			file,
			fc.name,
			index,
			p.Logs.String(),
			logs.String(),
		)
	}

	return nil
}
