package bridge

import (
	"fmt"
	"math/big"
	"path"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/bridge/common"
	bridgeHelper "github.com/0xPolygon/polygon-edge/command/bridge/helper"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/Ethernal-Tech/ethgo"
	"github.com/stretchr/testify/require"
)

// TestE2E_BridgeLoad_MultipleDepositBothEnds is an end-to-end test that validates the functionality
// of a bridge under load by performing multiple deposit operations in both directions (external to internal
// and internal to external). The test ensures that deposits are processed correctly and balances are updated
// as expected.
//
// The test performs the following steps:
//   - Generates a set of test accounts and premines them with sufficient funds for withdrawals.
//   - Sets up a test cluster with a single bridge and a relayer private key.
//   - Deploys a RootERC20 token contract on the external chain.
//   - Performs deposits from the external chain to the internal chain for all test accounts and verifies
//     that the balances on the internal chain match the expected deposit amounts.
//   - Performs deposits from the internal chain to the external chain for all test accounts and verifies
//     that the balances on the external chain match the expected deposit amounts.
//   - Continuously restarts the bridge relayer during the test to simulate real-world scenarios and ensure
//     robustness.
//   - Waits for all deposits to be processed and verifies the success of the bridge events.
//
// The test uses channels to handle errors from concurrent deposit operations and ensures that all operations
// complete successfully within a specified timeout. If any operation fails or the timeout is reached, the test
// fails.
func TestE2E_BridgeLoad_MultipleDepositBothEnds(t *testing.T) {
	const (
		transfersCount  = 10
		epochSize       = 10
		sprintSize      = uint64(5)
		numberOfBridges = 1
	)

	var (
		bridgeAmount        = ethgo.Ether(2)
		bridgeMessageResult contractsapi.BridgeMessageResultEvent
	)

	accountAddrs := make([]types.Address, transfersCount)
	accounts := make([]string, transfersCount)
	amounts := make([]string, transfersCount)
	accountKeys := make([]string, transfersCount)

	for i := 0; i < transfersCount; i++ {
		key, err := crypto.GenerateECDSAKey()
		require.NoError(t, err)

		rawKey, err := key.MarshallPrivateKey()
		require.NoError(t, err)

		accountKeys[i] = hex.EncodeToString(rawKey)
		accountAddrs[i] = key.Address()
		accounts[i] = key.Address().String()
		amounts[i] = fmt.Sprintf("%d", bridgeAmount)

		t.Logf("Receiver#%d=%s\n", i+1, accounts[i])
	}

	relayerPrivateKey, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithEpochSize(epochSize),
		framework.WithBridges(numberOfBridges),
		framework.WithBridgeBatchThreshold(100),
		framework.WithRelayerPrivateKey(relayerPrivateKey),
		framework.WithSecretsCallback(func(addrs []types.Address, tcc *framework.TestClusterConfig) {
			for i := 0; i < len(addrs); i++ {
				// premine receivers, so that they are able to do withdrawals
				tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))
			}

			tcc.StakeAmounts = append(tcc.StakeAmounts, ethgo.Ether(10))

			tcc.Premine = append(tcc.Premine, accounts...)
			tcc.Premine = append(tcc.Premine, relayerPrivateKey.String())
		}))

	defer cluster.Stop()

	bridgeOne := 0

	cluster.WaitForReady(t)

	polybftCfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	validatorSrv := cluster.Servers[0]

	validatorEndpoint := validatorSrv.JSONRPC()

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	chainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeCfg := polybftCfg.Bridge[chainID.Uint64()]

	internalTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(validatorEndpoint))
	require.NoError(t, err)

	deployerKey, err := bridgeHelper.DecodePrivateKey("")
	require.NoError(t, err)

	deployTx := types.NewTx(types.NewLegacyTx(
		types.WithTo(nil),
		types.WithInput(contractsapi.RootERC20.Bytecode),
	))

	receipt, err := externalChainTxRelayer.SendTransaction(deployTx, deployerKey)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

	rootERC20Token := types.Address(receipt.ContractAddress)

	bridge := cluster.Bridges[bridgeOne]

	channel := make(chan error)

	waitNextBlock := func() error {
		currentBlock, err := validatorEndpoint.BlockNumber()
		if err != nil {
			return err
		}

		if err := cluster.WaitForBlock(currentBlock+1, 30*time.Second); err != nil {
			return err
		}

		return nil
	}

	// deposit external to internal
	go func(channel chan<- error) {
		for i := range accounts {
			if err := bridge.Deposit(
				common.ERC20,
				rootERC20Token,
				bridgeCfg.ExternalERC20PredicateAddr,
				bridgeHelper.TestAccountPrivKey,
				accounts[i],
				amounts[i],
				"",
				bridge.JSONRPCAddr(),
				bridgeHelper.TestAccountPrivKey,
				false,
			); err != nil {
				channel <- err

				return
			}

			t.Log("deposit made for account=", accounts[i], "external to internal")

			if err = waitNextBlock(); err != nil {
				channel <- err

				return
			}
		}

		finalBlockNum, err := validatorEndpoint.BlockNumber()
		if err != nil {
			channel <- err

			return
		}

		finalBlockNum += 3 * epochSize
		if err := cluster.WaitForBlock(finalBlockNum, time.Minute*2); err != nil {
			channel <- err

			return
		}

		// the bridge transactions are processed and there should be a success state sync events
		logs, err := getFilteredLogs(bridgeMessageResult.Sig(), 0, finalBlockNum, validatorEndpoint)
		if err != nil {
			channel <- err

			return
		}

		// assert that all deposits are executed successfully
		// because of the token mapping with the first deposit
		assertBridgeEventResultSuccess(t, logs, transfersCount+1)

		childERC20Token := getChildToken(t, contractsapi.RootERC20Predicate.Abi,
			bridgeCfg.ExternalERC20PredicateAddr, rootERC20Token, externalChainTxRelayer)

		for _, receiver := range accounts {
			balance := erc20BalanceOf(t, types.StringToAddress(receiver), childERC20Token, internalTxRelayer)
			t.Log("balance=", balance, "receiver=", receiver)

			if balance.Cmp(bridgeAmount) != 0 {
				channel <- fmt.Errorf("balance=%d, expected=%d", balance, bridgeAmount)

				return
			}
		}

		channel <- nil
	}(channel)

	// deposit internal to external
	go func(channel chan<- error) {
		amount := int64(100)
		amountStr := fmt.Sprintf("%d", amount)
		validatorAddr := validatorSrv.JSONRPCAddr()

		for i := range accounts {
			if err := bridge.Deposit(
				common.ERC20,
				contracts.NativeERC20TokenContract,
				bridgeCfg.InternalMintableERC20PredicateAddr,
				accountKeys[i],
				accounts[i],
				amountStr,
				"",
				validatorAddr,
				"",
				false,
			); err != nil {
				channel <- err

				return
			}

			t.Log("deposit made for account=", accounts[i], "internal to external")

			if err = waitNextBlock(); err != nil {
				channel <- err

				return
			}
		}

		if err := cluster.WaitUntil(time.Minute*2, time.Second*2, func() bool {
			for i := uint64(1); i <= transfersCount+1; i++ {
				if !isEventProcessed(t, bridgeCfg.ExternalGatewayAddr, externalChainTxRelayer, i, false) {
					return false
				}
			}

			return true
		}); err != nil {
			channel <- err

			return
		}

		externalChainToken := getChildToken(t, contractsapi.ChildERC20Predicate.Abi,
			bridgeCfg.ExternalMintableERC20PredicateAddr,
			contracts.NativeERC20TokenContract, externalChainTxRelayer)

		for _, acc := range accountAddrs {
			balance := erc20BalanceOf(t, acc, externalChainToken, externalChainTxRelayer)
			if balance.Cmp(big.NewInt(amount)) != 0 {
				channel <- fmt.Errorf("balance=%d, expected=%d", balance, amount)

				return
			}
		}

		channel <- nil
	}(channel)

	timeout := time.After(time.Minute * transfersCount / 2)
	restart := time.NewTicker(time.Second * 15)
	counter := 0

	for {
		select {
		case err = <-channel:
			if err != nil {
				t.Fatal(err)

				return
			} else if counter++; counter == 2 {
				return
			}

		case <-restart.C:
			cluster.BridgeRelayers[0].Stop()
			time.Sleep(time.Second)
			cluster.BridgeRelayers[0].Start()

		case <-timeout:
			t.Fatal("timeout")

			return
		}
	}
}
