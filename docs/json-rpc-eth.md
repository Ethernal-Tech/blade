## eth_blockNumber

Returns the number of the most recent block.

### Parameters

None

### Returns

*  <b> QUANTITY </b> - integer of the current block number the client is on.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
````

## eth_call

Executes a new message call immediately without creating a transaction on the blockchain.

### Parameters
<b> Object </b>  - The transaction call object

*  <b>  from: DATA, 20 Bytes </b> - (optional) The address the transaction is sent from.
*  <b>  to: DATA, 20 Bytes </b> - The address the transaction is directed to.
*  <b>  gas: QUANTITY </b> - (optional) Integer of the gas provided for the transaction execution. eth_call consumes zero gas, but this parameter may be needed by some executions.
*  <b>  gasPrice: QUANTITY </b> - (optional) Integer of the gasPrice used for each paid gas.
*  <b>  maxPriorityFeePerGas: QUANTITY </b> - (optional) Maximum fee, in Wei, the sender is willing to pay per gas above the base fee. Can be used only in EIP1559 transactions. If used, must specify maxFeePerGas.
*  <b>  maxFeePerGas: QUANTITY </b> - (optional) Maximum total fee (base fee + priority fee), in Wei, the sender is willing to pay per gas. Can be used only in EIP1559 transactions. If used, must specify maxPriorityFeePerGas.
*  <b>  value: QUANTITY </b> - (optional) Integer of the value sent with this transaction.
*  <b>  data: DATA </b> - (optional) Hash of the method signature and encoded parameters. For details see Ethereum Contract ABI in the Solidity documentation.
*  <b>  nonce: QUANTITY </b> - (optional) Transaction nonce.
*  <b>  type: QUANTITY </b> - (optional) Transaction type.
*  <b>  chainId: QUANTITY </b> - (optional) Chain ID.
*  <b>  accessList: Array </b> - (optional) List of addresses and storage keys that the transaction plans to access. Used only in non-frontier transactions.
*  <b>  QUANTITY|TAG </b> - integer block number, or the string "latest"

### Returns

*  <b>  DATA </b> - the return value of executed contract.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_call","params":[{see above},"latest"],"id":1}'
````

## eth_chainId

Returns the currently configured chain id, a value used in replay-protected transaction signing as introduced by EIP-155.

### Parameters

* None

### Returns

* <b> QUANTITY </b> - big integer of the current chain id.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
````

## eth_createAccessList

Creates an EIP-2930 access list that you can include in a transaction.

### Parameters
<b> Object </b>  - The transaction call object

*  <b>  from: DATA, 20 Bytes </b> - (optional) The address the transaction is sent from.
*  <b>  to: DATA, 20 Bytes </b> - The address the transaction is directed to.
*  <b>  gas: QUANTITY </b> - (optional) Integer of the gas provided for the transaction execution. eth_call consumes zero gas, but this parameter may be needed by some executions.
*  <b>  gasPrice: QUANTITY </b> - (optional) Integer of the gasPrice used for each paid gas.
*  <b>  maxPriorityFeePerGas: QUANTITY </b> - (optional) Maximum fee, in Wei, the sender is willing to pay per gas above the base fee. Can be used only in EIP1559 transactions. If used, must specify maxFeePerGas.
*  <b>  maxFeePerGas: QUANTITY </b> - (optional) Maximum total fee (base fee + priority fee), in Wei, the sender is willing to pay per gas. Can be used only in EIP1559 transactions. If used, must specify maxPriorityFeePerGas.
*  <b>  value: QUANTITY </b> - (optional) Integer of the value sent with this transaction.
*  <b>  data: DATA </b> - (optional) Hash of the method signature and encoded parameters. For details see Ethereum Contract ABI in the Solidity documentation.
*  <b>  nonce: QUANTITY </b> - (optional) Transaction nonce.
*  <b>  type: QUANTITY </b> - (optional) Transaction type.
*  <b>  chainId: QUANTITY </b> - (optional) Chain ID.
*  <b>  accessList: Array </b> - (optional) List of addresses and storage keys that the transaction plans to access. Used only in non-frontier transactions.
*  <b>  QUANTITY|TAG </b> - integer block number, or the string "latest"

### Returns

<b>  Object </b> - access list object with the following fields:
storageKeys: array - storage keys to be accessed by the transaction
*  <b>  accessList: Array of Objects </b> - list of objects with the following fields:
>  * <b> address: DATA, 20 Bytes </b> - address to be accessed by the transaction.
>  * <b> storageKeys: Array </b> - storage keys to be accessed by the transaction.
*  <b>  gasUsed: QUANTITY </b> - approximate gas cost for the transaction if the access list is included.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_createAccessList","params":[{see above},"latest"],"id":1}'
````

## eth_estimateGas

Generates and returns an estimate of how much gas is necessary to allow the transaction to complete. The transaction will not be added to the blockchain. Note that the estimate may be significantly more than the amount of gas actually used by the transaction, for a variety of reasons including EVM mechanics and node performance.

### Parameters

Properties 'to' or 'data' must be provided, while all others are optional.

<b> Object </b>  - The transaction call object

*  <b>  from: DATA, 20 Bytes </b> - the address the transaction is sent from.
*  <b>  to: DATA, 20 Bytes </b> - the address the transaction is directed to.
*  <b>  gas: QUANTITY </b> - integer of the gas provided for the transaction execution. eth_call consumes zero gas, but this parameter may be needed by some executions.
*  <b>  gasPrice: QUANTITY </b> - integer of the gasPrice used for each paid gas.
*  <b>  maxPriorityFeePerGas: QUANTITY </b> - maximum fee, in Wei, the sender is willing to pay per gas above the base fee. Can be used only in EIP1559 transactions. If used, must specify maxFeePerGas.
*  <b>  maxFeePerGas: QUANTITY </b> - maximum total fee (base fee + priority fee), in Wei, the sender is willing to pay per gas. Can be used only in EIP1559 transactions. If used, must specify maxPriorityFeePerGas.
*  <b>  value: QUANTITY </b> - integer of the value sent with this transaction.
*  <b>  data: DATA </b> - hash of the method signature and encoded parameters. For details see Ethereum Contract ABI in the Solidity documentation.
*  <b>  nonce: QUANTITY </b> - transaction nonce.
*  <b>  type: QUANTITY </b> - transaction type.
*  <b>  chainId: QUANTITY </b> - chain ID.
*  <b>  accessList: Array </b> - list of addresses and storage keys that the transaction plans to access. Used only in non-frontier transactions.
*  <b>  QUANTITY|TAG </b>  - integer block number, or the string "latest"

### Returns

*  <b>  QUANTITY </b> - the amount of gas used.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_estimateGas","params":[{see above},"latest"],"id":1}'
````

## eth_feeHistory

Returns base fee per gas and transaction effective priority fee per gas history for the requested block range, allowing you to track trends over time.

As of EIP-4844, this method tracks transaction blob gas fees as well.

### Parameters

*  <b>  blockCount: QUANTITY </b> - number of blocks in the requested range. Between 1 and 1024 blocks can be requested in a single query. If blocks in the specified block range are not available, then only the fee history for available blocks is returned. Accepts hexadecimal or integer values.
*  <b>  newestBlock: QUANTITY|TAG </b> - integer block number, or the string "latest"
*  <b>  rewardPercentiles: QUANTITY </b> - (optional) A monotonically increasing list of decimal percentile values to sample from each block's effective priority fees per gas in ascending order, weighted by gas used.

### Returns

*  <b>  Object </b> - fee history results object.

<b> Object </b>  - A fee history object:
*  <b> oldestBlock: QUANTITY </b> - lowest number block of the returned range.
*  <b> baseFeePerGas: Array </b> - array of block base fees per gas, including an extra block value. The extra value is the next block after the newest block in the returned range. Returns zeroes for blocks created before EIP-1559.
*  <b> gasUsedRatio: Array </b> - array of block gas used ratios. These are calculated as the ratio of gasUsed and gasLimit.
*  <b> reward: Array </b> - array of effective priority fee per gas data points from a single block. All zeroes are returned if the block is empty.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_feeHistory","params":["0x5", "latest", [20,30]],"id":1}'
````

## eth_gasPrice

Returns the current price of gas in wei.
If minimum gas price is enforced by setting the `--price-limit` flag,
this endpoint will return the value defined by this flag as minimum gas price.

### Parameters

None

### Returns

*  <b> QUANTITY </b> - integer of the current gas price in wei.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":1}'
````

## eth_getBalance

Returns the balance of the account of the given address.

### Parameters

*  <b> DATA, 20 Bytes </b> - address to check for balance.
*  <b> QUANTITY|TAG </b> - integer block number, or the string "latest"

### Returns


*  <b> QUANTITY </b> - integer of the current balance in wei.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "latest"],"id":1}'
````

## eth_getBlockByHash

Returns block information by hash.

### Parameters

* <b> DATA , 32 Bytes </b> - Hash of a block.
* <b> Boolean </b> - If true it returns the full transaction objects, if false only the hashes of the transactions.

### Returns

<b> Object </b>  - A block object, or null when no block was found:

*  <b> parentHash: DATA, 32 Bytes </b> - hash of the parent block.
*  <b> sha3Uncles: DATA, 32 Bytes </b> - SHA3 of the uncles data in the block.
*  <b> miner: DATA, 20 Bytes </b> - the address of the beneficiary to whom the mining rewards were given.
*  <b> stateRoot: DATA, 32 Bytes </b> - the root of the final state trie of the block.
*  <b> transactionsRoot: DATA, 32 Bytes </b> - the root of the transaction trie of the block.
*  <b> receiptsRoot: DATA, 32 Bytes </b> - the root of the receipts trie of the block.
*  <b> logsBloom: DATA, 256 Bytes </b>- the bloom filter for the logs of the block.
*  <b> difficulty: QUANTITY </b> - integer of the difficulty for this block.
*  <b> totalDifficulty: QUANTITY </b> - integer of the total difficulty of the chain until this block.
*  <b> number: QUANTITY </b> - the block number.
*  <b> gasLimit: QUANTITY </b> - the maximum gas allowed in this block.
*  <b> gasUsed: QUANTITY </b> - the total used gas by all transactions in this block.
*  <b> timestamp: QUANTITY </b> - the unix timestamp for when the block was collated.
*  <b> extraData: DATA </b> - the “extra data” field of this block.
*  <b> mixHash: DATA, 32 Bytes </b> - "0xadce6e5230abe012342a44e4e9b6d05997d6f015387ae0e59be924afc7ec70c1" represents a hash of "PolyBFT Mix" to identify whether the block is from PolyBFT consensus engine.
*  <b> nonce: DATA, 8 Bytes </b> - hash of the generated proof-of-work.
*  <b> hash: DATA, 32 Bytes </b> - hash of the block.
*  <b> baseFeePerGas: QUANTITY </b> - base fee per gas.
*  <b> size: QUANTITY </b> - integer the size of this block in bytes.
*  <b> transactions: Array </b> - Array of transaction objects, or 32 Bytes transaction hashes depending on the last given parameter.
*  <b> uncles: Array </b> - Array of uncle hashes.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockByHash","params":["0xdc0818cf78f21a8e70579cb46a43643f78291264dda342ae31049421c82d21ae",false],"id":1}'
````

## eth_getBlockByNumber

Returns block information by number.

### Parameters

*  <b>QUANTITY|TAG </b> - integer of a block number, or the string "latest"
*  <b> Boolean </b> - If true it returns the full transaction objects, if false only the hashes of the transactions.

### Returns

Object - A block object, or null when no block was found:

*  <b> parentHash: DATA, 32 Bytes </b> - hash of the parent block.
*  <b> sha3Uncles: DATA, 32 Bytes </b> - SHA3 of the uncles data in the block.
*  <b> miner: DATA, 20 Bytes </b> - the address of the beneficiary to whom the mining rewards were given.
*  <b> stateRoot: DATA, 32 Bytes </b> - the root of the final state trie of the block.
*  <b> transactionsRoot: DATA, 32 Bytes </b> - the root of the transaction trie of the block.
*  <b> receiptsRoot: DATA, 32 Bytes </b> - the root of the receipts trie of the block.
*  <b> logsBloom: DATA, 256 Bytes </b>- the bloom filter for the logs of the block.
*  <b> difficulty: QUANTITY </b> - integer of the difficulty for this block.
*  <b> totalDifficulty: QUANTITY </b> - integer of the total difficulty of the chain until this block.
*  <b> number: QUANTITY </b> - the block number.
*  <b> gasLimit: QUANTITY </b> - the maximum gas allowed in this block.
*  <b> gasUsed: QUANTITY </b> - the total used gas by all transactions in this block.
*  <b> timestamp: QUANTITY </b> - the unix timestamp for when the block was collated.
*  <b> extraData: DATA </b> - the “extra data” field of this block.
*  <b> mixHash: DATA, 32 Bytes </b> - "0xadce6e5230abe012342a44e4e9b6d05997d6f015387ae0e59be924afc7ec70c1" represents a hash of "PolyBFT Mix" to identify whether the block is from PolyBFT consensus engine.
*  <b> nonce: DATA, 8 Bytes </b> - hash of the generated proof-of-work.
*  <b> hash: DATA, 32 Bytes </b> - hash of the block.
*  <b> baseFeePerGas: QUANTITY </b> - base fee per gas.
*  <b> size: QUANTITY </b> - integer the size of this block in bytes.
*  <b> transactions: Array </b> - Array of transaction objects, or 32 Bytes transaction hashes depending on the last given parameter.
*  <b> uncles: Array </b> - Array of uncle hashes.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", true],"id":1}'
````

## eth_getBlockReceipts

Returns all transaction receipts for a given block. Transaction receipts provide a way to track the success or failure of a transaction (1 if successful and 0 if failed), as well as the amount of gas used and any event logs that might have been produced by a smart contract during the transaction.

### Parameters

*  <b>QUANTITY|TAG </b> - integer of a block number, or the string "latest"

### Returns

*  <b> Array of transaction receipt objects </b>  - A transaction receipt objects array, or null when no receipt was found.

<b> Object </b>  - A transaction receipt object:
*  <b> cumulativeGasUsed : QUANTITY </b> - The total amount of gas used when this transaction was executed in the block.
*  <b> logsBloom: DATA, 256 Bytes </b> - Bloom filter for light clients to quickly retrieve related logs.
*  <b> logs: Array </b> - Array of log objects, which this transaction generated.
*  <b> transactionHash : DATA, 32 Bytes </b> - hash of the transaction.
*  <b> transactionIndex: QUANTITY </b> - integer of the transactions index position in the block.
*  <b> blockHash: DATA, 32 Bytes </b> - hash of the block where this transaction was in.
*  <b> blockNumber: QUANTITY </b> - block number where this transaction was in.
*  <b> gasUsed : QUANTITY </b> - The amount of gas used by this specific transaction alone.
*  <b> contractAddress : DATA, 20 Bytes </b> - The contract address created, if the transaction was a contract creation, otherwise null.
*  <b> from: DATA, 20 Bytes </b> - address of the sender.
*  <b> to: DATA, 20 Bytes </b> - address of the receiver. null when its a contract creation transaction.
*  <b>  type: QUANTITY </b> - transaction type.

It also returns either :

*  <b> root  : DATA 32 bytes </b> - post-transaction stateroot (pre Byzantium)
*  <b>status: QUANTITY </b> - either 1 (success) or 0 (failure)

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockReceipts","params":["latest"],"id":1}'
````

## eth_getBlockTransactionCountByHash

Returns the number of transactions in a block matching the specified block hash.

### Parameters

* <b> DATA , 32 Bytes </b> - Hash of a block.

### Returns

*  <b> QUANTITY </b> - integer representing the number of transactions in the specified block, or null if no matching block number is found.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockTransactionCountByHash","params":["0x2168d7d6ce7da245a708a43958535e1fb1b9e076956a785357131538ea37928c"],"id":1}'
````

## eth_getBlockTransactionCountByNumber

Returns the number of transactions in a block matching the specified block number.

### Parameters

*  <b>QUANTITY|TAG </b> - integer of a block number, or the string "latest"

### Returns

*  <b> QUANTITY </b> - integer representing the number of transactions in the specified block, or null if no matching block number is found.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockTransactionCountByNumber","params":["latest"],"id":1}'
````

## eth_getCode

Returns code at a given address.

### Parameters

*  <b>  DATA, 20 Bytes </b> - address
*  <b>  QUANTITY|TAG </b> - integer block number, or the string "latest"

### Returns

*  <b>  DATA </b> - the code from the given address.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getCode","params":["0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b", "0x2"],"id":1}'
````

## eth_getFilterChanges

Polling method for a filter, which returns an array of logs that occurred since the last poll.

### Parameters

*  <b>  QUANTITY </b> - the filter id.

### Returns

<b> Array </b> - Array of log objects, or an empty array if nothing has changed since last poll.

*  For filters created with eth_newBlockFilter the return are block hashes (DATA, 32 Bytes), e.g. ["0x3454645634534..."].
*  For filters created with eth_newFilter logs are objects with the following params:
    * <b> removed: TAG </b> - true when the log was removed, due to a chain reorganization. false if its a valid log.
    * <b> logIndex: QUANTITY </b> - integer of the log index position in the block. null when its pending log.
    * <b> transactionIndex: QUANTITY </b> - integer of the transactions index position log was created from. null when its pending log.
    * <b> transactionHash: DATA, 32 Bytes </b> - hash of the transactions this log was created from. null when its pending log.
    * <b> blockHash: DATA, 32 Bytes </b> - hash of the block where this log was in.  null when its pending log.
    * <b> blockNumber: QUANTITY </b> - the block number where this log was in.  null when its pending log.
    * <b> address: DATA, 20 Bytes </b> - address from which this log originated.
    * <b> data: DATA </b> - contains one or more 32 Bytes non-indexed arguments of the log.
    * <b> topics: Array of DATA </b> - Array of 0 to 4 32 Bytes DATA of indexed log arguments. (In solidity: The first topic is the hash of the signature of the event (e.g. Deposit(address,bytes32,uint256)), except you declared the event with the anonymous specifier.)

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getFilterChanges","params":["0x16"],"id":1}'
````

## eth_getFilterLogs

Returns an array of all logs matching filter with given id.

> **Caution eth_getLogs vs. eth_getFilterLogs**<br>
> These 2 methods will return the same results for same filter options:
> 1. eth_getLogs with params [options]
> 2. eth_newFilter with params [options], getting a [filterId] back, then calling eth_getFilterLogs with [filterId]

### Parameters

*  <b>  QUANTITY </b> - the filter id.

### Returns

<b> Array </b> - Array of log objects, or an empty array

*  For filters created with eth_newFilter logs are objects with the following params:
    * <b> removed: TAG </b> - true when the log was removed, due to a chain reorganization. false if its a valid log.
    * <b> logIndex: QUANTITY </b> - integer of the log index position in the block. null when its pending log.
    * <b> transactionIndex: QUANTITY </b> - integer of the transactions index position log was created from. null when its pending log.
    * <b> transactionHash: DATA, 32 Bytes </b> - hash of the transactions this log was created from. null when its pending log.
    * <b> blockHash: DATA, 32 Bytes </b> - hash of the block where this log was in.  null when its pending log.
    * <b> blockNumber: QUANTITY </b> - the block number where this log was in.  null when its pending log.
    * <b> address: DATA, 20 Bytes </b> - address from which this log originated.
    * <b> data: DATA </b> - contains one or more 32 Bytes non-indexed arguments of the log.
    * <b> topics: Array of DATA </b> - Array of 0 to 4 32 Bytes DATA of indexed log arguments. (In solidity: The first topic is the hash of the signature of the event (e.g. Deposit(address,bytes32,uint256)), except you declared the event with the anonymous specifier.)

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getFilterLogs","params":["0x16"],"id":1}'
````

## eth_getHeaderByHash

Returns header data by block hash.

### Parameters

* <b> DATA , 32 Bytes </b> - Hash of a block.

### Returns

Object - A block header object, or null when no block was found:

*  <b> parentHash: DATA, 32 Bytes </b> - hash of the parent block.
*  <b> sha3Uncles: DATA, 32 Bytes </b> - SHA3 of the uncles data in the block.
*  <b> miner: DATA, 20 Bytes </b> - the address of the beneficiary to whom the mining rewards were given.
*  <b> stateRoot: DATA, 32 Bytes </b> - the root of the final state trie of the block.
*  <b> transactionsRoot: DATA, 32 Bytes </b> - the root of the transaction trie of the block.
*  <b> receiptsRoot: DATA, 32 Bytes </b> - the root of the receipts trie of the block.
*  <b> logsBloom: DATA, 256 Bytes </b>- the bloom filter for the logs of the block.
*  <b> difficulty: QUANTITY </b> - integer of the difficulty for this block.
*  <b> totalDifficulty: QUANTITY </b> - integer of the total difficulty of the chain until this block.
*  <b> number: QUANTITY </b> - the block number.
*  <b> gasLimit: QUANTITY </b> - the maximum gas allowed in this block.
*  <b> gasUsed: QUANTITY </b> - the total used gas by all transactions in this block.
*  <b> timestamp: QUANTITY </b> - the unix timestamp for when the block was collated.
*  <b> extraData: DATA </b> - the “extra data” field of this block.
*  <b> mixHash: DATA, 32 Bytes </b> - "0xadce6e5230abe012342a44e4e9b6d05997d6f015387ae0e59be924afc7ec70c1" represents a hash of "PolyBFT Mix" to identify whether the block is from PolyBFT consensus engine.
*  <b> nonce: DATA, 8 Bytes </b> - hash of the generated proof-of-work.
*  <b> hash: DATA, 32 Bytes </b> - hash of the block.
*  <b> baseFeePerGas: QUANTITY </b> - base fee per gas.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getHeaderByHash","params":["0x2168d7d6ce7da245a708a43958535e1fb1b9e076956a785357131538ea37928c"],"id":1}'
````

## eth_getHeaderByNumber

Returns header data by block number.

### Parameters

*  <b>QUANTITY|TAG </b> - integer of a block number, or the string "latest"

### Returns

Object - A block header object, or null when no block was found:

*  <b> parentHash: DATA, 32 Bytes </b> - hash of the parent block.
*  <b> sha3Uncles: DATA, 32 Bytes </b> - SHA3 of the uncles data in the block.
*  <b> miner: DATA, 20 Bytes </b> - the address of the beneficiary to whom the mining rewards were given.
*  <b> stateRoot: DATA, 32 Bytes </b> - the root of the final state trie of the block.
*  <b> transactionsRoot: DATA, 32 Bytes </b> - the root of the transaction trie of the block.
*  <b> receiptsRoot: DATA, 32 Bytes </b> - the root of the receipts trie of the block.
*  <b> logsBloom: DATA, 256 Bytes </b>- the bloom filter for the logs of the block.
*  <b> difficulty: QUANTITY </b> - integer of the difficulty for this block.
*  <b> totalDifficulty: QUANTITY </b> - integer of the total difficulty of the chain until this block.
*  <b> number: QUANTITY </b> - the block number.
*  <b> gasLimit: QUANTITY </b> - the maximum gas allowed in this block.
*  <b> gasUsed: QUANTITY </b> - the total used gas by all transactions in this block.
*  <b> timestamp: QUANTITY </b> - the unix timestamp for when the block was collated.
*  <b> extraData: DATA </b> - the “extra data” field of this block.
*  <b> mixHash: DATA, 32 Bytes </b> - "0xadce6e5230abe012342a44e4e9b6d05997d6f015387ae0e59be924afc7ec70c1" represents a hash of "PolyBFT Mix" to identify whether the block is from PolyBFT consensus engine.
*  <b> nonce: DATA, 8 Bytes </b> - hash of the generated proof-of-work.
*  <b> hash: DATA, 32 Bytes </b> - hash of the block.
*  <b> baseFeePerGas: QUANTITY </b> - base fee per gas.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getHeaderByNumber","params":["latest"],"id":1}'
````

## eth_getLogs

Returns an array of all logs matching a given filter object.

### Parameters
<b> Object </b>  - The filter options:

*  <b> fromBlock: QUANTITY|TAG </b> - (optional, default: "latest") Integer block number, or "latest" for the last mined block
*  <b> toBlock: QUANTITY|TAG </b> - (optional, default: "latest") Integer block number, or "latest" for the last mined block
*  <b> address: DATA|Array, 20 Bytes </b> - (optional) Contract address or a list of addresses from which logs should originate.
*  <b> topics: Array of DATA </b> - (optional) Array of 32 Bytes DATA topics. Topics are order-dependent. Each topic can also be an array of DATA with “or” options.
*  <b> blockhash: DATA, 32 Bytes </b> - (optional) With the addition of EIP-234, blockHash will be a new filter option which restricts the logs returned to the single block with the 32-byte hash blockHash. Using blockHash is equivalent to fromBlock = toBlock = the block number with hash blockHash. If blockHash is present in the filter criteria, then neither fromBlock nor toBlock is allowed.

### Returns

*  <b> Array of log objects </b> - log objects matching the filter criteria.

<b> Object </b>  - A log object:
*  <b> address: DATA, 20 Bytes </b> - address the log originated from.
*  <b> topics: Array of DATA, 32 bytes each </b> - event signature hash and 0 to 4 indexed log arguments.
*  <b> data: DATA, 20 bytes each </b> - non-indexed arguments of the log.
*  <b> blockNumber: QUANTITY </b> - number of the block that includes the log. null when log is pending.
*  <b> transactionHash: DATA, 32 Bytes </b> - hash of the starting transaction for the log. null when log is pending.
*  <b> transactionIndex: QUANTITY </b> - index position of the starting transaction for the log. null when log is pending.
*  <b> blockHash: DATA, 32 Bytes </b> - hash of the block that includes the log. null when log is pending.
*  <b> logIndex: QUANTITY </b> - log index position in the block. null when log is pending.
*  <b> removed: TAG </b> - true if log removed due to chain reorganization, otherwise false.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getLogs","params":[{"topics": ["0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b"]}],"id":1}'
````

## eth_getStorageAt

Returns the value from a storage position at a given address.

### Parameters

*  <b>  DATA, 20 Bytes </b> - address of the storage.
*  <b>  QUANTITY </b> - integer of the position in the storage.
*  <b>  QUANTITY|TAG </b> - integer block number, or the string "latest"

### Returns

*  <b>  DATA </b> - the value at this storage position.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["0x295a70b2de5e3953354a6a8344e616ed314d7251", "0x0", "latest"],"id":1}'
````

## eth_getTransactionByBlockHashAndIndex

Returns transaction information for the specified block number and transaction index position.

### Parameters

*  <b> DATA, 32 Bytes </b> - hash of a block
*  <b> QUANTITY </b> - transaction index position

### Returns

<b> Object </b> - A transaction object, or null when no transaction was found:

*  <b>  nonce: QUANTITY </b> - the number of transactions made by the sender prior to this one.
*  <b>  gasPrice: QUANTITY </b> - gas price provided by the sender in Wei.
*  <b>  gas: QUANTITY </b> - gas provided by the sender.
*  <b>  to: DATA, 20 Bytes </b> - address of the receiver. null when its a contract creation transaction.
*  <b>  value: QUANTITY </b> - value transferred in Wei.
*  <b>  input: DATA </b> - the data send along with the transaction.
*  <b>  v: QUANTITY </b> - ECDSA recovery id
*  <b>  r: DATA, 32 Bytes </b> - ECDSA signature r
*  <b>  s: DATA, 32 Bytes </b> - ECDSA signature s
*  <b>  hash: DATA, 32 Bytes </b> - hash of the transaction.
*  <b>  from: DATA, 20 Bytes </b> - address of the sender.
*  <b>  blockHash: DATA, 32 Bytes </b> - hash of the block where this transaction was in.
*  <b>  blockNumber: QUANTITY </b> - block number where this transaction was in.
*  <b>  transactionIndex: QUANTITY </b> - integer of the transactions index position in the block.
*  <b>  chainId: QUANTITY </b> - ID of the chain.
*  <b>  type: QUANTITY </b> - transaction type.
*  <b>  accessList: ARRAY </b> -list of addresses and storage keys that the transaction accessed to.

### Example
````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionByBlockHashAndIndex","params":["0x2168d7d6ce7da245a708a43958535e1fb1b9e076956a785357131538ea37928c","0"],"id":1}'
````

## eth_getTransactionByBlockNumberAndIndex

Returns transaction information for the specified block number and transaction index position.

### Parameters

*  <b> QUANTITY|TAG </b> - integer block number, or the string "latest"
*  <b> QUANTITY </b> - transaction index position

### Returns

<b> Object </b> - A transaction object, or null when no transaction was found:

*  <b>  nonce: QUANTITY </b> - the number of transactions made by the sender prior to this one.
*  <b>  gasPrice: QUANTITY </b> - gas price provided by the sender in Wei.
*  <b>  gas: QUANTITY </b> - gas provided by the sender.
*  <b>  to: DATA, 20 Bytes </b> - address of the receiver. null when its a contract creation transaction.
*  <b>  value: QUANTITY </b> - value transferred in Wei.
*  <b>  input: DATA </b> - the data send along with the transaction.
*  <b>  v: QUANTITY </b> - ECDSA recovery id
*  <b>  r: DATA, 32 Bytes </b> - ECDSA signature r
*  <b>  s: DATA, 32 Bytes </b> - ECDSA signature s
*  <b>  hash: DATA, 32 Bytes </b> - hash of the transaction.
*  <b>  from: DATA, 20 Bytes </b> - address of the sender.
*  <b>  blockHash: DATA, 32 Bytes </b> - hash of the block where this transaction was in.
*  <b>  blockNumber: QUANTITY </b> - block number where this transaction was in.
*  <b>  transactionIndex: QUANTITY </b> - integer of the transactions index position in the block.
*  <b>  chainId: QUANTITY </b> - ID of the chain.
*  <b>  type: QUANTITY </b> - transaction type.
*  <b>  accessList: ARRAY </b> -list of addresses and storage keys that the transaction accessed to.

### Example
````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionByBlockNumberAndIndex","params":["latest","0"],"id":1}'
````

## eth_getTransactionByHash

Returns the information about a transaction requested by transaction hash.

### Parameters

*  <b> DATA, 32 Bytes </b> - hash of a transaction

### Returns

<b> Object </b> - A transaction object, or null when no transaction was found:

*  <b>  nonce: QUANTITY </b> - the number of transactions made by the sender prior to this one.
*  <b>  gasPrice: QUANTITY </b> - gas price provided by the sender in Wei.
*  <b>  gas: QUANTITY </b> - gas provided by the sender.
*  <b>  to: DATA, 20 Bytes </b> - address of the receiver. null when its a contract creation transaction.
*  <b>  value: QUANTITY </b> - value transferred in Wei.
*  <b>  input: DATA </b> - the data send along with the transaction.
*  <b>  v: QUANTITY </b> - ECDSA recovery id
*  <b>  r: DATA, 32 Bytes </b> - ECDSA signature r
*  <b>  s: DATA, 32 Bytes </b> - ECDSA signature s
*  <b>  hash: DATA, 32 Bytes </b> - hash of the transaction.
*  <b>  from: DATA, 20 Bytes </b> - address of the sender.
*  <b>  blockHash: DATA, 32 Bytes </b> - hash of the block where this transaction was in.
*  <b>  blockNumber: QUANTITY </b> - block number where this transaction was in.
*  <b>  transactionIndex: QUANTITY </b> - integer of the transactions index position in the block.
*  <b>  chainId: QUANTITY </b> - ID of the chain.
*  <b>  type: QUANTITY </b> - transaction type.
*  <b>  accessList: ARRAY </b> -list of addresses and storage keys that the transaction accessed to.

### Example
````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionByHash","params":["0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b"],"id":1}'
````

## eth_getTransactionCount

Returns the number of transactions sent from an address.

### Parameters

*  <b>  DATA, 20 Bytes </b> - address.
*  <b>  QUANTITY|TAG </b> - integer block number, or the string "latest"

### Returns


*  <b>  QUANTITY </b> - integer of the number of transactions send from this address.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionCount","params":["0x407d73d8a49eeb85d32cf465507dd71d507100c1","latest"],"id":1}'
````

## eth_getTransactionReceipt

Returns the receipt of a transaction by transaction hash.

Note That the receipt is not available for pending transactions.

### Parameters

*  <b> DATA, 32 Bytes </b> - hash of a transaction

### Returns

<b> Object </b>  - A transaction receipt object, or null when no receipt was found:

*  <b> cumulativeGasUsed : QUANTITY </b> - The total amount of gas used when this transaction was executed in the block.
*  <b> logsBloom: DATA, 256 Bytes </b> - Bloom filter for light clients to quickly retrieve related logs.
*  <b> logs: Array </b> - Array of log objects, which this transaction generated.
*  <b> transactionHash : DATA, 32 Bytes </b> - hash of the transaction.
*  <b> transactionIndex: QUANTITY </b> - integer of the transactions index position in the block.
*  <b> blockHash: DATA, 32 Bytes </b> - hash of the block where this transaction was in.
*  <b> blockNumber: QUANTITY </b> - block number where this transaction was in.
*  <b> gasUsed : QUANTITY </b> - The amount of gas used by this specific transaction alone.
*  <b> contractAddress : DATA, 20 Bytes </b> - The contract address created, if the transaction was a contract creation, otherwise null.
*  <b> from: DATA, 20 Bytes </b> - address of the sender.
*  <b> to: DATA, 20 Bytes </b> - address of the receiver. null when its a contract creation transaction.
*  <b>  type: QUANTITY </b> - transaction type.

It also returns either :

*  <b> root  : DATA 32 bytes </b> - post-transaction stateroot (pre Byzantium)
*  <b>status: QUANTITY </b> - either 1 (success) or 0 (failure)

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionReceipt","params":["0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238"],"id":1}'
````

## eth_maxPriorityFeePerGas

Returns an estimate of how much priority fee, in Wei, you can pay to get a transaction included in the current block.

### Parameters

* None

### Returns

*  <b>  QUANTITY </b> - hexadecimal value in Wei.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_maxPriorityFeePerGas","params":[],"id":1}'
````

## eth_newBlockFilter

Creates a filter in the node, to notify when a new block arrives.
To check if the state has changed, call eth_getFilterChanges.

### Parameters

None

### Returns

*  <b> QUANTITY </b> - a filter id.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_newBlockFilter","params":[],"id":1}'
````

## eth_newFilter

Creates a filter object, based on filter options.
To get all matching logs for specific filter, call eth_getFilterLogs.
To check if the state has changed, call eth_getFilterChanges.

### Parameters
<b> Object </b> - The filter options:

*  <b>  fromBlock: QUANTITY|TAG </b> - (optional, default: "latest") Integer block number, or "latest" for the last mined block
*  <b>  toBlock: QUANTITY|TAG </b> - (optional, default: "latest") Integer block number, or "latest" for the last mined block
*  <b>  address: DATA|Array, 20 Bytes </b> - (optional) Contract address or a list of addresses from which logs should originate.
*  <b>  topics: Array of DATA </b> - (optional) Array of 32 Bytes DATA topics. Topics are order-dependent. Each topic can also be an array of DATA with “or” options.

### Returns


*  <b> QUANTITY </b> - a filter id.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X --data '{"jsonrpc":"2.0","method":"eth_newFilter","params":[{"topics":["0x12341234"]}],"id":1}'
````

## eth_sendRawTransaction

Creates new message call transaction or a contract creation for signed transactions.

### Parameters

*  <b> DATA </b> - The signed transaction data.

### Returns


*  <b> DATA, 32 Bytes </b> - the transaction hash, or the zero hash if the transaction is not yet available.

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"],"id":1}'
````

## eth_syncing

Returns information about the sync status of the node

### Parameters

* None

### Returns

*<b> Boolean (FALSE) </b> - if the node isn't syncing (which means it has fully synced)

*<b> Object </b> - an object with sync status data if the node is syncing
  *  <b>startingBlock: QUANTITY </b> - The block at which the import started (will only be reset, after the sync reached his head)
  *  <b>currentBlock: QUANTITY </b> - The current block, same as eth_blockNumber
  *  <b>highestBlock: QUANTITY </b> - The estimated highest block

### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}'
````

## eth_uninstallFilter

Uninstalls a filter with a given id. Should always be called when a watch is no longer needed.
Additionally, filters timeout when they aren’t requested with eth_getFilterChanges for some time.

### Parameters

*  <b> QUANTITY </b> - The filter id.

### Returns


*  <b> Boolean </b> - true if the filter was successfully uninstalled, otherwise false.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"eth_uninstallFilter","params":["0xb"],"id":1}'
````

## eth_unsubscribe

Subscriptions are cancelled with a regular RPC call with eth_unsubscribe as a method and the subscription id as the first parameter. It returns a bool indicating if the subscription was cancelled successfully.

### Parameters

*  <b> SUBSCRIPTION ID </b>

### Returns

*  <b> UNSUBSCRIBED FLAG </b> - true if the subscription was cancelled successful.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_unsubscribe","params":["0x9cef478923ff08bf67fde6c64013158d"],"id":1}'
````
