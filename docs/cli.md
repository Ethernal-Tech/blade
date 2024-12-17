# CLI configuration

Configuration parameters are crucial for setting up and operating a Blade-powered chain. You can configure these parameters using the server commands. Before running these commands, it is essential to generate keys using the blade secrets command.

For information on the available CLI commands and their configuration flags and descriptions refer to the sections below.


## Backup
### Description
Create blockchain backup file by fetching blockchain data from the running node.<br>
Usage: `./blade backup [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--from string`     | Backup starting block number.                        | 0            | NO        |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
| `--out string`          | Export path for the backup file. |       | YES        |
| `--to string`           | Backup ending block number.          | latest block          | NO        |
</details>

### Example
```bash
./blade backup --out backup-file
```

## Genesis
### Description
Generates the genesis configuration file with the passed in parameters.<br>
Usage: `./blade genesis [flags]`<br>
Usage: `./blade genesis [command]`<br>
Available commands:<br>
* predeploy

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description | Default Value | Mandatory |
| :-------- | :---------- | :------------ | :-------- |
| `--base-fee-config string` | Initial base fee (in wei), base fee elasticity multiplier, and base fee change denominator (provided in the following format: `[<baseFee>][:<baseFeeEM>][:<baseFeeChangeDenom>]`). BaseFeeChangeDenom represents the value to bound the amount the base fee can change between blocks. Default BaseFee is 1 Gwei, BaseFeeEM is 2 and BaseFeeChangeDenom is 8. Note: BaseFee, BaseFeeEM, and BaseFeeChangeDenom should be greater than 0. | 1000000000:2:8 | NO |
| `--blade-admin string` | Address of owner/admin of NativeERC20 token and StakeManager. |  | YES |
| `--block-gas-limit uint` | The maximum amount of gas used by all transactions in a block. | 5242880 | NO |
| `--block-time duration` | The predefined period which determines block creation frequency. | 2s | NO |
| `--block-time-drift uint` | Configuration for block time drift value (in seconds). | 10s | NO |
| `--block-tracker-poll-interval duration` | Interval (number of seconds) at which block tracker polls for latest block at rootchain. | 1s | NO |
| `--bootnode strings` | MultiAddr URL for p2p discovery bootstrap. This flag can be used multiple times. |  | NO |
| `--bridge-allow-list-admin strings` | List of addresses to use as admin accounts in the bridge allow list. |  | NO |
| `--bridge-allow-list-enabled strings` | List of addresses to enable by default in the bridge allow list. |  | NO |
| `--bridge-block-list-admin strings` | List of addresses to use as admin accounts in the bridge block list. |  | NO |
| `--bridge-block-list-enabled strings` | List of addresses to enable by default in the bridge block list. |  | NO |
| `--burn-contract string` | The burn contract block and address (format: `<block>:<address>[:<burn destination>]`) |  | NO |
| `--chain-id uint` | The ID of the chain. | 100 | NO |
| `--checkpoint-interval uint` | Checkpoint submission interval in blocks. | 900 | NO |
| `--consensus string` | The consensus protocol to be used. | polybft | NO |
| `--contract-deployer-allow-list-admin strings` | List of addresses to use as admin accounts in the contract deployer allow list. |  | NO |
| `--contract-deployer-allow-list-enabled strings` | List of addresses to enable by default in the contract deployer allow list. |  | NO |
| `--contract-deployer-block-list-admin strings` | List of addresses to use as admin accounts in the contract deployer block list. |  | NO |
| `--contract-deployer-block-list-enabled strings` | List of addresses to enable by default in the contract deployer block list. |  | NO |
| `--dir string` | File path for the Blade genesis data. | ./genesis.json | NO |
| `--epoch-reward uint` | Reward size for block sealing. | 1 | NO |
| `--epoch-size uint` | Epoch size for the chain. | 10 | NO |
| `--max-validator-count uint` | The maximum number of validators in the validator set for PoS. | 9007199254740990 | NO |
| `--min-validator-count uint` | The minimum number of validators in the validator set for PoS. | 4 | NO | 
| `--name string` | Chain name. | blade | NO |
| `--native-token-config string` | native token configuration, provided in the following format: `<name:symbol:decimals count:is minted on local chain>` |  | NO |
| `--premine strings` | Premined accounts and balances (format: `<address>[:<balance>]`). | :1000000000000000000000000 | NO |
| `--proposal-quorum uint` | Percentage of total validator stake needed for a governance proposal to be accepted (from 0 to 100%). | 67 | NO |
| `--proxy-contracts-admin strings` | Admin for proxy contracts. | | YES |
| `--reward-token-code string` | Hex encoded reward token byte code. |  | NO |
| `--reward-wallet string` | Configuration of reward wallet in format `<address:amount>` |  | NO |
| `--sprint-size uint` | Number of blocks included into a sprint. | 5 | NO |
| `--stake strings` | Staked accounts and balances (format: `<address>[:<stake>]`). | :1000000000000000000000 | NO |
| `--stake-token string` | Stake token address. | 0x0000000000000000000000000000000000001010 | NO |
| `--transactions-allow-list-admin strings` | List of addresses to use as admin accounts in the transactions allow list. |  | NO |
| `--transactions-allow-list-enabled strings` | List of addresses to enable by default in the transactions allow list. |  | NO |
| `--transactions-block-list-admin strings` | List of addresses to use as admin accounts in the transactions block list. |  | NO |
| `--transactions-block-list-enabled strings` | List of addresses to enable by default in the transactions block list. |  | NO |
| `--trieroot string` | Trie root from the corresponding triedb. |  | NO |
| `--validators strings` | Initial validator addresses for the chain. |  | YES |
| `--validators-path string` | Root path containing polybft validators' secrets. | ./ | NO |
| `--validators-prefix string` | Folder prefix names for validators secrets. | test-chain- | NO |
| `--vote-delay string` | Number of blocks after proposal is submitted before voting starts. | 10 | NO |
| `--vote-period string` | Number of blocks that the voting period for a proposal lasts. | 10000 | NO |
| `--vote-proposal-threshold string` | Number of vote tokens (in wei) required in order for a voter to submit a proposal. | 1000 | NO |
| `--withdrawal-wait-period uint` | Number of epochs after which withdrawal can be done from child chain. | 1 | NO |


> **Info**<br>
> Mutually Exclusive Parameters
>- `--validators`: Validators defined by the user (format: `<P2P multi address>:<public ECDSA address>:<public BLS key>`). If this flag is set, the entire multi address must be specified. If not set, validators configuration will be read from `--validators-path`.
>- `--validators-path`: Root path containing polybft validators' secrets. If `--validators` flag is not specified, validators' configuration will be read from this path.
>- `--validators-prefix`: Folder prefix names for polybft validators' secrets. If `--validators` flag is set, this prefix will be used for folder names.

</details>

### Example
```bash
./blade genesis --reward-wallet 0xDEADBEEF --premine 0x0000000000000000000000000000000000000000 --proxy-contracts-admin 0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed --blade-admin 0x61324166B0202DB1E7502924326262274Fa4358F --validators /ip4/127.0.0.1/tcp/1478/p2p/16Uiu2HAmMYyzK7c649Tnn6XdqFLP7fpPB2QWdck1Ee9vj5a7Nhg8:0x61324166B0202DB1E7502924326262274Fa4358F:06d8d9e6af67c28e85ac400b72c2e635e83234f8a380865e050a206554049a222c4792120d84977a6ca669df56ff3a1cf1cfeccddb650e7aacff4ed6c1d4e37b055858209f80117b3c0a6e7a28e456d4caf2270f430f9df2ba37221f23e9bbd313c9ef488e1849cc5c40d18284d019dde5ed86770309b9c24b70ceff6167a6ca
```

## genesis predeploy
### Description
Specifies the contract to be predeployed on chain start.<br>
Usage: `./blade genesis predeploy [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--artifacts-name string`    | Built-in contract artifact name.  |             | YES       |
| `--artifacts-path string`    | Path to the contract artifacts JSON.  |             | NO       |
| `--chain string`    | Genesis file to update.  | ./genesis.json            | NO  |
| `--constructor-args strings`    | Constructor arguments if any.  |             | NO       |
| `--deployer-address string`    | Contract deployer account address.   |  0           | YES       |
| `--predeploye-address string`    | The address to predeploy to. Must be >= 0x0000000000000000000000000000000000001100   |  0x0000000000000000000000000000000000001100   | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--artifacts-name` and `--artifacts-path`: These flags are mutually >exclusive. Use either `--artifacts-name` for built-in contracts or `--artifacts-path` for externaly defined contracts.
</details>

### Example
```bash
./blade genesis predeploy --artifacts-name RootERC20 --deployer-address 0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed
```

## Mint-erc20
### Description
Mints ERC20 tokens to specified addresses.<br>
Usage: `./blade mint-erc20 [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--addresses strings`     | Receivers addresses.                        |             | YES        |
| `--amounts strings`   | ERC20 token amounts.   |             | YES       |
| `--erc20-token string`          | ERC20 token address. |       | YES        |
| `--jsonrpc string`           | JSON RPC interface.          | 0.0.0.0:8545          | NO        |
| `--private-key string`           | Minter user private key.          |           | YES        |
| `--tx-timeout duration`           | Timeout for transaction processing.          | 50s          | NO        |
</details>

### Example
```bash
./blade mint-erc20 --addresses 0x85da99c8a7c2c95964c8efd687e95e632fc533d6 0x26F3f1f3F1d75c6d5d5146d1e44cec8831d0283A --amounts 1 2 --erc20-token 0x37e2e1f3F1d75c6d5d6336d1e44cec8831d0272a --private-key hex_encoded_private_key
```

## Monitor
### Description
Starts logging block add / remove events on the blockchain.<br>
Usage: `./blade monitor [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
</details>

### Example
```bash
./blade monitor
```

## Peers
### Description
Top level command for interacting with the network peers. Only accepts subcommands.<br>
Usage: `./blade peers [command]`<br>
Available commands:<br>
* add
* list
* status

## peers add
### Description
Adds new peers to the peer list, using the peer's libp2p address.<br>
Usage: `./blade peers add [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--addr strings`   | The libp2p peers addresses.  |  | YES  |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632 | NO |
</details>

### Example
```bash
./blade peers add --addr /ip4/192.168.200.201/tcp/1478/p2p/16Uiu2HAmGEMQmFqe2U4ag35BWiXniZ6orJVgaxdtSyFwXhFqT4Ko
```

## peers list
### Description
Returns the list of connected peers, including the local node.<br>
Usage: `./blade peers list [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
</details>

### Example
```bash
./blade peers list
```

## peers status
### Description
Returns status of the specified peer, using the libp2p peer node ID.<br>
Usage: `./blade peers status [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
| `--peer-id string`   | Libp2p node ID of a specific peer within p2p network.  |             | YES       |
</details>

### Example
```bash
./blade peers status --peer-id 16Uiu2HAmGEMQmFqe2U4ag35BWiXniZ6orJVgaxdtSyFwXhFqT4Ko
```

## Regenesis
### Description
Copies trie db for specific block to a separate folder.<br>
Usage: `./blade regenesis [flags]`<br>
Usage: `./blade regenesis [command]`<br>
Available commands:<br>
* getroot
* history

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description | Default Value | Mandatory |
| :-------- | :---------- | :------------ | :-------- |
| `--source-path string` | Directory containing trie data which will be copied. |  | YES |
| `--stateRoot string` | Hash of state root which will be copied. |  | YES |
| `--target-path string` | Directory where to copy trie data. |  | YES |
</details>

### Example
```bash
./blade regenesis --source-path <dir containing old chain trie> --stateRoot <state root which will be copied into target trie> --target-path <directory containing new trie>
```

## regenesis getroot
### Description
Returns blockchain state root.<br>
Usage: `./blade regenesis getroot [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--block int`    | Block number of trie snapshot.  |    head         | NO       |
| `--rpc string`   | Blockchain JSON RPC IP address.  |             | YES      |
</details>

### Example
```bash
./blade regenesis getroot --rpc http://localhost:10002
```

## regenesis history
### Description
Run history test (compare chain and trie db state roots).<br>
Usage: `./blade regenesis history [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--chaindb string`    | Path to chain DB.  |    | YES     |
| `--from uint`   | Lower bound of regenesis test.  |  0  | NO      |
| `--to uint`   | Upper bound of regenesis test.  |  head  | NO      |
| `--triedb string`    | Path to trie DB.  |    | YES     |
</details>

### Example
```bash
./blade regenesis history --chaindb <chaindb path> --triedb <triedb path>
```

## Secrets
### Description
Top level SecretsManager command for interacting with secrets functionality. Only accepts subcommands.<br>
Usage: `./blade secrets [command]`<br>
Available commands:<br>
* generate
* init
* output

## secrets generate
### Description
Initializes the secrets manager configuration in the provided directory.<br>
Usage: `./blade secrets generate [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--dir string` | File path for the secrets manager configuration file.  | ./secretsManagerConfig.json | NO |
| `--extra string` | Specifies the extra fields map in string format 'key1=val1,key2=val2'. |             | NO |
| `--grpc-address string` | The GRPC interface.   | 127.0.0.1:9632 | NO |
| `--name string` | Name of the node for on-service record keeping. |  | YES |
| `--namespace string` | Namespace for the service. | admin | NO |
| `--server-url string` | Server URL for the service. |  | YES |
| `--token string` | Access token for the hashicorp-vault service. |  | YES (only for hashicorp-vault) |
| `--type string` | Type of the secrets manager. Available types: hashicorp-vault, aws-ssm, gcp-ssm and alibaba-ssm. | hashicorp-vault | NO |
</details>

### Example
```bash
./blade secrets generate --name blade-test --type alibaba-ssm --server-url oos.eu-central-1.aliyuncs.com --extra 'region=eu-central-1,ssm-parameter-path=/devnet'
```

## secrets init
### Description
Initializes private keys for Blade (Validator + Networking) to the specified Secrets Manager.<br>
Usage: `./blade secrets init [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--account`    | The flag indicating whether a new account is created.  | TRUE            | NO       |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
| `--insecure`    | The flag indicating whether the secrets stored locally are encrypted.  | FALSE            | NO       |
| `--json-tls-cert`    | The flag indicating whether a new self signed TLS certificate is created for JSON RPC.  | TRUE            | NO       |
| `--network`    | The flag indicating whether a new network key is created.  | TRUE            | NO       |
| `--num int`    | Indicating how many secrets should be created, only for the local FS.  | 1            | NO       |
| `--output`    | The flag indicating whether to output existing secrets.  | FALSE            | NO       |
| `--private`           | The flag indicating whether the private key is printed.            | FALSE         | NO        |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.
>
>- `--num` and `--config`: These flags are mutually exclusive. Set `--num` to define number of secrets to be created (only for local FS) or use `--config` to provide the SecretsManager config file path.

</details>

### Example
```bash
./blade secrets init --data-dir data --insecure
```

## secrets output
### Description
Outputs validator key address and public network key from the provided Secrets Manager.<br>
Usage: `./blade secrets output [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--bls`    | Output only the BLS public key from the provided secrets manager.  | FALSE            | NO       |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
| `--node-id`    | Output only the node id from the provided secrets manager.  | FALSE            | NO       |
| `--validator`    | Output only the validator key address from the provided secrets manager.  | FALSE            | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.

</details>

### Example
```bash
./blade secrets output --data-dir data
```

## Server
### Description
Default command starting the Blade client, by bootstrapping all modules together.<br>
Usage: `./blade server [flags]`<br>
Usage: `./blade server [command]`<br>
Available commands:<br>
* export

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description | Default Value | Mandatory |
| :-------- | :---------- | :------------ | :-------- |
| `--access-control-allow-origins strings` | CORS header indicating whether any JSON RPC response can be shared with the specified origin. | [*] | NO |
| `--block-gas-target string` | The target block gas limit for the chain. If omitted, the value of the parent block is used which will be the value set by the `--block-gas-limit` flag of the genesis command. If this flag is set, the block fill take block gas limit of the parent block and increment it by small delta (parentGasLimit /1024). If the block gas target is reached that the value of it will be set as a gas limit for the current block. | 0x0 | NO |
| `--chain string` | Genesis file used for starting the chain. The genesis file is generated by running the genesis CLI command. | ./genesis.json | NO |
| `--concurrent-requests-debug uint` | Maximal number of concurrent requests for debug endpoints. | 32 | NO | 
| `--config string` | The path to the CLI config. Supported extensions are: .json, .hcl, .yaml and .yml. If this flag is set, other flags will be overridden. If some value that will be overridden is not specified in a config file, default value for that parameter is used. |  | NO |
| `--data-dir string` | The data directory used for storing Blade client data. |  | YES |
| `--dns string` | The host DNS address which can be used by a remote peer for connection. |  | NO |
| `--gossip-msg-size int` | Maximum size of gossip message in bytes. | 1048576 | NO |
| `--grpc-address string` | The address of the GRPC interface. | 127.0.0.1:9632 | NO |
| `--jsonrpc string` | The address of the JSON RPC interface. | 0.0.0.0:8545 | NO |
| `--json-rpc-batch-request-limit uint` | Max length to be considered when handling json rpc batch requests, value of 0 disables it. | 20 | NO |
| `--json-rpc-block-range-limit uint` | Max block range to be considered when executing json-rpc requests that consider fromBlock/toBlock values (e.g. eth_getLogs), value of 0 disables it. | 1000 | NO |
| `--libp2p string` | The address and port for the libp2p service. | 127.0.0.1:1478 | NO |
| `--log-level string` | The log level for the console output. | INFO | NO |
| `--log-to string` | Write all logs to the file at specified location instead of writing them to console. |  | NO |
| `--max-enqueued uint` | Maximum number of enqueued transactions in the pool per account. | 128 | NO |
| `--max-inbound-peers int` | The client's max number of inbound peers allowed. | 32 | NO |
| `--max-outbound-peers int` | The client's max number of outbound peers allowed. | 8 | NO |
| `--max-peers int` | The client's max number of peers allowed. | 40 | NO |
| `--max-slots uint` | Maximum slots in the transaction pool. When the maximum capacity is reached, transaction is not stored in the pool. One transaction occupies txSize/32kB number of slots. If e.g. --max-slots is 5, and there are tx1 which has 2kB and tx2 which has 33kB, that means that 3 slots are occupied and there are 2 free slots left. This parameter refers to the enqueued and promoted transactions in the pool. | 4096 | NO |
| `--metrics-interval duration` | The interval (in seconds) at which special metrics are generated. A value of zero means the metrics are disabled. | 8s | NO |
| `--nat string` | The external IP address without port, as can be seen by peers. The string specidied can be in IPv4 dotted decimal ("192.0.2.1"), IPv6 ("2001:db8::68"), or IPv4-mapped IPv6 ("::ffff:192.0.2.1") form. |  | NO |
| `--no-discover` | Prevent the client from discovering other peers. | FALSE | NO |
| `--num-block-confirmations uint` | Minimal number of child blocks required for the parent block to be considered final. This parameter is used by the event Tracker when reading logs from the parent chain. | 64 | NO |
| `--num-blocks-reconcile uint` | Defines how many blocks we will sync up from the latest block on tracked chain. If a node that has a tracker, was offline for days, months, a year, it is going to miss a lot of blocks potentially. In the meantime, we expect the rest of nodes to have collected the desired events and did their logic with them, continuing consensus and relayer stuff. In order to not waste too much unnecessary time in syncing all those blocks, with NumOfBlocksToReconcile, we tell the tracker to sync only latestBlock.Number - NumOfBlocksToReconcile number of blocks. | 64 | NO |
| `--price-limit uint` | The minimum gas price limit to enforce for acceptance into the pool. | 0 | NO |
| `--prometheus string` | The address and port for the prometheus instrumentation service (address:port). If only port is defined (:port) it will bind to 0.0.0.0:port. |  | NO |
| `--relayer` | Start the state sync relayer service. | FALSE | NO |
| `--restore string` | The path to the archive blockchain data to restore on initialization. |  | NO |
| `--seal` | The flag indicating that the client should seal blocks. | TRUE | NO |
| `--secrets-config string` | The path to the SecretsManager config file. If omitted, the local FS secrets manager is used. |  | NO |
| `--sync-batch-size uint` | Defines a batch size of blocks that will be gotten from tracked chain, when tracker is out of sync and needs to sync a number of blocks. (e.g., SyncBatchSize = 10, trackers last processed block is 10, latest block on tracked chain is 100, it will get blocks 11-20, get logs from confirmed blocks of given batch, remove processed confirm logs from memory, and continue to the next batch) .| 128 | NO |
| `--tls-cert-file string` | Path to TLS cert file, if no file is provided then cert file is loaded from secrets manager. |  | NO |
| `--tls-key-file string` | Path to TLS key file, if no file is provided then key file is loaded from secrets manager. |  | NO |
| `--tx-gossip-batch-size uint` | Maximum number of transactions in a single gossip message. | 1 | NO |
| `--use-tls` | Start JSON RPC endpoint with TLS enabled. | FALSE | NO |
| `--websocket-read-limit uint` | Maximum size in bytes for a message read from the peer by websocket. | 8192 | NO |
</details>

### Example
```bash
./blade server --data-dir data
```

## server export
### Description
Export default-config.yaml file with default parameters that can be used to run the server.<br>
Usage: `./blade server export [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--type string`    | File type of exported config file (yaml or json).  |  yaml  | NO |
</details>

### Example
```bash
./blade server export
```

## Status
### Description
Returns status of the Blade client.<br>
Usage: `./blade status [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
</details>

### Example
```bash
./blade status
```

## TxPool
### Description
Top level command for interacting with the transaction pool. Only accepts subcommands.<br>
Usage: `./blade txpool [command]`<br>
Available commands:<br>
* status
* subscribe

## txpool status
### Description
Returns the number of transactions in the transaction pool.<br>
Usage: `./blade txpool status [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
</details>

### Example
```bash
./blade txpool status
```

## txpool subscribe
### Description
Logs specific TxPool events.<br>
Usage: `./blade txpool subscribe [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--added`   | Subscribe to transaction added events.   | FALSE            | NO       |
| `--demoted`   | Subscribe to transaction demoted events.   | FALSE            | NO       |
| `--dropped`   | Subscribe to transaction dropped events.   | FALSE            | NO       |
| `--enqueued`   | Subscribe to transaction enqueued events.   | FALSE            | NO       |
| `--grpc-address string`   | The GRPC interface.   | 127.0.0.1:9632            | NO       |
| `--promoted`   | Subscribe to transaction promoted events.   | FALSE            | NO       |
| `--pruned-enqueued`   | Subscribe to transaction pruned-enqueued events.   | FALSE            | NO       |
| `--pruned-promoted`   | Subscribe to transaction pruned-promoted events.   | FALSE            | NO       |
</details>

### Example
```bash
./blade txpool subscribe --added --demoted  --dropped --enqueued --promoted --pruned-enqueued --pruned-promoted
```

## Validator
### Description
Validator command for interacting with validators. Only accepts subcommands.<br>
Usage: `./blade validator [command]`<br>
Available commands:<br>
* info
* register-validator
* stake
* unstake
* whitelist-validators
* withdraw
* withdraw-rewards

## validator info
### Description
Gets validator info.<br>
Usage: `./blade validator info [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--jsonrpc string`    | The JSON RPC interface.  | 0.0.0.0:8545            | NO       |
| `--tx-timeout duration`    | Timeout for transaction processing.  | 50s            | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.
</details>

### Example
```bash
./blade validator info --data-dir data
```

## validator register-validator
### Description
Registers a whitelisted validator to supernet manager on rootchain.<br>
Usage: `./blade validator register-validator [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--amount string`   | Amount to stake to validator.   | 0            | NO       |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--jsonrpc string`    | The JSON RPC interface.  | 0.0.0.0:8545            | NO       |
| `--stake-token string`    | Stake token address.  | 0x0000000000000000000000000000000000001010            | NO       |
| `--tx-timeout duration`    | Timeout for transaction processing.  | 50s            | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.
</details>

### Example
```bash
./blade validator register-validator --data-dir data
```

## validator stake
### Description
Stakes the amount sent to validator.<br>
Usage: `./blade validator stake [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--amount string`   | Amount to stake to validator   |            | YES       |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--jsonrpc string`    | The JSON RPC interface.  | 0.0.0.0:8545            | NO       |
| `--stake-token string`    | Stake token address.  | 0x0000000000000000000000000000000000001010            | NO       |
| `--tx-timeout duration`    | Timeout for transaction processing.  | 2m30s            | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.
</details>

### Example
```bash
./blade validator stake --amount 10 --data-dir data
```

## validator unstake
### Description
Unstakes the amount sent for validator or undelegates amount from validator.<br>
Usage: `./blade validator unstake [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--amount string`   | Amount to unstake from validator   |            | YES       |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--jsonrpc string`    | The JSON RPC interface.  | 0.0.0.0:8545            | NO       |
| `--tx-timeout duration`    | Timeout for transaction processing.  | 2m30s            | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.
</details>

### Example
```bash
./blade validator unstake --amount 10 --data-dir data
```

## validator whitelist-validators
### Description
Whitelist new validators.<br>
Usage: `./blade validator whitelist-validators [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--addresses strings`   | Account addresses of a possible validators.   |            | YES       |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--jsonrpc string`    | The JSON RPC interface.  | 0.0.0.0:8545            | NO       |
| `--private-key string`    | Hex-encoded private key of the account executing the command.  |            | YES       |
| `--tx-timeout duration`    | Timeout for transaction processing.  | 2m30s            | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.
</details>

### Example
```bash
./blade validator whitelist-validators --addresses 0x85da99c8a7c2c95964c8efd687e95e632fc533d6 --data-dir data --private-key <private key>
```

## validator withdraw
### Description
Withdraws validator's withdrawable stake.<br>
Usage: `./blade validator withdraw [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--jsonrpc string`    | The JSON RPC interface.  | 0.0.0.0:8545            | NO       |
| `--tx-timeout duration`    | Timeout for transaction processing.  | 2m30s            | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.
</details>

### Example
```bash
./blade validator withdraw --data-dir data
```

## validator withdraw-rewards
### Description
Withdraws validator pending rewards on child chain.<br>
Usage: `./blade validator withdraw-rewards [flags]`

<details>
<summary><b>Flags ↓</b></summary>

| Parameter | Description                                                      | Default Value | Mandatory |
| :-------- | :--------------------------------------------------------------- | :------------ | :-------- |
| `--config string`    | Path to the SecretsManager config file, if omitted, the local FS secrets manager is used.  |             | NO       |
| `--data-dir string`    | Directory for the Blade data if the local FS is used.  |             | YES       |
| `--jsonrpc string`    | The JSON RPC interface.  | 0.0.0.0:8545            | NO       |
| `--tx-timeout duration`    | Timeout for transaction processing.  | 2m30s            | NO       |

> **Info**<br>
> Mutually Exclusive Parameters
>- `--config` and `--data-dir`: These flags are mutually >exclusive. Use either `--config` to specify the path to the SecretsManager config file or `--data-dir` to set the directory for the Blade data if the local FS is used.
</details>

### Example
```bash
./blade validator withdraw-rewards --data-dir data
```

## Version
### Description
Returns current Blade version.<br>
Usage: `./blade version`

### Example
```bash
./blade version
```
