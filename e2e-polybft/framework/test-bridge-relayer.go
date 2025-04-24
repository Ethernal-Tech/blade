package framework

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/crypto"
)

type TestRelayer struct {
	t                *testing.T
	clusterConfig    *TestClusterConfig
	externalChainID  uint64
	genesisPath      string
	key              *crypto.ECDSAKey
	validatorJSONRPC string

	node *node
}

func NewTestBridgeRelayer(
	t *testing.T,
	clusterConfig *TestClusterConfig,
	externalChainID uint64,
	genesisPath string,
	key *crypto.ECDSAKey,
	validatorJSONRPC string) *TestRelayer {
	t.Helper()

	relayer := &TestRelayer{
		t:                t,
		clusterConfig:    clusterConfig,
		externalChainID:  externalChainID,
		genesisPath:      genesisPath,
		key:              key,
		validatorJSONRPC: validatorJSONRPC,
	}

	return relayer
}

func (t *TestRelayer) Start() {
	marshalledKey, err := t.key.MarshallPrivateKey()
	if err != nil {
		t.t.Fatal(err)
	}

	// build arguments
	args := []string{
		"bridge-relayer",
		"--genesis-path", t.genesisPath,
		"--private-key", hex.EncodeToString(marshalledKey),
		"--internal-chain-rpc", t.validatorJSONRPC,
		"--external-chain-id", strconv.FormatUint(t.externalChainID, 10),
		"--poll-interval", strconv.Itoa(5),
		"--database-path", fmt.Sprintf("%s/bridge-relayer-%d.db", filepath.Dir(t.t.TempDir()), t.externalChainID),
		"--metrics-endpoint", fmt.Sprintf(":54321/"),
	}

	stdout := t.clusterConfig.GetStdout("bridge-relayer")

	node, err := newNode(t.clusterConfig.Binary, args, stdout)
	if err != nil {
		t.t.Fatal(err)
	}

	t.node = node

	time.Sleep(250 * time.Millisecond)
}

func (t *TestRelayer) Stop() {
	if err := t.node.Stop(); err != nil {
		t.t.Fatal(err)
	}

	t.node = nil
}
