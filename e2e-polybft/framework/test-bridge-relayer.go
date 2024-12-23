package framework

import (
	"strconv"
	"testing"
	"time"
)

type TestRelayer struct {
	t *testing.T

	clusterConfig   *TestClusterConfig
	externalChainID uint64
	internalChainID uint64
	node            *node
	genesisPath     string
}

func NewTestBridgeRelayer(t *testing.T, clusterConfig *TestClusterConfig, externalChainID, internalChainID uint64, genesisPath string, privateKey string) *TestRelayer {
	relayer := &TestRelayer{
		t:               t,
		internalChainID: internalChainID,
		externalChainID: externalChainID,
		genesisPath:     genesisPath,
		clusterConfig:   clusterConfig,
	}

	relayer.Start(privateKey)

	return relayer
}

func (t *TestRelayer) Start(privateKey string) {
	//Build arguments
	args := []string{
		"bridge-relayer",
		"--genesis-path", t.genesisPath,
		"--private-key", privateKey,
		"--internal-chain-rpc", "https://localhost:12001",
		"--external-chain-id", strconv.FormatUint(t.externalChainID, 10),
		"--poll-interval", strconv.Itoa(5),
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
