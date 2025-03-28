package bridge

import (
	"fmt"
	"math/big"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/bridge/common"
	polycfg "github.com/0xPolygon/polygon-edge/consensus/polybft/config"
	"github.com/0xPolygon/polygon-edge/consensus/polybft/contractsapi"
	"github.com/0xPolygon/polygon-edge/contracts"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/e2e-polybft/framework"
	helperCommon "github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/jsonrpc"
	"github.com/0xPolygon/polygon-edge/txrelayer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/stretchr/testify/require"
)

// This load test is designed to evaluate the performance and reliability of a blockchain bridge
// by simulating high-volume transactions involving multiple token standards. The test covers the
// transfer of ERC-20, ERC-721, ERC-1155, and native tokens between an internal and an external
// blockchain. By executing these transfers in parallel, the test aims to assess system scalability,
// latency, and potential bottlenecks under stress conditions.
//
// The test is divided into two distinct phases:
// 1. Initialization Phase
//   - This phase is responsible for setting up the necessary conditions before the actual
//     load test begins.
//   - Initial states for all participating accounts are set to ensure sufficient tokens for
//     all intended transfers.
//   - Required smart contracts are deployed, and accounts are pre-funded with native tokens
//     to cover gas fees.
//
// 2. Execution Phase (Load Testing)
//   - This is the core phase where the actual load test occurs.
//   - Multiple users initiate concurrent token transfers, sending assets from the internal
//     to the external chain and vice versa.
//   - The transactions are executed in a structured manner, ensuring an even distribution
//     across different token standards.
//
// The test is controlled through two primary parameters:
// 1. Number of Users (const numberOfUsers)
//   - This constant defines the number of users that will participate in the test for each
//     token type and direction.
//   - Each token type (ERC-20, ERC-721, ERC-1155, and native) will have an equal number of
//     users transferring tokens in both directions (internal to external and vice verse).
//   - For example, if `numberOfUsers` is set to 4, then 4 users will send tokens for each
//     token type and direction, thus 8 per token. Since we have 4 token types, the total
//     number of distinct users participating in the test will be 4 * 2 * 4 = 32.
//
// 2. Number of Transfers per User (numberOfTransfers)
//   - This constant determines how many individual transfers each user will execute.
//   - For example, if `numberOfTransfers“ is set to 3, then each of the 32 users will make
//     3 transfers. The total number of bridge transfers executed during the test (excluding
//     those from the initialization phase) will be 32 * 3 = 96.
//
// The test is designed to maximize concurrency by ensuring all users send transactions in parallel,
// creating a realistic high-load scenario. The time required for transactions to be processed on the
// destination chain is a critical metric for evaluating the bridge's efficiency. Therefore this time
// is measured and displayed as "TOTAL EXECUTION TIME". The test has high coverage in terms of error
// checking, particularly when it comes to verifying proper token transfers, that is, checking the
// balance of all accounts. The test helps determine how well the bridge scales under increasing load,
// including potential optimizations for future improvements.
func TestE2E_Bridge_Load(t *testing.T) {
	const (
		numberOfUsers     = 2
		numberOfTransfers = 2
	)

	// The initial number of bridging events for all token types is increased by 1, because the
	// first event represents the deployment of the child smart contract. To understand why the
	// `initNumOfERC721Events` is set this way, read the ERC721 initialization below.
	initNumOfERC20Events := numberOfUsers + 1
	initNumOfERC721Events := numberOfUsers*numberOfTransfers + 1
	initNumOfERC1155Events := numberOfUsers + 1
	initNumOfNatTokEvents := numberOfUsers + 1
	totalInitEvents := initNumOfERC20Events +
		initNumOfERC721Events +
		initNumOfERC1155Events +
		initNumOfNatTokEvents

	e2iNum := 4 * numberOfUsers * numberOfTransfers
	i2eNum := totalInitEvents + (4 * numberOfUsers * numberOfTransfers)

	t.Logf("Number of users (accounts) per token and direction: %d", numberOfUsers)
	t.Logf("Number of users (accounts) per token: %d", numberOfUsers*2)
	t.Logf("Number of users (accounts) per direction: %d", numberOfUsers*4)
	t.Logf("Total number of users (accounts): %d", numberOfUsers*8)
	t.Logf("Number of bridge messages per user (account): %d\n\n", numberOfTransfers)

	t.Logf("Total number of bridge messages: %d", i2eNum+e2iNum)
	t.Logf("\t- number of initialization messages: %d", totalInitEvents)
	t.Logf("\t- number of execution messages: %d", i2eNum+e2iNum-totalInitEvents)
	t.Logf("\t- number of I2E execution messages: %d", i2eNum-totalInitEvents)
	t.Logf("\t- number of E2I execution messages: %d\n\n", e2iNum)

	var allAddresses []string

	genAccFn := func(accounts []*crypto.ECDSAKey) {
		for i := range numberOfUsers {
			ecdsaKey, err := crypto.GenerateECDSAKey()
			require.NoError(t, err)

			accounts[i] = ecdsaKey

			allAddresses = append(allAddresses, accounts[i].Address().String())
		}
	}

	i2eERC20Accounts := make([]*crypto.ECDSAKey, numberOfUsers)
	e2iERC20Accounts := make([]*crypto.ECDSAKey, numberOfUsers)

	genAccFn(i2eERC20Accounts)

	t.Logf("%d I2E ERC20 accounts created", numberOfUsers)

	genAccFn(e2iERC20Accounts)

	t.Logf("%d E2I ERC20 accounts created", numberOfUsers)

	i2eERC721Accounts := make([]*crypto.ECDSAKey, numberOfUsers)
	e2iERC721Accounts := make([]*crypto.ECDSAKey, numberOfUsers)

	genAccFn(i2eERC721Accounts)

	t.Logf("%d I2E ERC721 accounts created", numberOfUsers)

	genAccFn(e2iERC721Accounts)

	t.Logf("%d E2I ERC721 accounts created", numberOfUsers)

	i2eERC1155Accounts := make([]*crypto.ECDSAKey, numberOfUsers)
	e2iERC1155Accounts := make([]*crypto.ECDSAKey, numberOfUsers)

	genAccFn(i2eERC1155Accounts)

	t.Logf("%d I2E ERC1155 accounts created", numberOfUsers)

	genAccFn(e2iERC1155Accounts)

	t.Logf("%d E2I ERC1155 accounts created", numberOfUsers)

	i2eNatTokAccounts := make([]*crypto.ECDSAKey, numberOfUsers)
	e2iNatTokAccounts := make([]*crypto.ECDSAKey, numberOfUsers)

	genAccFn(i2eNatTokAccounts)

	t.Logf("%d I2E Native token accounts created", numberOfUsers)

	genAccFn(e2iNatTokAccounts)

	t.Logf("%d E2I Native token accounts created", numberOfUsers)

	deployerKeyERC20, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)
	deployerKeyERC721, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)
	deployerKeyERC1155, err := crypto.GenerateECDSAKey()
	require.NoError(t, err)

	cluster := framework.NewTestCluster(t, 5,
		framework.WithTestRewardToken(),
		framework.WithNumBlockConfirmations(10),
		framework.WithEpochSize(20),
		framework.WithBridges(1),
		framework.WithBlockGasLimit(200_000_000),
		framework.WithSecretsCallback(func(_ []types.Address, tcc *framework.TestClusterConfig) {
			tcc.Premine = append(tcc.Premine, allAddresses...)
			tcc.Premine = append(tcc.Premine, deployerKeyERC20.Address().String(),
				deployerKeyERC721.Address().String(),
				deployerKeyERC1155.Address().String())
		}))

	defer cluster.Stop()

	cluster.WaitForReady(t)

	cfg, err := polycfg.LoadPolyBFTConfig(path.Join(cluster.Config.TmpDir, command.DefaultGenesisFileName))
	require.NoError(t, err)

	externalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(cluster.Bridges[0].JSONRPCAddr()))
	require.NoError(t, err)

	externalChainID, err := externalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	internalChainTxRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithClient(cluster.Servers[0].JSONRPC()))
	require.NoError(t, err)

	internalChainID, err := internalChainTxRelayer.Client().ChainID()
	require.NoError(t, err)

	bridgeConfig := cfg.Bridge[externalChainID.Uint64()]

	// Function for deploying the root (ERC20, ERC721 and ERC1155) smart contract.
	deployRootFn := func(contract []byte, deployerKey *crypto.ECDSAKey) types.Address {
		tx := types.NewTx(types.NewLegacyTx(
			types.WithTo(nil),
			types.WithInput(contract),
		))

		receipt, err := internalChainTxRelayer.SendTransaction(tx, deployerKey)
		require.NoError(t, err)
		require.NotNil(t, receipt)
		require.Equal(t, uint64(types.ReceiptSuccess), receipt.Status)

		return types.Address(receipt.ContractAddress)
	}

	var (
		rootERC20Token   types.Address
		rootERC721Token  types.Address
		rootERC1155Token types.Address
		rootNativeToken  types.Address
	)

	wg := sync.WaitGroup{}

	wg.Add(4)

	timer := time.Now().UTC()

	// ERC20 initialization
	//
	// The following goroutine ensures the setup of initial ERC20 balances for accounts testing
	// this kind of token. To achieve it, the root ERC20 smart contract is first deployed on the
	// internal (Blade) chain. Afterward, for each account that will later, during the execution
	// phase of the test, transfer tokens between the internal and external chain and vice versa,
	// minting is performed. Additionally, for accounts transferring tokens from the external to
	// the internal chain, the process includes bridging tokens to the external chain (deposit),
	// specifically to the child ERC20 smart contract. The deployment of the child smart contract
	// is performed automatically before the first bridging operation.
	go func() {
		defer wg.Done()

		rootERC20Token = deployRootFn(contractsapi.RootERC20.Bytecode, deployerKeyERC20)
		t.Logf("Root ERC20 smart contract was successfully deployed on the internal chain %d at address %s",
			internalChainID, rootERC20Token.String())

		pk, err := deployerKeyERC20.MarshallPrivateKey()
		require.NoError(t, err)

		deployer := hex.EncodeToString(pk)

		for i := range numberOfUsers {
			require.NoError(t,
				cluster.Bridges[0].Deposit(
					common.ERC20,
					rootERC20Token,
					bridgeConfig.InternalMintableERC20PredicateAddr,
					deployer,
					e2iERC20Accounts[i].Address().String(),
					"1000000",
					"",
					cluster.Servers[0].JSONRPCAddr(),
					deployer,
					true,
				))

			require.NoError(t,
				cluster.Bridges[0].Mint(
					common.ERC20,
					rootERC20Token,
					i2eERC20Accounts[i].Address().String(),
					"",
					"1000000",
					cluster.Servers[0].JSONRPCAddr(),
					deployer,
				))
		}
	}()

	// ERC721 initialization
	//
	// The following goroutine ensures the setup of initial state for accounts testing ERC721
	// token. To achieve it, the root ERC721 smart contract is first deployed on the internal
	// (Blade) chain. Afterward, for each account that will later, during the execution phase
	// of the test, transfer tokens between the internal and external chain (and vice versa),
	// minting is performed. The number of tokens minted for each account matches the number
	// of bridge transfers that the account will perform later during the execution phase. For
	// example, if the `numberOfTransfers` is set to 5, then 5 minting operation is performed
	// for each account. Additionally, for accounts transferring tokens from the external to
	// the internal chain, the process includes bridging tokens to the external chain (deposit),
	// specifically to the child ERC721 smart contract. The deployment of this smart contract
	// is performed automatically before the first bridging operation.
	go func() {
		defer wg.Done()

		rootERC721Token = deployRootFn(contractsapi.RootERC721.Bytecode, deployerKeyERC721)
		t.Logf("Root ERC721 smart contract was successfully deployed on the internal chain %d at address %s",
			internalChainID, rootERC721Token.String())

		pk, err := deployerKeyERC721.MarshallPrivateKey()
		require.NoError(t, err)

		deployer := hex.EncodeToString(pk)

		var idCounter int64

		for i := range numberOfUsers {
			for range numberOfTransfers {
				require.NoError(t,
					cluster.Bridges[0].Mint(
						common.ERC721,
						rootERC721Token,
						i2eERC721Accounts[i].Address().String(),
						"",
						"",
						cluster.Servers[0].JSONRPCAddr(),
						deployer,
					))

				idCounter++
			}
		}

		for i := range numberOfUsers {
			for range numberOfTransfers {
				require.NoError(t,
					cluster.Bridges[0].Deposit(
						common.ERC721,
						rootERC721Token,
						bridgeConfig.InternalMintableERC721PredicateAddr,
						deployer,
						e2iERC721Accounts[i].Address().String(),
						"",
						fmt.Sprintf("%d", idCounter),
						cluster.Servers[0].JSONRPCAddr(),
						deployer,
						true,
					))

				idCounter++
			}
		}
	}()

	// ERC1155 initialization
	//
	// The following goroutine ensures the setup of initial ERC1155 balances for accounts testing
	// this kind of token. To achieve it, the root ERC1155 smart contract is first deployed on the
	// internal (Blade) chain. Afterward, for each account that will later, during the execution
	// phase of the test, transfer tokens between the internal and external chain and vice versa,
	// minting is performed. Additionally, for accounts transferring tokens from the external to
	// the internal chain, the process includes bridging tokens to the external chain (deposit),
	// specifically to the child ERC1155 smart contract. The deployment of this smart contract is
	// performed automatically before the first bridging operation.
	go func() {
		defer wg.Done()

		rootERC1155Token = deployRootFn(contractsapi.RootERC1155.Bytecode, deployerKeyERC1155)
		t.Logf("Root ERC1155 smart contract was successfully deployed on the internal chain %d at address %s",
			internalChainID, rootERC1155Token.String())

		pk, err := deployerKeyERC1155.MarshallPrivateKey()
		require.NoError(t, err)

		deployer := hex.EncodeToString(pk)

		for i := range numberOfUsers {
			require.NoError(t,
				cluster.Bridges[0].Deposit(
					common.ERC1155,
					rootERC1155Token,
					bridgeConfig.InternalMintableERC1155PredicateAddr,
					deployer,
					e2iERC1155Accounts[i].Address().String(),
					"1000000",
					"20",
					cluster.Servers[0].JSONRPCAddr(),
					deployer,
					true,
				))

			require.NoError(t,
				cluster.Bridges[0].Mint(
					common.ERC1155,
					rootERC1155Token,
					i2eERC1155Accounts[i].Address().String(),
					"20",
					"1000000",
					cluster.Servers[0].JSONRPCAddr(),
					deployer,
				))
		}
	}()

	initialBalance, err := internalChainTxRelayer.Client().GetBalance(i2eNatTokAccounts[0].Address(), jsonrpc.LatestBlockNumberOrHash)
	require.NoError(t, err)

	// Native token initialization
	//
	// The following goroutine ensures the setup of initial balances for accounts testing native
	// tokens. To achieve it, the root smart contract does not need to be deployed in this case,
	// since it is predeployed. Also, we don't need to perform any kind of minting, as it is done
	// during the setup of the cluster. For each account that will later, during execution phase,
	// transfer tokens from the external to the internal chain, bridging of tokens to an external
	// chain (deposit) is performed, specifically to the child smart contract. The deployment of
	// this smart contract is performed automatically before the first bridging operation.
	go func() {
		defer wg.Done()

		rootNativeToken = contracts.NativeERC20TokenContract
		t.Logf("Root Native token smart contract was successfully \"deployed\" on the internal chain %d at address %s",
			internalChainID, rootNativeToken.String())

		for i := range numberOfUsers {
			pk, err := e2iNatTokAccounts[i].MarshallPrivateKey()
			require.NoError(t, err)

			sender := hex.EncodeToString(pk)

			require.NoError(t,
				cluster.Bridges[0].Deposit(
					common.ERC20,
					rootNativeToken,
					bridgeConfig.InternalMintableERC20PredicateAddr,
					sender,
					e2iNatTokAccounts[i].Address().String(),
					"1000000",
					"",
					cluster.Servers[0].JSONRPCAddr(),
					"",
					true,
				))
		}
	}()

	wg.Wait()

	// In the initialization phase, we only wait for events on the external chain to be processed.
	require.NoError(t, cluster.WaitUntil(time.Minute*100, time.Second*2, func() bool {
		for i := range totalInitEvents {
			if !isEventProcessed(t,
				bridgeConfig.ExternalGatewayAddr,
				externalChainTxRelayer,
				uint64(i+1),
				false) {
				return false
			}
		}

		return true
	}))

	childERC20Token := getChildToken(t,
		contractsapi.RootERC20Predicate.Abi,
		bridgeConfig.InternalMintableERC20PredicateAddr,
		rootERC20Token,
		internalChainTxRelayer)

	t.Logf("Child ERC20 smart contract was successfully deployed on the external chain at address %s",
		childERC20Token.String())

	// Function for checking the balance of the ERC20 token.
	erc20CheckFn := func(
		account *crypto.ECDSAKey,
		validBalanceInt,
		validBalanceEx *big.Int) {
		balance := erc20BalanceOf(t, account.Address(), childERC20Token, externalChainTxRelayer)

		require.Equal(t, validBalanceEx, balance)

		balance = erc20BalanceOf(t, account.Address(), rootERC20Token, internalChainTxRelayer)

		require.Equal(t, validBalanceInt, balance)
	}

	// ERC20 balance check, external -> internal
	for _, account := range e2iERC20Accounts {
		erc20CheckFn(account, big.NewInt(0), big.NewInt(1000000))
	}

	// ERC20 balance check, internal -> external
	for _, account := range i2eERC20Accounts {
		erc20CheckFn(account, big.NewInt(1000000), big.NewInt(0))
	}

	childERC721Token := getChildToken(t,
		contractsapi.RootERC721Predicate.Abi,
		bridgeConfig.InternalMintableERC721PredicateAddr,
		rootERC721Token,
		internalChainTxRelayer)

	t.Logf("Child ERC721 smart contract was successfully deployed on the external chain at address %s",
		childERC721Token.String())

	var idCounter int64

	// Function for checking the balance of the ERC721 token.
	erc721CheckFn := func(
		account *crypto.ECDSAKey,
		relayer txrelayer.TxRelayer,
		contract types.Address) {
		for range numberOfTransfers {
			owner := erc721OwnerOf(t, big.NewInt(idCounter), contract, relayer)

			require.Equal(t, account.Address(), owner)

			idCounter++
		}
	}

	// ERC721 balance check, internal -> external
	for _, account := range i2eERC721Accounts {
		erc721CheckFn(account, internalChainTxRelayer, rootERC721Token)
	}

	// ERC721 balance check, external -> internal
	for _, account := range e2iERC721Accounts {
		erc721CheckFn(account, externalChainTxRelayer, childERC721Token)
	}

	childERC1155Token := getChildToken(t, contractsapi.RootERC1155Predicate.Abi,
		bridgeConfig.InternalMintableERC1155PredicateAddr, rootERC1155Token, internalChainTxRelayer)

	t.Logf("Child ERC1155 smart contract was successfully deployed on the external chain at address %s",
		childERC1155Token.String())

	// Function for checking the balance of the ERC1155 token.
	erc1155CheckFn := func(
		account *crypto.ECDSAKey,
		relayer txrelayer.TxRelayer,
		contract types.Address,
		vb *big.Int) {
		balanceOfFn := &contractsapi.BalanceOfChildERC1155Fn{
			Account: account.Address(),
			ID:      big.NewInt(20),
		}

		balanceInput, err := balanceOfFn.EncodeAbi()
		require.NoError(t, err)

		balanceRaw, err := relayer.Call(types.ZeroAddress, contract, balanceInput)
		require.NoError(t, err)

		balance, err := helperCommon.ParseUint256orHex(&balanceRaw)
		require.NoError(t, err)

		require.Equal(t, vb, balance)
	}

	// ERC1155 balance check, internal -> external
	for _, account := range i2eERC1155Accounts {
		erc1155CheckFn(account, internalChainTxRelayer, rootERC1155Token, big.NewInt(1000000))
	}

	// ERC1155 balance check, external -> internal
	for _, account := range e2iERC1155Accounts {
		erc1155CheckFn(account, externalChainTxRelayer, childERC1155Token, big.NewInt(1000000))
	}

	childNativeToken := getChildToken(t, contractsapi.RootERC20Predicate.Abi,
		bridgeConfig.InternalMintableERC20PredicateAddr, rootNativeToken, internalChainTxRelayer)

	t.Logf("Child Native token smart contract was successfully deployed on the external chain at address %s",
		childERC20Token.String())

	postInitI2EInternalBalance := initialBalance
	postInitI2EExternalBalance := big.NewInt(0)

	transferred, _ := new(big.Int).SetString("1000000", 10)
	postInitE2IInternalBalance := big.NewInt(0).Sub(initialBalance, transferred)
	postInitE2IExternalBalance := transferred

	// Function for checking the balance of the native token.
	natTokCheckFn := func(
		account *crypto.ECDSAKey,
		validBalanceInt,
		validBalanceEx *big.Int) {
		balance := erc20BalanceOf(t, account.Address(), childNativeToken, externalChainTxRelayer)

		require.Equal(t, validBalanceEx, balance)

		balance, err := internalChainTxRelayer.Client().GetBalance(account.Address(), jsonrpc.LatestBlockNumberOrHash)
		require.NoError(t, err)

		require.Equal(t, validBalanceInt, balance)
	}

	// Function for ERC721 token ID calculation
	erc721TokenIDFn := func(tokenId string, j int) string {
		id, err := strconv.Atoi(tokenId)
		require.NoError(t, err)

		return fmt.Sprint(id + j)
	}

	// Function for ERC20, ERC721, ERC1155 and native token transfers
	ercTokenTransferFn := func(
		i2eAccount, e2iAccount *crypto.ECDSAKey,
		rootERC20Token, childERC20Token, intMintableAddr, extMintableAddr types.Address,
		tokenType common.TokenType,
		i2eTokenID, e2iTokenID string) {
		// starting 2 go routines
		wg.Add(2)

		// internal -> external
		go func() {
			defer wg.Done()

			sender, err := i2eAccount.MarshallPrivateKey()
			require.NoError(t, err)

			for j := range numberOfTransfers {
				tokenID := i2eTokenID
				if tokenType == common.ERC721 {
					tokenID = erc721TokenIDFn(tokenID, j)
				}

				require.NoError(t,
					cluster.Bridges[0].Deposit(
						tokenType,
						rootERC20Token,
						intMintableAddr,
						hex.EncodeToString(sender),
						i2eAccount.Address().String(),
						"10",
						tokenID,
						cluster.Servers[0].JSONRPCAddr(),
						"",
						false,
					))
			}
		}()

		// external -> internal
		go func() {
			defer wg.Done()

			sender, err := e2iAccount.MarshallPrivateKey()
			require.NoError(t, err)

			for j := range numberOfTransfers {
				tokenID := e2iTokenID
				if tokenType == common.ERC721 {
					tokenID = erc721TokenIDFn(tokenID, j)
				}

				require.NoError(t,
					cluster.Bridges[0].Withdraw(
						tokenType,
						hex.EncodeToString(sender),
						e2iAccount.Address().String(),
						"10",
						tokenID,
						cluster.Bridges[0].JSONRPCAddr(),
						extMintableAddr,
						childERC20Token,
						false))
			}
		}()
	}

	// Native token balance check, external -> internal
	for _, account := range e2iNatTokAccounts {
		natTokCheckFn(account, postInitE2IInternalBalance, postInitE2IExternalBalance)
	}

	// ERC1155 balance check, internal -> external
	for _, account := range i2eNatTokAccounts {
		natTokCheckFn(account, postInitI2EInternalBalance, postInitI2EExternalBalance)
	}

	t.Logf("END OF INITIALIZATION PHASE, TOTAL TIME: %v", time.Now().UTC().Sub(timer))

	timer = time.Now().UTC()

	// ERC20 execution
	for i := range numberOfUsers {
		ercTokenTransferFn(i2eERC20Accounts[i], e2iERC20Accounts[i], rootERC20Token, childERC20Token,
			bridgeConfig.InternalMintableERC20PredicateAddr, bridgeConfig.ExternalMintableERC20PredicateAddr,
			common.ERC20, "", "")
	}

	// ERC721 execution
	for i := range numberOfUsers {
		startI2E := i * numberOfTransfers
		startE2I := i*numberOfTransfers + numberOfUsers*numberOfTransfers

		ercTokenTransferFn(i2eERC721Accounts[i], e2iERC721Accounts[i], rootERC721Token, childERC721Token,
			bridgeConfig.InternalMintableERC721PredicateAddr, bridgeConfig.ExternalMintableERC721PredicateAddr,
			common.ERC721, fmt.Sprintf("%d", startI2E), fmt.Sprintf("%d", startE2I))
	}

	// ERC1155 execution
	for i := range numberOfUsers {
		ercTokenTransferFn(i2eERC1155Accounts[i], e2iERC1155Accounts[i], rootERC1155Token, childERC1155Token,
			bridgeConfig.InternalMintableERC1155PredicateAddr, bridgeConfig.ExternalMintableERC1155PredicateAddr,
			common.ERC1155, "20", "20")
	}

	// Native token execution
	for i := range numberOfUsers {
		ercTokenTransferFn(i2eNatTokAccounts[i], e2iNatTokAccounts[i], rootNativeToken, childNativeToken,
			bridgeConfig.InternalMintableERC20PredicateAddr, bridgeConfig.ExternalMintableERC20PredicateAddr,
			common.ERC20, "", "")
	}

	wg.Wait()

	// Wait for all bridge messages going from the external to the internal chain to be processed.
	require.NoError(t, cluster.WaitUntil(time.Minute*100, time.Second*2, func() bool {
		for i := range e2iNum {
			if !isEventProcessed(t,
				bridgeConfig.InternalGatewayAddr,
				internalChainTxRelayer,
				uint64(i+1),
				false) {
				return false
			}
		}

		return true
	}))

	// Wait for all bridge messages going from the internal to the external chain to be processed.
	require.NoError(t, cluster.WaitUntil(time.Minute*100, time.Second*2, func() bool {
		for i := range i2eNum {
			if !isEventProcessed(t,
				bridgeConfig.ExternalGatewayAddr,
				externalChainTxRelayer,
				uint64(i+1),
				false) {
				return false
			}
		}

		return true
	}))

	const diff = numberOfTransfers * 10

	// ERC20 balance check, internal -> external
	for _, account := range i2eERC20Accounts {
		erc20CheckFn(account, big.NewInt(1000000-diff), big.NewInt(diff))
	}

	// ERC20 balance check, external -> internal
	for _, account := range e2iERC20Accounts {
		erc20CheckFn(account, big.NewInt(diff), big.NewInt(1000000-diff))
	}

	idCounter = 0

	// ERC721 balance check, internal -> external
	for _, account := range i2eERC721Accounts {
		erc721CheckFn(account, externalChainTxRelayer, childERC721Token)
	}

	// ERC721 balance check, external -> internal
	for _, account := range e2iERC721Accounts {
		erc721CheckFn(account, internalChainTxRelayer, rootERC721Token)
	}

	// ERC1155 balance check, internal -> external
	for _, account := range i2eERC1155Accounts {
		erc1155CheckFn(account, externalChainTxRelayer, childERC1155Token, big.NewInt(diff))

		erc1155CheckFn(account, internalChainTxRelayer, rootERC1155Token, big.NewInt(1000000-diff))
	}

	// ERC1155 balance check, external -> internal
	for _, account := range e2iERC1155Accounts {
		erc1155CheckFn(account, internalChainTxRelayer, rootERC1155Token, big.NewInt(diff))

		erc1155CheckFn(account, externalChainTxRelayer, childERC1155Token, big.NewInt(1000000-diff))
	}

	// Native token balance check, internal -> external
	for _, account := range i2eNatTokAccounts {
		natTokCheckFn(account,
			big.NewInt(0).Sub(postInitI2EInternalBalance, big.NewInt(diff)),
			big.NewInt(diff))
	}

	// Native token balance check, external -> internal
	for _, account := range e2iNatTokAccounts {
		natTokCheckFn(account,
			big.NewInt(0).Add(postInitE2IInternalBalance, big.NewInt(diff)),
			big.NewInt(0).Sub(postInitE2IExternalBalance, big.NewInt(diff)))
	}

	t.Logf("END OF EXECUTION PHASE, TOTAL TIME: %v", time.Now().UTC().Sub(timer))
}
