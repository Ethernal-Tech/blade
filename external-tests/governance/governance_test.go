package governance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGovernance(t *testing.T) {
	t.Skip("this is only added for the sake of the example and running it in local")

	config := &GovernanceTestConfig{
		JSONRPCUrl:    "http://localhost:10002",
		ValidatorKeys: []string{"1a7626c5a1d89030f300ca5f63eecac3bae3e56f14033ea2d9ad471e7c93020e"},
		EpochSize:     10,
	}

	runner, err := NewGovernanceTestRunner(config)
	if err != nil {
		t.Fatal(err)
	}

	defer runner.Close()

	require.NoError(t, runner.Run())
}
