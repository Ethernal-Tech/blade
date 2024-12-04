package server

import (
	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/consensus"
	consensusDev "github.com/0xPolygon/polygon-edge/consensus/dev"
	consensusDummy "github.com/0xPolygon/polygon-edge/consensus/dummy"
	consensusPolyBFT "github.com/0xPolygon/polygon-edge/consensus/polybft"
	consensusPolyBFTConfig "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/forkmanager"
	"github.com/0xPolygon/polygon-edge/secrets"
	alibabassm "github.com/0xPolygon/polygon-edge/secrets/alibaba"
	"github.com/0xPolygon/polygon-edge/secrets/awsssm"
	"github.com/0xPolygon/polygon-edge/secrets/gcpssm"
	"github.com/0xPolygon/polygon-edge/secrets/hashicorpvault"
	"github.com/0xPolygon/polygon-edge/secrets/local"
	"github.com/0xPolygon/polygon-edge/state"
)

type GenesisFactoryHook func(config *chain.Chain, engineName string) func(*state.Transition) error

type ConsensusType string

type ForkManagerFactory func(forks *chain.Forks) error

type ForkManagerInitialParamsFactory func(config *chain.Chain) (*forkmanager.ForkParams, error)

type IsL1OriginatedTokenCheck func(config *chain.Params) (bool, error)

const (
	DevConsensus     ConsensusType = "dev"
	PolyBFTConsensus ConsensusType = consensusPolyBFTConfig.ConsensusName
	DummyConsensus   ConsensusType = "dummy"
)

var consensusBackends = map[ConsensusType]consensus.Factory{
	DevConsensus:     consensusDev.Factory,
	PolyBFTConsensus: consensusPolyBFT.Factory,
	DummyConsensus:   consensusDummy.Factory,
}

// secretsManagerBackends defines the SecretManager factories for different
// secret management solutions
var secretsManagerBackends = map[secrets.SecretsManagerType]secrets.SecretsManagerFactory{
	secrets.Local:          local.SecretsManagerFactory,
	secrets.HashicorpVault: hashicorpvault.SecretsManagerFactory,
	secrets.AWSSSM:         awsssm.SecretsManagerFactory,
	secrets.GCPSSM:         gcpssm.SecretsManagerFactory,
	secrets.AlibabaSSM:     alibabassm.SecretsManagerFactory,
}

var genesisCreationFactory = map[ConsensusType]GenesisFactoryHook{
	PolyBFTConsensus: consensusPolyBFT.GenesisPostHookFactory,
}

var forkManagerFactory = map[ConsensusType]ForkManagerFactory{
	PolyBFTConsensus: consensusPolyBFT.ForkManagerFactory,
}

var forkManagerInitialParamsFactory = map[ConsensusType]ForkManagerInitialParamsFactory{
	PolyBFTConsensus: consensusPolyBFT.ForkManagerInitialParamsFactory,
}

var isL1OriginatedTokenCheckFactory = map[ConsensusType]IsL1OriginatedTokenCheck{
	PolyBFTConsensus: consensusPolyBFT.IsL1OriginatedTokenCheck,
}

func ConsensusSupported(value string) bool {
	_, ok := consensusBackends[ConsensusType(value)]

	return ok
}
