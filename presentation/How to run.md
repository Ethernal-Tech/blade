# Run Blade & Deposit/Withdraw

## Running Blade

In 2 separate terminals run this commands (runs blade with bridge to geth with 101 chainId)

```bash
./scripts/cluster polybft with-logs with-bridge
```

## Prerequirement for deposit/withdraw

```bash
export FOUNDRY_DISABLE_NIGHTLY_WARNING=true

# rpcs
geth_rpc="http://127.0.0.1:8545"
blade_rpc="http://127.0.0.1:10002"

# keys
priv_key="342605fcc263dd89b9ac7d975a4eb084b5ab6f6113dd278a8442ac4a8a8223f7"
pub_key="0xdb74B1a3f34D8a20a99B2d5A52e67697e951fd99"

# contracts
native_erc20="0x0000000000000000000000000000000000000106"
erc20_pred="0x0000000000000000000000000000000000005038"
external_erc20_pred=$(cat genesis.json | grep externalERC20Pred | sed -E 's/.*"externalERC20PredicateAddress": "(0x[0-9a-fA-F]+)".*/\1/')

# functions
approve_func="function approve(address,uint256)"
deposit_func="function deposit(address,uint256)"
withdraw_func="function withdraw(address,uint256)"
src2dest="sourceTokenToDestinationToken(address)(address)"
balance_of="function balanceOf(address)(uint256)"

# codes
mock_erc20="contracts/mocks/MockERC20.sol:MockERC20"

# amounts
one_eth=1000000000000000000000000 # amount in wei, 1 ETH
half_eth=500000000000000000000000 # amount in wei, 0.5 ETH
```

```sh
# Mock ERC20 contract if necessary:

cd blade-contracts
forge create --private-key $priv_key --rpc-url $geth_rpc $mock_erc20
```

## Deposit/Withdraw

### Approve deposit:

```bash
cast send --private-key $priv_key --rpc-url $blade_rpc $native_erc20 $approve_func $erc20_pred $one_eth
```

### Deposit:

```bash
cast send --private-key $priv_key --rpc-url $blade_rpc $erc20_pred $deposit_func $native_erc20 $half_eth
```

### Check token on destination from native ERC20

```bash
external_token_addr=$(cast call $erc20_pred --rpc-url $blade_rpc $src2dest $native_erc20)
echo $external_token_addr

cast call $external_token_addr --rpc-url $geth_rpc $balance_of $pub_key
```

### Withdraw:

```bash
cast send --private-key $priv_key --rpc-url $geth_rpc $external_erc20_pred $withdraw_func $external_token_addr $half_eth
```
