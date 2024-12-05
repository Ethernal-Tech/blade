package framework

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/command"
	bridgeCommon "github.com/0xPolygon/polygon-edge/command/bridge/common"
	bridgeHelper "github.com/0xPolygon/polygon-edge/command/bridge/helper"
	"github.com/0xPolygon/polygon-edge/command/bridge/server"
	"github.com/0xPolygon/polygon-edge/command/genesis"
	cmdHelper "github.com/0xPolygon/polygon-edge/command/helper"
	polybftsecrets "github.com/0xPolygon/polygon-edge/command/secrets/init"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/wallet"
	"github.com/0xPolygon/polygon-edge/types"
	"golang.org/x/sync/errgroup"
)

var initialPortForBridge = uint64(8545)

type TestBridge struct {
	t             *testing.T
	clusterConfig *TestClusterConfig
	id            uint64
	node          *node
}

func NewTestBridge(t *testing.T, clusterConfig *TestClusterConfig, idOfChain uint64) (*TestBridge, error) {
	t.Helper()

	bridge := &TestBridge{
		t:             t,
		clusterConfig: clusterConfig,
		id:            idOfChain,
	}

	err := bridge.Start()
	if err != nil {
		return nil, err
	}

	return bridge, nil
}

func (t *TestBridge) Start() error {
	// Build arguments
	args := []string{
		"bridge",
		"server",
		"--data-dir", t.clusterConfig.Dir(fmt.Sprintf("test-external-chain-%d", t.id)),
		"--chain-id", strconv.FormatUint(t.id, 10),
		"--port", strconv.FormatUint(t.calculatePort(), 10),
	}

	stdout := t.clusterConfig.GetStdout(fmt.Sprintf("bridge-%d", t.id))

	bridgeNode, err := newNode(t.clusterConfig.Binary, args, stdout)
	if err != nil {
		return err
	}

	t.node = bridgeNode

	return server.PingServer(context.Background(), t.calculatePort())
}

func (t *TestBridge) Stop() {
	if err := t.node.Stop(); err != nil {
		t.t.Error(err)
	}

	t.node = nil
}

func (t *TestBridge) JSONRPCAddr() string {
	return fmt.Sprintf("http://%s:%d", hostIP, t.calculatePort())
}

func (t *TestBridge) WaitUntil(pollFrequency, timeout time.Duration, handler func() (bool, error)) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		case <-time.After(pollFrequency):
		}

		isConditionMet, err := handler()
		if err != nil {
			return err
		}

		if isConditionMet {
			return nil
		}
	}
}

// Deposit function invokes bridge deposit of ERC tokens (from the root to the child chain)
// with given receivers, amounts and/or token ids
func (t *TestBridge) Deposit(token bridgeCommon.TokenType, rootTokenAddr, rootPredicateAddr types.Address,
	senderKey, receivers, amounts, tokenIDs, jsonRPCAddr, minterKey string, internalChainMintable bool) error {
	args := []string{}

	if receivers == "" {
		return errors.New("provide at least one receiver address value")
	}

	if jsonRPCAddr == "" {
		return errors.New("provide a JSON RPC endpoint URL")
	}

	switch token {
	case bridgeCommon.ERC20:
		if amounts == "" {
			return errors.New("provide at least one amount value")
		}

		if tokenIDs != "" {
			return errors.New("not expected to provide token ids for ERC 20 deposits")
		}

		args = append(args,
			"bridge",
			"deposit-erc20",
			"--root-token", rootTokenAddr.String(),
			"--root-predicate", rootPredicateAddr.String(),
			"--receivers", receivers,
			"--amounts", amounts,
			"--sender-key", senderKey,
			"--minter-key", minterKey,
			"--json-rpc", jsonRPCAddr)

		if internalChainMintable {
			args = append(args, "--internal-chain-mintable")
		}

	case bridgeCommon.ERC721:
		if tokenIDs == "" {
			return errors.New("provide at least one token id value")
		}

		args = append(args,
			"bridge",
			"deposit-erc721",
			"--root-token", rootTokenAddr.String(),
			"--root-predicate", rootPredicateAddr.String(),
			"--receivers", receivers,
			"--token-ids", tokenIDs,
			"--sender-key", senderKey,
			"--minter-key", minterKey,
			"--json-rpc", jsonRPCAddr)

		if internalChainMintable {
			args = append(args, "--internal-chain-mintable")
		}

	case bridgeCommon.ERC1155:
		if amounts == "" {
			return errors.New("provide at least one amount value")
		}

		if tokenIDs == "" {
			return errors.New("provide at least one token id value")
		}

		args = append(args,
			"bridge",
			"deposit-erc1155",
			"--root-token", rootTokenAddr.String(),
			"--root-predicate", rootPredicateAddr.String(),
			"--receivers", receivers,
			"--amounts", amounts,
			"--token-ids", tokenIDs,
			"--sender-key", senderKey,
			"--minter-key", minterKey,
			"--json-rpc", jsonRPCAddr)

		if internalChainMintable {
			args = append(args, "--internal-chain-mintable")
		}
	}

	return t.cmdRun(args...)
}

// Withdraw function is used to invoke bridge withdrawals for any kind of ERC tokens (from the child to the root chain)
// with given receivers, amounts and/or token ids
func (t *TestBridge) Withdraw(token bridgeCommon.TokenType,
	senderKey, receivers, amounts, tokenIDs, jsonRPCAddr string,
	childPredicate, childToken types.Address, internalChainMintable bool) error {
	if senderKey == "" {
		return errors.New("provide hex-encoded sender private key")
	}

	if receivers == "" {
		return errors.New("provide at least one receiver address value")
	}

	if jsonRPCAddr == "" {
		return errors.New("provide a JSON RPC endpoint URL")
	}

	args := []string{}

	switch token {
	case bridgeCommon.ERC20:
		if amounts == "" {
			return errors.New("provide at least one amount value")
		}

		if tokenIDs != "" {
			return errors.New("not expected to provide token ids for ERC 20 withdrawals")
		}

		args = append(args,
			"bridge",
			"withdraw-erc20",
			"--child-predicate", childPredicate.String(),
			"--child-token", childToken.String(),
			"--sender-key", senderKey,
			"--receivers", receivers,
			"--amounts", amounts,
			"--json-rpc", jsonRPCAddr)

		if internalChainMintable {
			args = append(args, "--internal-chain-mintable")
		}

	case bridgeCommon.ERC721:
		if tokenIDs == "" {
			return errors.New("provide at least one token id value")
		}

		args = append(args,
			"bridge",
			"withdraw-erc721",
			"--child-predicate", childPredicate.String(),
			"--child-token", childToken.String(),
			"--sender-key", senderKey,
			"--receivers", receivers,
			"--token-ids", tokenIDs,
			"--json-rpc", jsonRPCAddr)

		if internalChainMintable {
			args = append(args, "--internal-chain-mintable")
		}

	case bridgeCommon.ERC1155:
		if amounts == "" {
			return errors.New("provide at least one amount value")
		}

		if tokenIDs == "" {
			return errors.New("provide at least one token id value")
		}

		args = append(args,
			"bridge",
			"withdraw-erc1155",
			"--child-predicate", childPredicate.String(),
			"--child-token", childToken.String(),
			"--sender-key", senderKey,
			"--receivers", receivers,
			"--amounts", amounts,
			"--token-ids", tokenIDs,
			"--json-rpc", jsonRPCAddr)

		if internalChainMintable {
			args = append(args, "--internal-chain-mintable")
		}
	}

	return t.cmdRun(args...)
}

// SendExitTransaction sends exit transaction to the root chain
func (t *TestBridge) SendExitTransaction(exitHelper types.Address, exitID uint64, childJSONRPCAddr string) error {
	if childJSONRPCAddr == "" {
		return errors.New("provide a child chain JSON RPC endpoint URL")
	}

	return t.cmdRun(
		"bridge",
		"exit",
		"--exit-helper", exitHelper.String(),
		"--exit-id", strconv.FormatUint(exitID, 10),
		"--root-json-rpc", t.JSONRPCAddr(),
		"--child-json-rpc", childJSONRPCAddr,
	)
}

// cmdRun executes arbitrary command from the given binary
func (t *TestBridge) cmdRun(args ...string) error {
	return runCommand(t.clusterConfig.Binary, args, t.clusterConfig.GetStdout(fmt.Sprintf("bridge-%d", t.id)))
}

// deployExternalChainContracts deploys and initializes external chain contracts
func (t *TestBridge) deployExternalChainContracts(genesisPath string, threshold uint64) error {
	args := []string{
		"bridge",
		"deploy",
		"--proxy-contracts-admin", t.clusterConfig.GetProxyContractsAdmin(),
		"--genesis", genesisPath,
		"--test",
		"--bootstrap",
		"--threshold", strconv.FormatUint(threshold, 10),
	}

	if err := t.cmdRun(args...); err != nil {
		return fmt.Errorf("failed to deploy external chain contracts: %w", err)
	}

	return nil
}

// fundAddressesOnRoot sends predefined amount of tokens to external chain addresses
func (t *TestBridge) fundAddressesOnRoot(polybftConfig polycfg.PolyBFT) error {
	validatorSecrets, err := genesis.GetValidatorKeyFiles(t.clusterConfig.TmpDir, t.clusterConfig.ValidatorPrefix)
	if err != nil {
		return fmt.Errorf("could not get validator secrets on initial external chain funding of genesis validators: %w", err)
	}

	// first fund validators
	balances := make([]*big.Int, len(polybftConfig.InitialValidatorSet))
	secrets := make([]string, len(validatorSecrets))

	for i, secret := range validatorSecrets {
		secrets[i] = path.Join(t.clusterConfig.TmpDir, secret)
		balances[i] = command.DefaultPremineBalance
	}

	if err := t.FundValidators(
		secrets, balances); err != nil {
		return fmt.Errorf("failed to fund validators on the external chain: %w", err)
	}

	// then fund all other addresses so that if token is non-mintable
	// they can do premine on BladeManager
	if len(t.clusterConfig.Premine) == 0 {
		return nil
	}

	// non-validator addresses don't need to mint stake token,
	// they only need to be funded with root token
	args := []string{"bridge", "fund"}

	for _, premineRaw := range t.clusterConfig.Premine {
		premineInfo, err := cmdHelper.ParsePremineInfo(premineRaw)
		if err != nil {
			return err
		}

		args = append(args, "--addresses", premineInfo.Address.String())
		args = append(args, "--amounts", command.DefaultPremineBalance.String()) // this is more than enough tokens
	}

	if err := t.cmdRun(args...); err != nil {
		return fmt.Errorf("failed to fund non-validator addresses on root: %w", err)
	}

	return nil
}

// FundValidators sends tokens to a external chain validators
func (t *TestBridge) FundValidators(secretsPaths []string, amounts []*big.Int) error {
	if len(secretsPaths) != len(amounts) {
		return errors.New("expected the same length of secrets paths and amounts")
	}

	args := []string{"bridge", "fund"}

	for i := 0; i < len(secretsPaths); i++ {
		secretsManager, err := polybftsecrets.GetSecretsManager(secretsPaths[i], "", true)
		if err != nil {
			return err
		}

		key, err := wallet.GetEcdsaFromSecret(secretsManager)
		if err != nil {
			return err
		}

		args = append(args, "--addresses", key.Address().String())
		args = append(args, "--amounts", amounts[i].String())
	}

	if err := t.cmdRun(args...); err != nil {
		return err
	}

	return nil
}

// mintNativeRootToken mints native er20 token on root for provided validators and other accounts in premine flag
func (t *TestBridge) mintNativeRootToken(validatorAddresses []types.Address, tokenConfig *polycfg.Token,
	polybftConfig polycfg.PolyBFT) error {
	if tokenConfig.IsMintable {
		// if token is mintable, it is premined in genesis command,
		// so we just return here
		return nil
	}

	// if token is non-mintable, then to do premine we first need to mint those tokens
	// to validators and other provided addresses
	args := []string{
		"mint-erc20",
		"--jsonrpc", t.JSONRPCAddr(),
		"--erc20-token", polybftConfig.Bridge[tokenConfig.ChainID].ExternalNativeERC20Addr.String(),
	}

	// mint something for every validator
	for _, addr := range validatorAddresses {
		args = append(args, "--addresses", addr.String())
		args = append(args, "--amounts", command.DefaultPremineBalance.String())
	}

	// mint something to others as well
	for _, premineRaw := range t.clusterConfig.Premine {
		premineInfo, err := cmdHelper.ParsePremineInfo(premineRaw)
		if err != nil {
			return err
		}

		if premineInfo.Address == types.ZeroAddress {
			// we can not premine zero address on root
			// so just continue
			continue
		}

		args = append(args, "--addresses", premineInfo.Address.String())
		args = append(args, "--amounts", premineInfo.Amount.String())
	}

	return t.cmdRun(args...)
}

// premineNativeRootToken will premine token on root for every validator and other addresses in premine flag
func (t *TestBridge) premineNativeRootToken(genesisPath string, tokenConfig *polycfg.Token,
	polybftConfig polycfg.PolyBFT) error {
	if tokenConfig.IsMintable {
		// if token is mintable, it is premined in genesis command,
		// so we just return here
		return nil
	}

	bridgeConfig := polybftConfig.Bridge[tokenConfig.ChainID]

	validatorSecrets, err := genesis.GetValidatorKeyFiles(t.clusterConfig.TmpDir, t.clusterConfig.ValidatorPrefix)
	if err != nil {
		return fmt.Errorf("could not get validator secrets on premining native root"+
			" token for genesis validators: %w", err)
	}

	premineCmdArgs := func(secret, key string, premineAmount, stakedAmount *big.Int) error {
		args := []string{
			"bridge",
			"premine",
			"--jsonrpc", t.JSONRPCAddr(),
			"--premine-amount", premineAmount.String(),
			"--stake-amount", stakedAmount.String(),
			"--erc20-token", bridgeConfig.ExternalNativeERC20Addr.String(),
			"--genesis", genesisPath,
		}

		if secret != "" {
			args = append(args, "--"+polybftsecrets.AccountDirFlag, path.Join(t.clusterConfig.TmpDir, secret))
		} else {
			args = append(args, "--private-key", key)
		}

		return t.cmdRun(args...)
	}

	g, ctx := errgroup.WithContext(context.Background())

	// premine validators
	for i, secret := range validatorSecrets {
		i := i
		secret := secret

		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				validatorStake := t.clusterConfig.getStakeAmount(i)
				validatorNonStake := new(big.Int).Set(command.DefaultPremineBalance)
				validatorNonStake = validatorNonStake.Sub(validatorNonStake, validatorStake)

				if err := premineCmdArgs(secret, "", validatorNonStake, validatorStake); err != nil {
					return fmt.Errorf("failed to do premine of native root token for genesis validator: %w",
						err)
				}

				return nil
			}
		})
	}

	// now premine for other addresses
	for _, premineRaw := range t.clusterConfig.Premine {
		premineRaw := premineRaw

		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				premineInfo, err := cmdHelper.ParsePremineInfo(premineRaw)
				if err != nil {
					return fmt.Errorf("failed to do premine of native root token for non-validator"+
						" account: %w. premine raw: %s", err, premineRaw)
				}

				// there is no premine on root for zero address, only on L2
				if premineInfo.Address == types.ZeroAddress {
					return nil
				}

				// non-validator addresses only premine non-staked amounts, no stake premine is allowed
				if err := premineCmdArgs("", premineInfo.Key, premineInfo.Amount, big.NewInt(0)); err != nil {
					return fmt.Errorf("failed to do premine of native root token for "+
						"non-validator account: %w. premine raw: %s", err, premineRaw)
				}

				return nil
			}
		})
	}

	return g.Wait()
}

// calculatePort calculate port for specific bridge
// each bridge uses 3 ports the next starting port is 3 higher
func (t *TestBridge) calculatePort() uint64 {
	return initialPortForBridge + (t.id-1)*3
}

// finalizeGenesis finalizes genesis on BladeManager contract on root
func (t *TestBridge) finalizeGenesis(genesisPath string, tokenConfig *polycfg.Token) error {
	if tokenConfig.IsMintable {
		// we don't need to finalize anything when we have mintable (child originated) token
		return nil
	}

	args := []string{
		"bridge",
		"finalize-bridge",
		"--jsonrpc", t.JSONRPCAddr(),
		"--private-key", bridgeHelper.TestAccountPrivKey,
		"--genesis", genesisPath,
	}

	if err := t.cmdRun(args...); err != nil {
		return fmt.Errorf("failed to finalize genesis on blade manager: %w", err)
	}

	return nil
}
