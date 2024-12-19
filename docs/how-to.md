## Configure London Hard Fork
In order to enable London Hard Fork and dynamic fee transactions execute `genesis` CLI command with `--burn-contract` flag.

### Example:
```bash
./blade genesis --reward-wallet 0xDEADBEEF --premine 0x0000000000000000000000000000000000000000 --proxy-contracts-admin 0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed --blade-admin 0x61324166B0202DB1E7502924326262274Fa4358F --burn-contract "0:0x0000000000000000000000000000000000000000" --validators /ip4/127.0.0.1/tcp/1478/p2p/16Uiu2HAmMYyzK7c649Tnn6XdqFLP7fpPB2QWdck1Ee9vj5a7Nhg8:0x61324166B0202DB1E7502924326262274Fa4358F:06d8d9e6af67c28e85ac400b72c2e635e83234f8a380865e050a206554049a222c4792120d84977a6ca669df56ff3a1cf1cfeccddb650e7aacff4ed6c1d4e37b055858209f80117b3c0a6e7a28e456d4caf2270f430f9df2ba37221f23e9bbd313c9ef488e1849cc5c40d18284d019dde5ed86770309b9c24b70ceff6167a6ca
```

### Dynamic Fee Transactions​

Transaction fees for non-atomic transactions are based on Ethereum's EIP-1559 style Dynamic Fee Transactions, which consists of a gas fee cap and a gas tip cap.

The fee cap specifies the maximum price the transaction is willing to pay per unit of gas. The tip cap (also called the priority fee) specifies the maximum amount above the base fee that the transaction is willing to pay per unit of gas. Therefore, the effective gas price paid by a transaction will be min(gasFeeCap, baseFee + gasTipCap). Unlike in Ethereum, where the priority fee is paid to the miner that produces the block, in Avalanche both the base fee and the priority fee are burned. For legacy transactions, which only specify a single gas price, the gas price serves as both the gas fee cap and the gas tip cap.

Use the eth_baseFee API method to estimate the base fee for the next block. If more blocks are produced in between the time that you construct your transaction and it is included in a block, the base fee could be different from the base fee estimated by the API call, so it is important to treat this value as an estimate.

Next, use eth_maxPriorityFeePerGas API call to estimate the priority fee needed to be included in a block. This API call will look at the most recent blocks and see what tips have been paid by recent transactions in order to be included in the block.

Transactions are ordered by the priority fee, then the timestamp (oldest first).

Based off of this information, you can specify the gasFeeCap and gasTipCap to your liking based on how you prioritize getting your transaction included as quickly as possible vs. minimizing the price paid per unit of gas.

### Base Fee​

The base fee can go as low as 1 nAVAX (Gwei) and has no upper bound. You can use the eth_baseFee and eth_maxPriorityFeePerGas API methods, or Snowtrace's C-Chain Gas Tracker, to estimate the gas price to use in your transactions.

<br>

## Create and send transactions

You can send signed transactions using the `eth_sendRawTransaction` JSON RPC API method.

Signed transactions can be simple value transfers, contract creation, or contract invocation. Use client libraries to create and send a signed raw transaction to transfer Ether and create a smart contract.

### eth_call vs eth_sendRawTransaction

You can interact with contracts using `eth_call` or `eth_sendRawTransaction`. The table below compares the characteristics of both calls.

| eth_call | eth_sendRawTransaction |
| :-------- | :---------- |
| Read-only	| Write |
| Invokes contract function locally | Broadcasts to the network |
| Does not change state of blockchain | Updates the blockchain (for example, transfers ether between accounts) |
| Does not consume gas | Requires gas |
| Synchronous | Asynchronous |
| Returns the value of a contract function available immediately | Returns transaction hash only. A block might not include all possible transactions (for example, if the gas price is too low). |

<br>

## Configure Blade for maximal throughput

### genesis CLI command

Increase gas limit with `--block-gas-limit` flag, set it to 200 M or even higher value. Optionaly decrease block time to 1s with flag `--block-time`.

### server CLI command

Start server with following flags and values (this is just an example, you can use different values depending on the use case):
```bash
--max-enqueued 2000000
--max-slots 2000000
--gossip-msg-size 8388608
--tx-gossip-batch-size 10000
```

For more details about flags check server CLI command documentation.