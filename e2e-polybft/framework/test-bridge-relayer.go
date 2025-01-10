package framework

import (
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/crypto"
)

type TestRelayer struct {
	t *testing.T

	clusterConfig *TestClusterConfig
	node          *node
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
		t:             t,
		clusterConfig: clusterConfig,
	}

	relayer.start(externalChainID, key, genesisPath, validatorJSONRPC)

	return relayer
}

func (t *TestRelayer) start(externalChainID uint64, key *crypto.ECDSAKey, genesisPath, validatorJSONRPC string) {
	marshalledKey, err := key.MarshallPrivateKey()
	if err != nil {
		t.t.Fatal(err)
	}

	// build arguments
	args := []string{
		"bridge-relayer",
		"--genesis-path", genesisPath,
		"--private-key", hex.EncodeToString(marshalledKey),
		"--internal-chain-rpc", validatorJSONRPC,
		"--external-chain-id", strconv.FormatUint(externalChainID, 10),
		"--poll-interval", strconv.Itoa(5),
		"--database-path", t.t.TempDir() + "bridge-relayer.db",
	}

	stdout := t.clusterConfig.GetStdout("bridge-relayer")

	node, err := newNode(t.clusterConfig.Binary, args, stdout)
	if err != nil {
		t.t.Fatal(err)
	}

	t.node = node

	time.Sleep(250 * time.Millisecond)
}

func (t *TestRelayer) stop() {
	if err := t.node.Stop(); err != nil {
		t.t.Fatal(err)
	}

	t.node = nil
}
