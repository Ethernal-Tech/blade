package deploy

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/0xPolygon/polygon-edge/command/helper"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
)

const (
	deployerKeyFlag          = "deployer-key"
	jsonRPCFlag              = "json-rpc"
	erc20AddrFlag            = "erc20-token"
	isBootstrapFlag          = "bootstrap"
	externalRPCFlag          = "external-json-rpc"
	internalRPCFlag          = "internal-json-rpc"
	bridgeBatchThresholdFlag = "batch-threshold"
)

type deployParams struct {
	genesisPath          string
	deployerKey          string
	externalRPCAddress   string
	internalRPCAddress   string
	rootERC20TokenAddr   string
	proxyContractsAdmin  string
	txTimeout            time.Duration
	isTestMode           bool
	isBootstrap          bool
	bridgeBatchThreshold uint64
}

func (ip *deployParams) validateFlags() error {
	var err error

	if _, err = os.Stat(ip.genesisPath); err != nil {
		return fmt.Errorf("provided genesis path '%s' is invalid. Error: %w ", ip.genesisPath, err)
	}

	consensusCfg, err = polycfg.LoadPolyBFTConfig(ip.genesisPath)
	if err != nil {
		return err
	}

	if consensusCfg.NativeTokenConfig == nil {
		return errors.New("native token configuration is undefined")
	}

	// when using mintable native token, child native token on root chain gets mapped automatically
	if consensusCfg.NativeTokenConfig.IsMintable && ip.rootERC20TokenAddr != "" {
		return errors.New("if child chain native token is mintable, root native token must not pre-exist on root chain")
	}

	return helper.ValidateProxyContractsAdmin(ip.proxyContractsAdmin)
}
