# debug

## debug\_accountRange

The AccountRange method provides a way to enumerate accounts in a block while supporting pagination and various filters. It is designed to optimize querying by allowing the caller to skip code, storage, or incomplete accounts, thus controlling the scope and size of the results.

#### Parameters

* filter: QUANTITY|TAG - integer block number, or the string "latest".
* start: Array - start node key ([]byte) from the tree.
* maxResults: QUANTITY - the maximum number of accounts to return. Adjusted to fall within a predefined maximum range.
* noCode: Boolean - if true, skips retrieving the code for accounts.
* noStorage: Boolean - if true, skips retrieving the storage for accounts.
* incompletes: Boolean - if true, includes accounts without code or storage in the result.

#### Returns

* Object - returns an IteratorDump object, with the following fields:
  * Object - dump object, with the following fields:

    + Root: Array - root in the tree ([]byte)
    + Accounts: Array - an account map whose key is an address and the value is a DumpAccount object, which has the following fields:
      - Balance: String - Balance
      - Nonce: QUANTITY - Nonce
      - Root: Array - Root ([]byte)
      - CodeHash: Array - CodeHash ([]byte)
      - Code: Array - Code ([]byte)
      - Storage: Array - Storage
      - Address: Array - Address ([]byte)
      - Key: Array - Key ([]byte)

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_accountRange","params":[10, "", 2, false, false, false],"id":1}'

```
<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result":
  {
    "root": "YbcKG3qGHpnvXUXcDcTuxCj3V0vOGpt4/wHfIsnFjG8=",
    "accounts":
    {
      "0x0000000000000000000000000000000000000101":
      {
        "balance": "0",
        "nonce": 0,
        "root": "GC+YUXAq0MX6qDZSQa7plE6/KgzpAnvfmnemOpLQjqs=",
        "codeHash": "ABE7+ZXWu17BLZtPKM2daMQgSGEV1Jw9yHevoD67PPw=",
        "storage":
        {
          "0x0000000000000000000000000000000000000000000000000000000000000036":"0000000000000000000000000000000000000000000000000000000000000002","0x8e0cc0f1f0504b4cb44a23b328568106915b169e79003737a7b094503cdbeeb0":"0000000000000000000000000000000000000000000000000000000000000001","0x8e0cc0f1f0504b4cb44a23b328568106915b169e79003737a7b094503cdbeeb1":"000000000000000000000000000000000000000000000000000000000000000a","0xbff1b53d0f70f16319f1906b82d0a4d5ea1bc8510376ffd434e80b642d0ea0a8":"000000000000000000000000000000000000000000000000000000000000000a"
        },
        "address": "0x0000000000000000000000000000000000000101",
        "key": "AAAAAAAAAAAAAAAAAAAAAAAAAQE="
      },
      "0x4208f7e4F8a238c0c602c9Cac344C4d17117b37e":
      {
        "balance": "998999999971502542406185",
        "nonce": 0,
        "root": "VugfFxvMVab/g0XmksD4bltI4BuZbK3AAWIvteNjtCE=",
        "codeHash": "xdJGAYb3IzySfn2y3McDwOUAtlPKgic7e/rYBF2FpHA=",
        "address": "0x4208f7e4F8a238c0c602c9Cac344C4d17117b37e",
        "key": "Qgj35PiiOMDGAsnKw0TE0XEXs34="
      }
    },
    "next": "//////////////////////////4="
  }
}
```

</details>



## debug\_blockProfile

The BlockProfile method enables goroutine blocking profile collection for a specified duration and writes the collected profile data to a file. This method helps in understanding goroutine blocking behavior, which is useful for detecting bottlenecks or inefficient blocking of goroutines.

#### Parameters

* file: String - the file path where the blocking profile data will be written. The method saves the collected profile data into this file.
* nsec: QUANTITY - the duration (in seconds) for which the goroutine blocking profiling will run. The profiling will collect data for the specified number of seconds and then stop.

#### Returns

* String - success: The method returns the absolute path of the file where the profile data has been written. This is returned as an interface{}, which in this case would be the string representing the file path. Failure: The method returns an error if something goes wrong while starting or stopping the blocking profile collection or if there is an issue with the file operations (e.g., invalid file path). If the operation completes successfully, it returns nil for the error.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_blockProfile","params":["block.txt", 3],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/block.txt"
}
```

</details>



## debug\_chaindbCompact

The ChaindbCompact method flattens the entire key-value database into a single level. It removes all unused slots and merges all keys to optimize the database by compacting it.

#### Parameters

None

#### Returns

* QUANTITY - the method returns a true or false value indicating whether the compaction was successful.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_chaindbCompact","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": true
}
```

</details>



## debug\_chaindbProperty

The ChaindbProperty method is designed to retrieve properties from the underlying LevelDB key-value database. It allows users to query specific LevelDB statistics or other properties by passing a property name. If no property is specified, it defaults to querying the leveldb.stats.

#### Parameters

* property: String -  the name of the property to retrieve from the LevelDB database. Specifies which property or statistic to query from the LevelDB databaseb. If property is an empty string, it defaults to "leveldb.stats".

#### Returns

* String - returns a particular internal stat of the database.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_chaindbProperty","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result":
  "Compactions\n Level |   Tables   |    Size(MB)   |    Time(sec)  |    Read(MB)   |   Write(MB)\n-------+------------+---------------+---------------+---------------+---------------\n   0   |          1 |       3.36916 |       0.16406 |       0.00000 |      13.38125\n   1   |          5 |       8.87396 |       0.17261 |      15.90825 |      14.77012\n-------+------------+---------------+---------------+---------------+---------------\n Total |          6 |      12.24312 |       0.33667 |      15.90825 |      28.15137\n"
}
```

</details>



## debug\_cpuProfile

CpuProfile turns on CPU profiling for nsec seconds and writes profile data to file.

#### Parameters

* file: String - this is the name or path of the file where the CPU profiling data will be written. The file path can be relative or absolute. If it's relative, the method resolves and returns the absolute path. Example: "cpu_profile.out".
* nsec: QUANTITY - this is the duration, in seconds, for which the CPU profiling should run. It specifies how long the CPU profiling session will capture performance data. Must be a positive integer; otherwise, the method should handle it as invalid input. Example: 30 (for 30 seconds).

#### Returns

* String - success: Returns an interface{} that contains the absolute path to the file where the CPU profiling data has been written. The path is always resolved to its absolute form, ensuring consistency and clarity for the caller. Example: "/home/user/cpu_profile.out". Failure: Returns an error if any part of the process fails, such as: An error starting the CPU profiling (StartCPUProfile). An error stopping the CPU profiling (StopCPUProfile). An error resolving the absolute path of the file (filepath.Abs). The error provides details about the specific failure encountered during execution.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_cpuProfile","params":["cpu.txt", 30],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/cpu.txt"
}
```

</details>



## debug\_dbGet

The DbGet method retrieves the raw value of a key stored in the database. This method allows access to data stored in the form of key-value pairs in the database.

#### Parameters

* key: String - the key for which we want to retrieve the value from the database.

#### Returns

* Array - []byte: the value stored at the given hash.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_dbGet","params":["69aa84ad9f171ea26d1e18c2ee89b91d1447d01fc1990d165a27acd715cb7625"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result":
    "+QHxoNElseyBgWP+dbt1MkHLagL0otADlusCTvmIC7YIiMazoPIZU8sGiupaOex1zQwxr1ggpUbernWGaTBno2RCGu2loE7diQmPGWCBLELOyBWkjPRfGY9wN1s9S9qeGf9nHQssoHxJXLo8LNAdVFOdzDpiIIv0B+f0hHDnS2qY1lhZZ5URoOc+gsjaY+wBLbJxNlDO6Kf91vhiupo9+WsQgRjGMRmGoNYb95K4oX0zAL9wNsCDigmGMqgQ0e8HuuYQ2h1/jsgWoO2Zgv4pwy71dFU2VHiwgL4rQBq9BTAEF0mMLctcAx/aoAObdBIVy64NQtksqy6QlfPrGbmohGlUBely8wWKNoRkoPW+PeZrNwhoswbd+BEYSMpXVCsHz+FbQ1s9wTOh6FHmoJg4iQ68Y4roQGepslWBOqLVqbq+Qm56Dj18mWFzdS/PoEX+/YFZRzsn//TNhiUryXmTG9Ruyxn+A/keLvhm12DcoLMcBYpNXd7kReP2DqmzmLDTtW+SUTlxfhyA2WcVtpFzoLYoZQdYurZs5zKLo6utNxih4SAUnFHtTg83P76v6/CSgKAuyFSKaLT2O4UHibXYhRjKGzS3mJp+rpd1bSECWlwAqKABk+PnA4WOj5EMZLPc8572uM8ExRkrUreHPpgURgNOBIA="
}
```

</details>



## debug\_dumpBlock

The DumpBlock method retrieves the entire blockchain state at a specific block. It uses a throttling mechanism to manage execution and ensures that only valid blocks are processed. The method fetches the block by number, creates options for dumping state information, and invokes the state dump process.

#### Parameters

* blockNumber: QUANTITY|TAG - integer of a block number, or the string "latest"

#### Returns

* Object - returns an Dump object, with the following fields:
   * Root: Array - root in the tree ([]byte)
   * Accounts: Array - an account map whose key is an address and the value is a DumpAccount object, which has the following fields:
      + Balance: String - Balance
      + Nonce: QUANTITY - Nonce
      + Root: Array - Root ([]byte)
      + CodeHash: Array - CodeHash ([]byte)
      + Code: Array - Code ([]byte)
      + Storage: Array - Storage
      + Address: Array - Address ([]byte)
      + Key: Array - Key ([]byte)
   
#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_dumpBlock","params":[10],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": 
  {
    "root": "WvqTbkYggPLxUqXv6Qjgbk0h59o46N02zmwwaDwS9DA=",
    "accounts":
    {
      "0x0000000000000000000000000000000000000101":
      {
        "balance": "0",
        "nonce": 0,
        "root": "GC+YUXAq0MX6qDZSQa7plE6/KgzpAnvfmnemOpLQjqs=",
        "codeHash": "ABE7+ZXWu17BLZtPKM2daMQgSGEV1Jw9yHevoD67PPw=",
        "storage":
        {
          "0x8e0cc0f1f0504b4cb44a23b328568106915b169e79003737a7b094503cdbeeb0":"0000000000000000000000000000000000000000000000000000000000000001","0x8e0cc0f1f0504b4cb44a23b328568106915b169e79003737a7b094503cdbeeb1":"000000000000000000000000000000000000000000000000000000000000000a","0xbff1b53d0f70f16319f1906b82d0a4d5ea1bc8510376ffd434e80b642d0ea0a8":"000000000000000000000000000000000000000000000000000000000000000a","0x0000000000000000000000000000000000000000000000000000000000000036":"0000000000000000000000000000000000000000000000000000000000000002"
        },
        "address": "0x0000000000000000000000000000000000000101",
        "key": "AAAAAAAAAAAAAAAAAAAAAAAAAQE="
      },
      "0xE622Abc372132a2e081F54b3efd7a2Fa088343da":
      {
        "balance": "998999999971502542406185",
        "nonce": 0,
        "root": "VugfFxvMVab/g0XmksD4bltI4BuZbK3AAWIvteNjtCE=",
        "codeHash": "xdJGAYb3IzySfn2y3McDwOUAtlPKgic7e/rYBF2FpHA=",
        "address": "0xE622Abc372132a2e081F54b3efd7a2Fa088343da",
        "key": "5iKrw3ITKi4IH1Sz79ei+giDQ9o="
      },
      "0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE":
      {
        "balance": "0",
        "nonce": 1,
        "root": "VugfFxvMVab/g0XmksD4bltI4BuZbK3AAWIvteNjtCE=",
        "codeHash": "xdJGAYb3IzySfn2y3McDwOUAtlPKgic7e/rYBF2FpHA=",
        "address": "0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE",
        "key": "//////////////////////////4="
      }
    }
  }
}
```

</details>



## debug\_freeOSMemory

FreeOSMemory forces a garbage collection.

#### Parameters

None

#### Returns

None

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_freeOSMemory","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": null
}
```

</details>



## debug\_gcStats

GcStats returns GC statistics.

#### Parameters

None

#### Returns

DATA - the function returns an object of type *debug.GCStats, which contains various statistics about the Go garbage collector. These statistics include details like the number of GC cycles, memory allocated and freed, and the duration of each cycle. This object is returned as an interface{}, which allows flexibility in the type handling. The debug.GCStats struct typically includes fields such as:
  * LastGC: QUANTITY - time of last collection.
  * NumGC: QUANTITY - the number of GC cycles that have been executed.
  * PauseTotal: QUANTITY - total time spent in GC pauses.
  * Pause: Array - pause history, most recent first.
  * PauseEnd: Array - pause end times history, most recent first.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_gcStats","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result":
  {
    "LastGC": "2025-01-19T18:08:46.605234142+01:00",
    "NumGC": 44,
    "PauseTotal": 3505059,
    "Pause": [
      91571,
      114686,
      73989,
      75862,
      100169,
      73157,
      87353,
      55073,
      116900,
      73628,
      63941,
      45625,
      75973,
      52148,
      104706,
      75341,
      55805,
      80281,
      108793,
      83656,
      73908,
      106027,
      69389,
      65303,
      50015,
      109375,
      88707,
      71955,
      75071,
      52079,
      70603,
      64351,
      75442,
      74430,
      126068,
      88869,
      122120,
      115829,
      72386,
      59433,
      83358,
      72438,
      49413,
      59833
    ],
    "PauseEnd": [
      "2025-01-19T18:08:46.605234142+01:00",
      "2025-01-19T18:06:46.284758751+01:00",
      "2025-01-19T18:04:46.147154771+01:00",
      "2025-01-19T18:02:45.605019086+01:00",
      "2025-01-19T18:00:44.74541066+01:00",
      "2025-01-19T17:58:44.675952764+01:00",
      "2025-01-19T17:57:39.584198133+01:00",
      "2025-01-19T17:55:39.544485367+01:00",
      "2025-01-19T17:53:39.518874244+01:00",
      "2025-01-19T17:51:39.504132479+01:00",
      "2025-01-19T17:49:39.48519388+01:00",
      "2025-01-19T17:47:39.268085334+01:00",
      "2025-01-19T17:45:39.183322612+01:00",
      "2025-01-19T17:43:39.061600456+01:00",
      "2025-01-19T17:41:38.941453074+01:00",
      "2025-01-19T17:39:38.917114673+01:00",
      "2025-01-19T17:37:38.604940973+01:00",
      "2025-01-19T17:35:38.484118725+01:00",
      "2025-01-19T17:33:38.327543088+01:00",
      "2025-01-19T17:31:38.00895862+01:00",
      "2025-01-19T17:29:37.604760753+01:00",
      "2025-01-19T17:27:37.528651913+01:00",
      "2025-01-19T17:25:37.054701511+01:00",
      "2025-01-19T17:24:09.681103984+01:00",
      "2025-01-19T17:22:09.671150074+01:00",
      "2025-01-19T17:20:09.665515504+01:00",
      "2025-01-19T17:18:09.659027321+01:00",
      "2025-01-19T17:16:09.651111752+01:00",
      "2025-01-19T17:14:09.64407029+01:00",
      "2025-01-19T17:12:09.636033208+01:00",
      "2025-01-19T17:10:09.62871868+01:00",
      "2025-01-19T17:08:09.622296609+01:00",
      "2025-01-19T17:06:09.615312556+01:00",
      "2025-01-19T17:04:09.605683551+01:00",
      "2025-01-19T17:02:09.600845601+01:00",
      "2025-01-19T17:00:09.583617883+01:00",
      "2025-01-19T16:58:09.544788587+01:00",
      "2025-01-19T16:56:09.541038256+01:00",
      "2025-01-19T16:56:09.514002368+01:00",
      "2025-01-19T16:56:09.314015388+01:00",
      "2025-01-19T16:56:09.295231118+01:00",
      "2025-01-19T16:56:09.279860625+01:00",
      "2025-01-19T16:56:09.266109565+01:00",
      "2025-01-19T16:56:09.260706435+01:00"
    ],
    "PauseQuantiles": null
  }
}
```

</details>



## debug\_getAccessibleState

The GetAccessibleState method is designed to find the first block within a specified range (from - to) where the node has accessible state data stored on disk. It considers the post-state of one block and the pre-state of the next block. The method ensures that the state data for the block is present and can be accessed from storage.

#### Parameters

* from: QUANTITY|TAG  - the starting block number. Integer of a block number, or the string "latest"
* to: QUANTITY|TAG - the ending block number. Integer of a block number, or the string "latest"

#### Returns

* QUANTITY - the method returns the block number (uint64) as the first block with accessible state.
   
#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_getAccessibleState","params":[3, 5],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": 3
}
```

</details>



## debug\_getModifiedAccountsByHash

This method aims to return all accounts that have changed between two specified blocks, identified by their block hashes. A "change" is defined as a difference in any of the following account properties: Nonce, Balance, Code Hash, Storage Hash.

#### Parameters

* startHash: DATA, 32 Bytes - the hash of the starting block.
* endHash: DATA - (*types.Hash) the optional hash of the ending block. If nil, the method will use the parent block of the startHash block as the endBlock.

#### Returns

* Array - returns a list of account addresses that have changed.
   
#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_getModifiedAccountsByHash","params":["0x47b0778abf13535354f63d0c345c7c08d483529551c2599cec536f1947c88763"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        "0x265241f900da16566d1da75b8fa627cf4b6f8160",
        "0x31e218d6f87c4558eeb1e1e2daa5d207e9c8861e",
        "0x7730c571fad047dcce53f83d5a55f01269c9883d",
        "0xa16cfb0e67a85d80fcd246d77623aa02482d2b10",
        "0x48c04ed5691981c42154c6167398f95e8f38a7ff",
        "0x0b9869c0b6601544f776284320423bc31fa0dc80",
        "0x303389f541ff2d620e42832f180a08e767b28e10",
        "0xca24e7d9e8a2ba3ada22383f5e2ad397b5677e25"
    ]
 }
```

</details>



## debug\_getModifiedAccountsByNumber

This method aims to return all accounts that have changed between two specified blocks, identified by their block numbers. A "change" is defined as a difference in any of the following account properties: Nonce, Balance, Code Hash, Storage Hash.

#### Parameters

* startNum: QUANTITY - number of the starting block.
* endNum: QUANTOTY - (*uint64) the optional number of the ending block. If nil, the method will use the parent block of the startNum block as the endBlock.

#### Returns

* Array - returns a list of account addresses that have changed.
   
#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_getModifiedAccountsByNumber","params":[10],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        "0x265241f900da16566d1da75b8fa627cf4b6f8160",
        "0x31e218d6f87c4558eeb1e1e2daa5d207e9c8861e",
        "0x7730c571fad047dcce53f83d5a55f01269c9883d",
        "0xa16cfb0e67a85d80fcd246d77623aa02482d2b10",
        "0x48c04ed5691981c42154c6167398f95e8f38a7ff",
        "0x0b9869c0b6601544f776284320423bc31fa0dc80",
        "0x303389f541ff2d620e42832f180a08e767b28e10",
        "0xca24e7d9e8a2ba3ada22383f5e2ad397b5677e25"
    ]
 }
```

</details>



## debug\_getRawBlock

Retrieves the RLP-encoded representation of a single block based on a given block number or hash.

#### Parameters

* filter: QUANTITY|TAG - integer block number, or the string "latest"

#### Returns

* Array - RLP-encoded byte slice ([]byte) representing the block.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_getRawBlock","params":[1],"id":1}' 
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "+QP5+QMWoNiFFS5PGJH8Js19fmJxhiOXWjzb/twjf1VRRVsjVofKoB3MTejex116q4W1Z7bM1BrTEkUblIp0E/ChQv1A1JNHlK2Y6hk8LfiCMDLehT/iaJQbLkjLoGmqhK2fFx6ibR4Ywu6JuR0UR9AfwZkNFlonrNcVy3YloAmVGXTbsDjZVcLcGO+5h/9f+Sg/PME6XRZiqrJc1r9coFh3Zwqj9elVldMmqvYHzHDmAsPOLvh+Po8sV8DWWzYiuQEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAIAAAAASAAAAAAAAAAAAAAAAAAAIAACAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAACAAAAAAAAAAAAAAAQAAAAAAAAAAAAEKhAL68ICDAaYJhGeOzHS5ARcAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPj1w8DAgPhDuEAAtKYA+8Sdi2QHXmqleYX2ZCZnpNbC3ErOcYirVnmhPxLnxqk/3wIjkIPtgsloQ/zqcge0gD64m2b5ajYcY/BbDvhDuEAgXcBH4RB7L8XkkGoKFyIN4lWw5kl5eVZ9ps8ZacAU8Qoft+XITCD4K35ure2PoOkwnxsukujXuA92T2AMdlknDfhlgAGgmvkN9q/mAbmd6bLJpyC1uyKoCS7z8aU54tvWkhG7w4qgmvkN9q/mAbmd6bLJpyC1uyKoCS7z8aU54tvWkhG7w4qgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACgrc5uUjCr4BI0KkTk6bbQWZfW8BU4euDlm+kkr8fscMGIAAAAAAAAAACED7i83/jdf/jagICDD0JAlAAAAAAAAAAAAAAAAAAAAAAAAAEBgLikjbOkwQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAKAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAgICU//////////////////////////7A"
}
```

</details>



## debug\_getRawHeader

Retrieves the RLP-encoded representation of a block header based on a given block number or hash.

#### Parameters

* filter: QUANTITY|TAG - integer block number, or the string "latest"

#### Returns

* Array - RLP-encoded byte slice ([]byte) representing the block header

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_getRawHeader","params":["latest"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "+QMWoNiFFS5PGJH8Js19fmJxhiOXWjzb/twjf1VRRVsjVofKoB3MTejex116q4W1Z7bM1BrTEkUblIp0E/ChQv1A1JNHlK2Y6hk8LfiCMDLehT/iaJQbLkjLoGmqhK2fFx6ibR4Ywu6JuR0UR9AfwZkNFlonrNcVy3YloAmVGXTbsDjZVcLcGO+5h/9f+Sg/PME6XRZiqrJc1r9coFh3Zwqj9elVldMmqvYHzHDmAsPOLvh+Po8sV8DWWzYiuQEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAIAAAAASAAAAAAAAAAAAAAAAAAAIAACAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAACAAAAAAAAAAAAAAAQAAAAAAAAAAAAEKhAL68ICDAaYJhGeOzHS5ARcAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPj1w8DAgPhDuEAAtKYA+8Sdi2QHXmqleYX2ZCZnpNbC3ErOcYirVnmhPxLnxqk/3wIjkIPtgsloQ/zqcge0gD64m2b5ajYcY/BbDvhDuEAgXcBH4RB7L8XkkGoKFyIN4lWw5kl5eVZ9ps8ZacAU8Qoft+XITCD4K35ure2PoOkwnxsukujXuA92T2AMdlknDfhlgAGgmvkN9q/mAbmd6bLJpyC1uyKoCS7z8aU54tvWkhG7w4qgmvkN9q/mAbmd6bLJpyC1uyKoCS7z8aU54tvWkhG7w4qgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACgrc5uUjCr4BI0KkTk6bbQWZfW8BU4euDlm+kkr8fscMGIAAAAAAAAAACED7i83w=="
}
```

</details>



## debug\_getRawReceipts

Retrieves the RLP-encoded binary representation of transaction receipts for a specified block.

#### Parameters

* filter: QUANTITY|TAG - integer block number, or the string "latest"

#### Returns

* Array - a slice of byte slices ([][]byte), where each inner slice represents an RLP-encoded receipt. Returns an empty slice if the block has no receipts.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_getRawReceipts","params":[10],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": 
  [
    "f/kByAGDAaYJuQEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAIAAAAASAAAAAAAAAAAAAAAAAAAIAACAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAACAAAAAAAAAAAAAAAQAAAAAAAAAAAPi++LyUAAAAAAAAAAAAAAAAAAAAAAAAAQH4hKAM6HEsTe5L1aaR8LwcOVlGcVkedzlfjr9qP7X2P76maqAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAaAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAaAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACqAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
  ]
}
```

</details>



## debug\_getRawTransaction

Retrieves the RLP-encoded representation of a transaction identified by its hash.

#### Parameters

* txHash: DATA, 32 Bytes - transaction hash.

#### Returns

* Array - RLP-encoded byte slice ([]byte) representing the transaction.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_getRawTransaction","params":["0x2fab8b2f856e40ff4690cc70ddf06d24b7f24f65dc84db17db3d21acc0832873"],"id":1}'
```
<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "f/jagICDD0JAlAAAAAAAAAAAAAAAAAAAAAAAAAEBgLikjbOkwQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAKAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAgICU//////////////////////////4="
}
```

</details>



## debug\_goTrace

The GoTrace method enables execution tracing in a Go application for a specified duration, saving the trace data to a given file. This is useful for diagnosing application performance and behavior by collecting detailed execution events.

#### Parameters

* file: String - the path of the file where the trace data will be written. The path can be relative or absolute. If relative, the method resolves it to an absolute path.
* nsec: QUANTITY - the duration for which tracing should be active, in seconds. During this period, execution trace events are captured and stored in the specified file.

#### Returns

* String - the absolute path of the file where trace data is written, allowing the user to locate the output.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_goTrace","params":["trace.txt", 30],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/trace.txt"
}
```

</details>



## debug\_intermediateRoots

The IntermediateRoots method executes a block and returns a list of intermediate state roots—representing the state of the blockchain after each transaction in the block.

#### Parameters

* DATA, 32 Bytes - block hash
* Object - The tracer options:

  + enableMemory: Boolean - (optional, default: false) The flag indicating enabling memory capture.
  + disableStack: Boolean - (optional, default: false) The flag indicating disabling stack capture.
  + disableStorage: Boolean - (optional, default: false) The flag indicating disabling storage capture.
  + enableReturnData: Boolean - (optional, default: false) The flag indicating enabling return data capture.
  + timeOut: String - (optional, default: "5s") The timeout for cancellation of execution.
  + tracer: String - (default: "structTracer") Defines the debug tracer used for given call. Supported values: structTracer, callTracer.

#### Returns

 * Array - list of intermediate state roots, where each hash corresponds to the state root after a transaction in the block.

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_intermediateRoots","params":["0x1190f352179918be580bda87e6bbe563d48ac2949e6041e5bf445dbd80a6ce60", {}],"id":1}'
````

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    "0x69aa84ad9f171ea26d1e18c2ee89b91d1447d01fc1990d165a27acd715cb7625"
  ]
}
```

</details>



## debug\_memStats

The MemStats method retrieves detailed runtime memory statistics for the Go application. It uses the runtime.ReadMemStats function to gather information about memory usage, which includes data about allocated memory, garbage collection statistics, and other relevant memory metrics.

#### Parameters

None

#### Returns

 DATA - this method returns a pointer to a runtime.MemStats structure, which contains detailed memory usage statistics for the Go runtime. This structure includes various memory statistics such as memory allocated, memory in use, garbage collection statistics, and more. This structure holds memory statistics related to the Go runtime. It includes fields such as:
  * Alloc: QUANTITY - the number of bytes allocated and still in use.
  * TotalAlloc: QUANTITY - the total number of bytes allocated (even if freed).
  * Sys: QUANTITY - the number of bytes obtained from the system (e.g., through malloc).
  * HeapAlloc: QUANTITY - the number of bytes allocated in the heap.
  * HeapSys: QUANTITY - the number of bytes obtained from the system for the heap.
  * HeapIdle: QUANTITY - the number of bytes in idle heap memory...


#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_memStats","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result":
  {
    "Alloc": 218673240,
    "TotalAlloc": 4970059624,
    "Sys": 407228760,
    "Lookups": 0,
    "Mallocs": 61929646,
    "Frees": 61101904,
    "HeapAlloc": 218673240,
    "HeapSys": 391839744,
    "HeapIdle": 167133184,
    "HeapInuse": 224706560,
    "HeapReleased": 149348352,
    "HeapObjects": 827742,
    "StackInuse": 2424832,
    "StackSys": 2424832,
    "MSpanInuse": 1589920,
    "MSpanSys": 2219520,
    "MCacheInuse": 19200,
    "MCacheSys": 31200,
    "BuckHashSys": 2341352,
    "GCSys": 5415344,
    "OtherSys": 2956768,
    "NextGC": 314778496,
    "LastGC": 1737310010035059382,
    "PauseTotalNs": 5826192,
    "PauseNs": [
      59833,
      49413,
      72438,
      83358,
      59433,
      72386,
      115829,
      122120,
      88869,
      126068,
      74430,
      75442,
      64351,
      70603,
      52079,
      75071,
      71955,
      88707,
      109375,
      50015,
      65303,
      69389,
      106027,
      73908,
      83656,
      108793,
      80281,
      55805,
      75341,
      104706,
      52148,
      75973,
      45625,
      63941,
      73628,
      116900,
      55073,
      87353,
      73157,
      100169,
      75862,
      73989,
      114686,
      91571,
      68369,
      103825,
      62457,
      140694,
      73047,
      62728,
      77205,
      114766,
      78357,
      65744,
      73158,
      74009,
      53920,
      75392,
      83207,
      84239,
      191239,
      75962,
      94939,
      47098,
      63810,
      56606,
      78898,
      74320,
      64421,
      55856,
      53721,
      72567,
      100579,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0
    ],
    "PauseEnd": [
      1737302169260706435,
      1737302169266109565,
      1737302169279860625,
      1737302169295231118,
      1737302169314015388,
      1737302169514002368,
      1737302169541038256,
      1737302289544788587,
      1737302409583617883,
      1737302529600845601,
      1737302649605683551,
      1737302769615312556,
      1737302889622296609,
      1737303009628718680,
      1737303129636033208,
      1737303249644070290,
      1737303369651111752,
      1737303489659027321,
      1737303609665515504,
      1737303729671150074,
      1737303849681103984,
      1737303937054701511,
      1737304057528651913,
      1737304177604760753,
      1737304298008958620,
      1737304418327543088,
      1737304538484118725,
      1737304658604940973,
      1737304778917114673,
      1737304898941453074,
      1737305019061600456,
      1737305139183322612,
      1737305259268085334,
      1737305379485193880,
      1737305499504132479,
      1737305619518874244,
      1737305739544485367,
      1737305859584198133,
      1737305924675952764,
      1737306044745410660,
      1737306165605019086,
      1737306286147154771,
      1737306406284758751,
      1737306526605234142,
      1737306647042313636,
      1737306767360313779,
      1737306887593512264,
      1737307007605763726,
      1737307128219526012,
      1737307248324478296,
      1737307368443791026,
      1737307488512779237,
      1737307608605550895,
      1737307729071377663,
      1737307849269277819,
      1737307969486176428,
      1737308089504980720,
      1737308209583781431,
      1737308329605408878,
      1737308449720566154,
      1737308569744165919,
      1737308689963568777,
      1737308809968663534,
      1737308929975069034,
      1737309049981166395,
      1737309169986421188,
      1737309289994432603,
      1737309410002193389,
      1737309530008485413,
      1737309650014927739,
      1737309770022270399,
      1737309890030363640,
      1737310010035059382,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0,
      0
    ],
    "NumGC": 73,
    "NumForcedGC": 1,
    "GCCPUFraction": 0.000007,
    "EnableGC": true,
    "DebugGC": false,
    "BySize": [
      {
        "Size": 0,
        "Mallocs": 0,
        "Frees": 0
      },
      {
        "Size": 8,
        "Mallocs": 1434637,
        "Frees": 1412425
      },
      {
        "Size": 16,
        "Mallocs": 11172197,
        "Frees": 11004805
      },
      {
        "Size": 24,
        "Mallocs": 1703116,
        "Frees": 1677191
      },
      {
        "Size": 32,
        "Mallocs": 6952054,
        "Frees": 6837765
      },
      {
        "Size": 48,
        "Mallocs": 4853167,
        "Frees": 4778366
      },
      {
        "Size": 64,
        "Mallocs": 15921161,
        "Frees": 15705625
      },
      {
        "Size": 80,
        "Mallocs": 2813373,
        "Frees": 2765572
      },
      {
        "Size": 96,
        "Mallocs": 2055626,
        "Frees": 2025065
      },
      {
        "Size": 112,
        "Mallocs": 645061,
        "Frees": 635650
      },
      {
        "Size": 128,
        "Mallocs": 621556,
        "Frees": 612203
      },
      {
        "Size": 144,
        "Mallocs": 325084,
        "Frees": 311031
      },
      {
        "Size": 160,
        "Mallocs": 444739,
        "Frees": 437883
      },
      {
        "Size": 176,
        "Mallocs": 175105,
        "Frees": 172639
      },
      {
        "Size": 192,
        "Mallocs": 87549,
        "Frees": 85916
      },
      {
        "Size": 208,
        "Mallocs": 138823,
        "Frees": 136477
      },
      {
        "Size": 224,
        "Mallocs": 159613,
        "Frees": 157091
      },
      {
        "Size": 240,
        "Mallocs": 34048,
        "Frees": 33426
      },
      {
        "Size": 256,
        "Mallocs": 3487477,
        "Frees": 3436629
      },
      {
        "Size": 288,
        "Mallocs": 426367,
        "Frees": 419881
      },
      {
        "Size": 320,
        "Mallocs": 135003,
        "Frees": 131744
      },
      {
        "Size": 352,
        "Mallocs": 15141,
        "Frees": 14642
      },
      {
        "Size": 384,
        "Mallocs": 381068,
        "Frees": 375739
      },
      {
        "Size": 416,
        "Mallocs": 12870,
        "Frees": 12612
      },
      {
        "Size": 448,
        "Mallocs": 16385,
        "Frees": 15503
      },
      {
        "Size": 480,
        "Mallocs": 8087,
        "Frees": 7896
      },
      {
        "Size": 512,
        "Mallocs": 32071,
        "Frees": 31595
      },
      {
        "Size": 576,
        "Mallocs": 44252,
        "Frees": 43616
      },
      {
        "Size": 640,
        "Mallocs": 271192,
        "Frees": 267210
      },
      {
        "Size": 704,
        "Mallocs": 25204,
        "Frees": 24796
      },
      {
        "Size": 768,
        "Mallocs": 24561,
        "Frees": 24129
      },
      {
        "Size": 896,
        "Mallocs": 35629,
        "Frees": 35041
      },
      {
        "Size": 1024,
        "Mallocs": 22508,
        "Frees": 22166
      },
      {
        "Size": 1152,
        "Mallocs": 39896,
        "Frees": 39231
      },
      {
        "Size": 1280,
        "Mallocs": 168281,
        "Frees": 165773
      },
      {
        "Size": 1408,
        "Mallocs": 3552,
        "Frees": 3388
      },
      {
        "Size": 1536,
        "Mallocs": 18185,
        "Frees": 17931
      },
      {
        "Size": 1792,
        "Mallocs": 784,
        "Frees": 705
      },
      {
        "Size": 2048,
        "Mallocs": 6000,
        "Frees": 5837
      },
      {
        "Size": 2304,
        "Mallocs": 7305,
        "Frees": 6990
      },
      {
        "Size": 2688,
        "Mallocs": 1244,
        "Frees": 922
      },
      {
        "Size": 3072,
        "Mallocs": 6860,
        "Frees": 6678
      },
      {
        "Size": 3200,
        "Mallocs": 551,
        "Frees": 539
      },
      {
        "Size": 3456,
        "Mallocs": 514,
        "Frees": 499
      },
      {
        "Size": 4096,
        "Mallocs": 6212,
        "Frees": 6054
      },
      {
        "Size": 4864,
        "Mallocs": 776,
        "Frees": 728
      },
      {
        "Size": 5376,
        "Mallocs": 4945,
        "Frees": 4853
      },
      {
        "Size": 6144,
        "Mallocs": 1260,
        "Frees": 1192
      },
      {
        "Size": 6528,
        "Mallocs": 183,
        "Frees": 176
      },
      {
        "Size": 6784,
        "Mallocs": 63,
        "Frees": 59
      },
      {
        "Size": 6912,
        "Mallocs": 77,
        "Frees": 75
      },
      {
        "Size": 8192,
        "Mallocs": 717,
        "Frees": 674
      },
      {
        "Size": 9472,
        "Mallocs": 191,
        "Frees": 153
      },
      {
        "Size": 9728,
        "Mallocs": 33,
        "Frees": 31
      },
      {
        "Size": 10240,
        "Mallocs": 67,
        "Frees": 61
      },
      {
        "Size": 10880,
        "Mallocs": 69,
        "Frees": 58
      },
      {
        "Size": 12288,
        "Mallocs": 220,
        "Frees": 209
      },
      {
        "Size": 13568,
        "Mallocs": 166,
        "Frees": 155
      },
      {
        "Size": 14336,
        "Mallocs": 37,
        "Frees": 30
      },
      {
        "Size": 16384,
        "Mallocs": 56,
        "Frees": 41
      },
      {
        "Size": 18432,
        "Mallocs": 540,
        "Frees": 516
      }
    ]
  }
}
```

</details>



## debug\_mutexProfile 

The MutexProfile method enables mutex profiling for a specified duration, captures the profiling data, and writes it to a file. Mutex profiling helps analyze the performance of lock contention in concurrent programs by recording information about the mutexes that are held by goroutines.

#### Parameters

* file: String - the file path where the mutex profile data will be written. This file will contain the profiling information collected during the execution of the method.
* nsec: QUANTITY - the duration (in seconds) for which the mutex profiling will run. The profiling will collect data for the specified number of seconds and then stop.

#### Returns

* String - success: This method returns the absolute path of the file where the profile data has been written. This is returned as an interface{}, which in this case would be the string value representing the file path. Failure: The method returns an error if something goes wrong during the process of starting or stopping the mutex profiling, or if there is an issue with the file operations (such as an invalid file path). If the operation completes successfully, it returns nil.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_mutexProfile","params":["mutex.txt", 3],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/mutex.txt"
}
```

</details>



## debug\_preimage

The Preimage method is a debug API function that returns the preimage (original data) for a given SHA3 hash, if it is known in the store. Specifically, it retrieves the code (smart contract code) associated with the given hash.

#### Parameters

* codeHash: DATA, 32 Bytes - CodeHash

#### Returns

* Array - the bytecode corresponding to the hash.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_preimage","params":["c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": ""
}
```

</details>



## debug\_printBlock

The PrintBlock method retrieves a block by its number from the blockchain data store and returns a pretty-printed representation of the block for easier readability and debugging.

#### Parameters

* number: QUANTITY - the number of the block to retrieve. This value uniquely identifies the desired block within the blockchain.

#### Returns

* String - a string containing the pretty-printed representation of the requested block. If the block is not found, nil is returned.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_printBlock","params":[10],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "(*types.Block)(0xc00128ca80)(Block(#10):)\n"
}
```

</details>



## debug\_setBlockProfileRate

The SetBlockProfileRate method sets the rate of collection for goroutine block profile data. By specifying the rate, you can control the granularity of the profiling data. A rate of 0 disables block profiling, while a rate of 1 enables the most detailed tracking.

#### Parameters

* rate: QUANTITY - the rate at which goroutine block profiling data is collected. A value of 0 disables block profiling, while a value of 1 enables the most detailed block profile data collection. 

#### Returns

None

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_setBlockProfileRate","params":[1],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": null
}
```

</details>



## debug\_setGCPercent

The SetGCPercent method adjusts the garbage collection (GC) target percentage for the Go runtime. This target determines how much memory allocation growth will trigger garbage collection. The method also returns the previous setting, allowing you to restore it if needed. A negative value disables GC.

#### Parameters

* v: QUANTITY - the garbage collection target percentage. A positive value represents the memory growth factor as a percentage. For example, a value of 100 means GC will be triggered when the heap doubles in size. A negative value disables garbage collection entirely.

#### Returns

* QUANTITY - the method returns the previous garbage collection percentage setting as an integer, wrapped in an interface{} type.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_setGCPercent","params":[20],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": 100
}
```

</details>



## debug\_setMutexProfileFraction

The SetMutexProfileFraction method adjusts the rate at which mutex contention events are recorded for profiling. This is useful for debugging and optimizing mutex usage in Go programs by analyzing lock contention.

#### Parameters

* rate: QUANTITY - the sampling rate for mutex contention profiling:
A value of 0 disables mutex profiling entirely. A positive value specifies the fraction of mutex contention events to profile. For example, a value of 1 records all mutex contention events, while a value of 10 records 1 out of every 10 events.

#### Returns

None

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_setMutexProfileFraction","params":[5],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": null
}
```

</details>



## debug\_stacks

The Stacks method retrieves a printed representation of the stacks of all goroutines in the application. It can also optionally filter the output based on specific package filters.

#### Parameters

* filter: DATA - a pointer to a string representing a filter expression for package names. If provided, it limits the stacks returned to those matching the filter. If no filter is needed, this parameter can be nil.
#### Returns

* String - a string representation of the stacks of all goroutines, possibly filtered by the provided filter expression. The stacks are typically printed in a format suitable for diagnostic or debugging purposes.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_stacks","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "goroutine 410429 [running]:\nruntime/pprof.writeGoroutineStacks({0x2e4be60, 0xc00e205dd0})\n\t/home/XXX/sdk/go1.22.10/src/runtime/pprof/pprof.go:743 +0x6a\nruntime/pprof.writeGoroutine({0x2e4be60?, 0xc00e205dd0?}, 0x41d198?)\n\t/home/XXX/sdk/go1.22.10/src/runtime/pprof/pprof.go:732 +0x25\nruntime/pprof.(*Profile).WriteTo(0x22f09dd?, {0x2e4be60?, 0xc00e205dd0?}, 0xc00a789ba0?)\n\t/home/XXX/sdk/go1.22.10/src/runtime/pprof/pprof.go:369 +0x14b\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*DebugHandler).Stacks(0x3b9aca00?, 0x0)\n\t/home/XXX/Code/blade/jsonrpc/debug_handler.go:135 +0x51\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*Debug).Stacks.func1()\n\t/home/XXX/Code/blade/jsonrpc/debug_endpoint.go:459 +0x1f\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*Throttling).AttemptRequest(0xc00069f840, {0x2e69b90?, 0x43a1fa0?}, 0xc0006acf20)\n\t/home/XXX/Code/blade/jsonrpc/throttling.go:40 +0x107\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*Debug).Stacks(0x0?, 0x478b33?)\n\t/home/XXX/Code/blade/jsonrpc/debug_endpoint.go:456 +0x4f\nreflect.Value.call({0xc000806cc0?, 0xc000cf0310?, 0x13?}, {0x22e8281, 0x4}, {0xc00e205da0, 0x2, 0x2?})\n\t/home/XXX/sdk/go1.22.10/src/reflect/value.go:596 +0xca6\nreflect.Value.Call({0xc000806cc0?, 0xc000cf0310?, 0xc00c8d9498?}, {0xc00e205da0?, 0x1dbe580?, 0xc00b8a6828?})\n\t/home/XXX/sdk/go1.22.10/src/reflect/value.go:380 +0xb9\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*Dispatcher).handleReq(0xc0009c3800, {{0x1df5940, 0xc00c8d9498}, {0xc00c8d94b0, 0xc}, {0xc00c1e0840, 0x2, 0x20}})\n\t/home/XXX/Code/blade/jsonrpc/dispatcher.go:446 +0x4df\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*Dispatcher).Handle(0xc0009c3800, {0xc00a9e0c00, 0x3c, 0x200})\n\t/home/XXX/Code/blade/jsonrpc/dispatcher.go:365 +0x16a\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*JSONRPC).handleJSONRPCRequest(0xc000cf5170, {0x2e64240, 0xc000d1e460}, 0x1c?)\n\t/home/XXX/Code/blade/jsonrpc/jsonrpc.go:363 +0x135\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*JSONRPC).handle(0xc000cf5170, {0x2e64240, 0xc000d1e460}, 0xc00b33afc0)\n\t/home/XXX/Code/blade/jsonrpc/jsonrpc.go:342 +0x2d0\nnet/http.HandlerFunc.ServeHTTP(0xc00e205d70?, {0x2e64240?, 0xc000d1e460?}, 0x2e3d938?)\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:2171 +0x29\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*JSONRPC).setupHTTP.(*JSONRPC).setupHTTP.middlewareFactory.func4.func5({0x2e64240, 0xc000d1e460}, 0xc00b33afc0)\n\t/home/XXX/Code/blade/jsonrpc/jsonrpc.go:209 +0x154\nnet/http.HandlerFunc.ServeHTTP(0xc000cab790?, {0x2e64240?, 0xc000d1e460?}, 0x10?)\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:2171 +0x29\nnet/http.(*ServeMux).ServeHTTP(0x414605?, {0x2e64240, 0xc000d1e460}, 0xc00b33afc0)\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:2688 +0x1ad\nnet/http.serverHandler.ServeHTTP({0x2e5b348?}, {0x2e64240?, 0xc000d1e460?}, 0x6?)\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:3142 +0x8e\nnet/http.(*conn).serve(0xc0012c0900, {0x2e69bc8, 0xc0008b86f0})\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:2044 +0x5e8\ncreated by net/http.(*Server).Serve in goroutine 192\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:3290 +0x4b4\n\ngoroutine 1 [chan receive, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/command/helper.HandleSignals(0xc0006ca0b0, {0x2e6e220, 0xc000cb6ca0})\n\t/home/XXX/Code/blade/command/helper/helper.go:55 +0xe8\ngithub.com/0xPolygon/polygon-edge/command/server.runServerLoop(0x4115ca0?, {0x2e6e220, 0xc000cb6ca0})\n\t/home/XXX/Code/blade/command/server/server.go:382 +0x70\ngithub.com/0xPolygon/polygon-edge/command/server.runCommand(0xc000c59808?, {0x22e815d?, 0x4?, 0x22e8161?})\n\t/home/XXX/Code/blade/command/server/server.go:365 +0x3b\ngithub.com/spf13/cobra.(*Command).execute(0xc000c59808, {0xc000c4a3c0, 0x13, 0x13})\n\t/home/XXX/go/pkg/mod/github.com/spf13/cobra@v1.8.1/command.go:989 +0xab1\ngithub.com/spf13/cobra.(*Command).ExecuteC(0xc000c50008)\n\t/home/XXX/go/pkg/mod/github.com/spf13/cobra@v1.8.1/command.go:1117 +0x3ff\ngithub.com/spf13/cobra.(*Command).Execute(...)\n\t/home/XXX/go/pkg/mod/github.com/spf13/cobra@v1.8.1/command.go:1041\ngithub.com/0xPolygon/polygon-edge/command/root.(*RootCommand).Execute(0xc0000061c0?)\n\t/home/XXX/Code/blade/command/root/root.go:68 +0x16\nmain.main()\n\t/home/XXX/Code/blade/main.go:18 +0x50\n\ngoroutine 26 [select]:\ngo.opencensus.io/stats/view.(*worker).start(0xc0005bba80)\n\t/home/XXX/go/pkg/mod/go.opencensus.io@v0.24.0/stats/view/worker.go:292 +0x9f\ncreated by go.opencensus.io/stats/view.init.0 in goroutine 1\n\t/home/XXX/go/pkg/mod/go.opencensus.io@v0.24.0/stats/view/worker.go:34 +0x8d\n\ngoroutine 27 [sync.Cond.Wait, 487 minutes]:\nsync.runtime_notifyListWait(0xc000aeec10, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/runtime/sema.go:569 +0x159\nsync.(*Cond).Wait(0x0?)\n\t/home/XXX/sdk/go1.22.10/src/sync/cond.go:70 +0x85\ngithub.com/cihub/seelog.(*asyncLoopLogger).processItem(0xc000985dd0)\n\t/home/XXX/go/pkg/mod/github.com/cihub/seelog@v0.0.0-20170130134532-f561c5e57575/behavior_asynclooplogger.go:50 +0x99\ngithub.com/cihub/seelog.(*asyncLoopLogger).processQueue(0xc000985dd0)\n\t/home/XXX/go/pkg/mod/github.com/cihub/seelog@v0.0.0-20170130134532-f561c5e57575/behavior_asynclooplogger.go:63 +0x33\ncreated by github.com/cihub/seelog.NewAsyncLoopLogger in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/cihub/seelog@v0.0.0-20170130134532-f561c5e57575/behavior_asynclooplogger.go:40 +0xcf\n\ngoroutine 28 [sync.Cond.Wait, 487 minutes]:\nsync.runtime_notifyListWait(0xc000aeed90, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/runtime/sema.go:569 +0x159\nsync.(*Cond).Wait(0x0?)\n\t/home/XXX/sdk/go1.22.10/src/sync/cond.go:70 +0x85\ngithub.com/cihub/seelog.(*asyncLoopLogger).processItem(0xc000985ef0)\n\t/home/XXX/go/pkg/mod/github.com/cihub/seelog@v0.0.0-20170130134532-f561c5e57575/behavior_asynclooplogger.go:50 +0x99\ngithub.com/cihub/seelog.(*asyncLoopLogger).processQueue(0xc000985ef0)\n\t/home/XXX/go/pkg/mod/github.com/cihub/seelog@v0.0.0-20170130134532-f561c5e57575/behavior_asynclooplogger.go:63 +0x33\ncreated by github.com/cihub/seelog.NewAsyncLoopLogger in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/cihub/seelog@v0.0.0-20170130134532-f561c5e57575/behavior_asynclooplogger.go:40 +0xcf\n\ngoroutine 149 [chan receive, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network.(*Topic).readLoop.func1()\n\t/home/XXX/Code/blade/network/gossip.go:92 +0x26\ncreated by github.com/0xPolygon/polygon-edge/network.(*Topic).readLoop in goroutine 178\n\t/home/XXX/Code/blade/network/gossip.go:91 +0xfd\n\ngoroutine 9 [select]:\ngithub.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem.(*memoryAddrBook).background(0xc000c3a780, {0x2e69c00, 0xc000ba9bd0})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/peerstore/pstoremem/addr_book.go:242 +0x125\ncreated by github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem.NewAddrBook in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/peerstore/pstoremem/addr_book.go:205 +0x1c5\n\ngoroutine 10 [select]:\ngithub.com/libp2p/go-libp2p/p2p/host/resource-manager.(*resourceManager).background(0xc000001ec0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/resource-manager/rcmgr.go:424 +0x110\ncreated by github.com/libp2p/go-libp2p/p2p/host/resource-manager.NewResourceManager in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/resource-manager/rcmgr.go:212 +0xba7\n\ngoroutine 11 [select]:\ngithub.com/libp2p/go-libp2p/p2p/net/connmgr.(*decayer).process(0xc00067c150)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/connmgr/decay.go:164 +0x213\ncreated by github.com/libp2p/go-libp2p/p2p/net/connmgr.NewDecayer in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/connmgr/decay.go:96 +0x245\n\ngoroutine 12 [select]:\ngithub.com/libp2p/go-libp2p/p2p/net/connmgr.(*BasicConnMgr).background(0xc0009ef608)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/connmgr/connmgr.go:359 +0x13b\ncreated by github.com/libp2p/go-libp2p/p2p/net/connmgr.NewConnManager in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/connmgr/connmgr.go:153 +0x376\n\ngoroutine 64 [select]:\ngithub.com/libp2p/go-libp2p/p2p/transport/quicreuse.(*reuse).gc(0xc0001c6c80)\n\t/home/XXX/go/pkg/mod/github.com/li"/home/XXX/Code/blade/WriteBlockProfile.txt"bp2p/go-libp2p@v0.38.1/p2p/transport/quicreuse/reuse.go:194 +0x110\ncreated by github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/transport/quicreuse/reuse.go:169 +0x145\n\ngoroutine 65 [select]:\ngithub.com/libp2p/go-libp2p/p2p/transport/quicreuse.(*reuse).gc(0xc0001c6cd0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/transport/quicreuse/reuse.go:194 +0x110\ncreated by github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/transport/quicreuse/reuse.go:169 +0x145\n\ngoroutine 78 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*connectednessEventEmitter).runEmitter(0xc000c1b830)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/connectedness_event_emitter.go:93 +0x125\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.newConnectednessEventEmitter in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/connectedness_event_emitter.go:47 +0x185\n\ngoroutine 79 [select, 2 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*DialBackoff).background(0xc0001ccd50, {0x2e69c00, 0xc000536000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_dial.go:128 +0xd7\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*DialBackoff).init in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_dial.go:121 +0xb0\n\ngoroutine 88 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/protocol/identify.(*ObservedAddrManager).worker(0xc000c1bcb0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/identify/obsaddr.go:329 +0x10f\ncreated by github.com/libp2p/go-libp2p/p2p/protocol/identify.NewObservedAddrManager in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/identify/obsaddr.go:191 +0x1d8\n\ngoroutine 294 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc0010422a0, {0xc0008a7558, 0x1, 0x1})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x4127b0?, {0xc0008a7558?, 0xae20ab?, 0xc000503340?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc000b9b100, {0xc0008a7558?, 0x27?, 0x47a5b2?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\nio.ReadAtLeast({0x7646a40400b8, 0xc000b9b100}, {0xc0008a7558, 0x1, 0x1}, 0x1)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-msgio.(*simpleByteReader).ReadByte(0xc0008a7548)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:185 +0x31\ngithub.com/multiformats/go-varint.ReadUvarint({0x2e4d620, 0xc0008a7548})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-varint@v0.0.7/varint.go:80 +0x51\ngithub.com/libp2p/go-msgio.(*varintReader).nextMsgLen(0xc000d39e00)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:119 +0x2a\ngithub.com/libp2p/go-msgio.(*varintReader).ReadMsg(0xc000d39e00)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:149 +0xbb\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewStream(0xc00084a908, {0x2e82680, 0xc000b9b100})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:66 +0x3d7\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).SetStreamHandler.func1({0x40f24b0?, 0x222e120?}, {0x7646a4040068?, 0xc000b9b100?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:659 +0x82\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler(0xc000bfdc20, {0x2e82680, 0xc000b9b100})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:487 +0x7e9\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1.1()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:142 +0xa7\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1 in goroutine 306\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:128 +0x1b7\n\ngoroutine 90 [select]:\ngithub.com/libp2p/go-libp2p/p2p/protocol/identify.(*natEmitter).worker(0xc0003279d0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/identify/nat_emitter.go:62 +0x18a\ncreated by github.com/libp2p/go-libp2p/p2p/protocol/identify.newNATEmitter in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/identify/nat_emitter.go:51 +0x327\n\ngoroutine 91 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client.(*Listener).Accept(0xc000807da0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/circuitv2/client/listen.go:21 +0x131\ngithub.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).handleIncoming(0xc000327a40)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/upgrader/listener.go:75 +0xfe\ncreated by github.com/libp2p/go-libp2p/p2p/net/upgrader.(*upgrader).UpgradeListener in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/upgrader/upgrader.go:119 +0x1c5\n\ngoroutine 92 [chan receive, 487 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).Accept(0xc000327a40)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/upgrader/listener.go:173 +0x3a\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr.func2()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_listen.go:161 +0x109\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_listen.go:139 +0x225\n\ngoroutine 150 [select]:\ngithub.com/0xPolygon/polygon-edge/blockchain.(*subscription).GetEvent(0xc000b23fd0?)\n\t/home/XXX/Code/blade/blockchain/subscription.go:50 +0x5c\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*FilterManager).Run.func1()\n\t/home/XXX/Code/blade/jsonrpc/filter_manager.go:366 +0x49\ncreated by github.com/0xPolygon/polygon-edge/jsonrpc.(*FilterManager).Run in goroutine 191\n\t/home/XXX/Code/blade/jsonrpc/filter_manager.go:364 +0xa9\n\ngoroutine 94 [IO wait, 487 minutes]:\ninternal/poll.runtime_pollWait(0x7646a555a7c0, 0x72)\n\t/home/XXX/sdk/go1.22.10/src/runtime/netpoll.go:345 +0x85\ninternal/poll.(*pollDesc).wait(0x3?, 0x0?, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:84 +0x27\ninternal/poll.(*pollDesc).waitRead(...)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:89\ninternal/poll.(*FD).Accept(0xc000b0e980)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_unix.go:611 +0x2ac\nnet.(*netFD).accept(0xc000b0e980)\n\t/home/XXX/sdk/go1.22.10/src/net/fd_unix.go:172 +0x29\nnet.(*TCPListener).accept(0xc000ae4700)\n\t/home/XXX/sdk/go1.22.10/src/net/tcpsock_posix.go:159 +0x1e\nnet.(*TCPListener).Accept(0xc000ae4700)\n\t/home/XXX/sdk/go1.22.10/src/net/tcpsock.go:327 +0x30\ngithub.com/multiformats/go-multiaddr/net.(*maListener).Accept(0xfc5b0f?)\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-multiaddr@v0.14.0/net/net.go:243 +0x24\ngithub.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).handleIncoming(0xc000327b20)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/upgrader/listener.go:75 +0xfe\ncreated by github.com/libp2p/go-libp2p/p2p/net/upgrader.(*upgrader).UpgradeListener in goroutine 93\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/upgrader/upgrader.go:119 +0x1c5\n\ngoroutine 95 [chan receive, 487 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).Accept(0xc000327b20)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/upgrader/listener.go:173 +0x3a\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr.func2()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_listen.go:161 +0x109\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr in goroutine 93\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_listen.go:139 +0x225\n\ngoroutine 97 [select]:\ngithub.com/libp2p/go-libp2p/p2p/host/pstoremanager.(*PeerstoreManager).background(0xc0008073e0, {0x2e69c00, 0xc0007dc140}, {0x2e66b70, 0xc000807f20})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/pstoremanager/pstoremanager.go:98 +0x2a5\ncreated by github.com/libp2p/go-libp2p/p2p/host/pstoremanager.(*PeerstoreManager).Start in goroutine 93\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/pstoremanager/pstoremanager.go:80 +0x213\n\ngoroutine 98 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).loop(0xc0003248c0, {0x2e69c00, 0xc0007bf270})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/identify/id.go:291 +0x43a\ncreated by github.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).Start in goroutine 93\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/identify/id.go:254 +0x190\n\ngoroutine 99 [select]:\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).background(0xc000bfdc20)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:612 +0x1f5\ncreated by github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).Start in goroutine 93\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:439 +0x105\n\ngoroutine 151 [select, 449 minutes]:\ngithub.com/0xPolygon/polygon-edge/txpool.(*eventSubscription).runLoop(0xc000d38280)\n\t/home/XXX/Code/blade/txpool/event_subscription.go:48 +0xb5\ncreated by github.com/0xPolygon/polygon-edge/txpool.(*eventManager).subscribe in goroutine 191\n\t/home/XXX/Code/blade/txpool/event_manager.go:52 +0x265\n\ngoroutine 108 [select, 1 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/host/autonat.(*AmbientAutoNAT).background(0xc0008cde10)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/autonat/autonat.go:187 +0x37c\ncreated by github.com/libp2p/go-libp2p/p2p/host/autonat.New in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/autonat/autonat.go:138 +0x6e5\n\ngoroutine 109 [select]:\ngithub.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem.(*memoryAddrBook).background(0xc000b0f080, {0x2e69c00, 0xc0007dc640})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/peerstore/pstoremem/addr_book.go:242 +0x125\ncreated by github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem.NewAddrBook in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/peerstore/pstoremem/addr_book.go:205 +0x1c5\n\ngoroutine 110 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*backoff).cleanupLoop(0xc0009c4300, {0x2e69b90, 0x43a1fa0})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/backoff.go:99 +0xd7\ncreated by github.com/libp2p/go-libp2p-pubsub.newBackoff in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/backoff.go:46 +0xdd\n\ngoroutine 111 [select]:\ngithub.com/libp2p/go-libp2p-pubsub/timecache.background({0x2e69c00, 0xc0007dc730}, {0x2e5d418, 0xc0009c4510}, 0xc0009c44e0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/timecache/util.go:16 +0x148\ncreated by github.com/libp2p/go-libp2p-pubsub/timecache.newFirstSeenCache in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/timecache/first_seen_cache.go:28 +0x125\n\ngoroutine 112 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).heartbeatTimer(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1503 +0x1e5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:543 +0x1b6\n\ngoroutine 113 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1092 +0xc5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:547 +0x1c5\n\ngoroutine 114 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1092 +0xc5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:547 +0x1c5\n\ngoroutine 115 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1092 +0xc5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:547 +0x1c5\n\ngoroutine 116 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1092 +0xc5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:547 +0x1c5\n\ngoroutine 117 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1092 +0xc5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:547 +0x1c5\n\ngoroutine 118 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1092 +0xc5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:547 +0x1c5\n\ngoroutine 119 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1092 +0xc5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:547 +0x1c5\n\ngoroutine 120 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:1092 +0xc5\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:547 +0x1c5\n\ngoroutine 121 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).manageAddrBook(0xc00084a6c8)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:578 +0x279\ncreated by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/gossipsub.go:551 +0x256\n\ngoroutine 122 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).watchForNewPeers(0xc00084a908, {0x2e69b90, 0x43a1fa0})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/peer_notify.go:69 +0x585\ncreated by github.com/libp2p/go-libp2p-pubsub.NewPubSub in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/pubsub.go:333 +0xddf\n\ngoroutine 123 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 124 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 125 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 126 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 127 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 128 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 129 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 130 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 131 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 132 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 133 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 134 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 135 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 136 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 137 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 138 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0007dc690)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:283 +0xbf\ncreated by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/validation.go:136 +0x66\n\ngoroutine 139 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).processLoop(0xc00084a908, {0x2e69b90, 0x43a1fa0})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/pubsub.go:574 +0x4bc\ncreated by github.com/libp2p/go-libp2p-pubsub.NewPubSub in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/pubsub.go:337 +0xe52\n\ngoroutine 140 [select, 3 minutes]:\ngithub.com/syndtr/goleveldb/leveldb.(*session).refLoop(0xc0002a73b0)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/session_util.go:189 +0x59e\ncreated by github.com/syndtr/goleveldb/leveldb.newSession in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/session.go:93 +0x296\n\ngoroutine 148 [select, 487 minutes]:\ngithub.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).loop.func1()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/identify/id.go:281 +0xe7\ncreated by github.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).loop in goroutine 98\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/protocol/identify/id.go:277 +0x3a5\n\ngoroutine 162 [select, 38 minutes]:\ngithub.com/syndtr/goleveldb/leveldb.(*DB).compactionError(0xc000682fc0)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db_compaction.go:92 +0x149\ncreated by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db.go:148 +0x447\n\ngoroutine 163 [select]:\ngithub.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain(0xc000682fc0)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db_state.go:101 +0x9c\ncreated by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db.go:149 +0x485\n\ngoroutine 164 [select, 38 minutes]:\ngithub.com/syndtr/goleveldb/leveldb.(*DB).tCompaction(0xc000682fc0)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db_compaction.go:845 +0x6ca\ncreated by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db.go:157 +0x50f\n\ngoroutine 165 [select, 43 minutes]:\ngithub.com/syndtr/goleveldb/leveldb.(*DB).mCompaction(0xc000682fc0)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db_compaction.go:782 +0x105\ncreated by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db.go:158 +0x54b\n\ngoroutine 166 [select, 2 minutes]:\ngithub.com/syndtr/goleveldb/leveldb.(*session).refLoop(0xc0002a7e00)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/session_util.go:189 +0x59e\ncreated by github.com/syndtr/goleveldb/leveldb.newSession in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/session.go:93 +0x296\n\ngoroutine 141 [select, 487 minutes]:\ngithub.com/syndtr/goleveldb/leveldb.(*DB).compactionError(0xc000603a40)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db_compaction.go:92 +0x149\ncreated by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db.go:148 +0x447\n\ngoroutine 142 [select]:\ngithub.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain(0xc000603a40)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db_state.go:101 +0x9c\ncreated by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db.go:149 +0x485\n\ngoroutine 143 [select, 487 minutes]:\ngithub.com/syndtr/goleveldb/leveldb.(*DB).tCompaction(0xc000603a40)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db_compaction.go:845 +0x6ca\ncreated by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db.go:157 +0x50f\n\ngoroutine 144 [select, 487 minutes]:\ngithub.com/syndtr/goleveldb/leveldb.(*DB).mCompaction(0xc000603a40)\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db_compaction.go:782 +0x105\ncreated by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1\n\t/home/XXX/go/pkg/mod/github.com/syndtr/goleveldb@v1.0.1-0.20220721030215-126854af5e6d/leveldb/db.go:158 +0x54b\n\ngoroutine 145 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/accounts.(*Manager).update(0xc000334070)\n\t/home/XXX/Code/blade/accounts/manager.go:118 +0x125\ncreated by github.com/0xPolygon/polygon-edge/accounts.NewManager in goroutine 1\n\t/home/XXX/Code/blade/accounts/manager.go:83 +0x587\n\ngoroutine 178 [select, 449 minutes]:\ngithub.com/libp2p/go-libp2p-pubsub.(*Subscription).Next(0xc000acc140, {0x2e69c00, 0xc0007e0d70})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/subscription.go:26 +0x85\ngithub.com/0xPolygon/polygon-edge/network.(*Topic).readLoop(0xc000acc0f0, 0xc000acc140, 0xc000c8e2c0)\n\t/home/XXX/Code/blade/network/gossip.go:97 +0x114\ncreated by github.com/0xPolygon/polygon-edge/network.(*Topic).Subscribe in goroutine 1\n\t/home/XXX/Code/blade/network/gossip.go:80 +0xda\n\ngoroutine 179 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*Subscription).Next(0xc000accf50, {0x2e69c00, 0xc0007dc190})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/subscription.go:26 +0x85\ngithub.com/0xPolygon/polygon-edge/network.(*Topic).readLoop(0xc000acc550, 0xc000accf50, 0xc00069f2c0)\n\t/home/XXX/Code/blade/network/gossip.go:97 +0x114\ncreated by github.com/0xPolygon/polygon-edge/network.(*Topic).Subscribe in goroutine 1\n\t/home/XXX/Code/blade/network/gossip.go:80 +0xda\n\ngoroutine 194 [chan receive, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network.(*Topic).readLoop.func1()\n\t/home/XXX/Code/blade/network/gossip.go:92 +0x26\ncreated by github.com/0xPolygon/polygon-edge/network.(*Topic).readLoop in goroutine 179\n\t/home/XXX/Code/blade/network/gossip.go:91 +0xfd\n\ngoroutine 180 [IO wait, 487 minutes]:\ninternal/poll.runtime_pollWait(0x7646a555a6c8, 0x72)\n\t/home/XXX/sdk/go1.22.10/src/runtime/netpoll.go:345 +0x85\ninternal/poll.(*pollDesc).wait(0x11?, 0x0?, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:84 +0x27\ninternal/poll.(*pollDesc).waitRead(...)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:89\ninternal/poll.(*FD).Accept(0xc000bb6a80)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_unix.go:611 +0x2ac\nnet.(*netFD).accept(0xc000bb6a80)\n\t/home/XXX/sdk/go1.22.10/src/net/fd_unix.go:172 +0x29\nnet.(*TCPListener).accept(0xc0009e73e0)\n\t/home/XXX/sdk/go1.22.10/src/net/tcpsock_posix.go:159 +0x1e\nnet.(*TCPListener).Accept(0xc0009e73e0)\n\t/home/XXX/sdk/go1.22.10/src/net/tcpsock.go:327 +0x30\ngoogle.golang.org/grpc.(*Server).Serve(0xc0001cc000, {0x2e63ae0, 0xc0009e73e0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:885 +0x49e\ngithub.com/0xPolygon/polygon-edge/server.(*Server).setupGRPC.func1()\n\t/home/XXX/Code/blade/server/server.go:1143 +0x28\ncreated by github.com/0xPolygon/polygon-edge/server.(*Server).setupGRPC in goroutine 1\n\t/home/XXX/Code/blade/server/server.go:1142 +0xfa\n\ngoroutine 181 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Accept(0xc0009e7580)\n\t/home/XXX/Code/blade/network/grpc/grpc.go:108 +0x7a\ngoogle.golang.org/grpc.(*Server).Serve(0xc000b3e800, {0x2e64fa0, 0xc0009e7580})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:885 +0x49e\ngithub.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Serve.func1()\n\t/home/XXX/Code/blade/network/grpc/grpc.go:83 +0x25\ncreated by github.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Serve in goroutine 1\n\t/home/XXX/Code/blade/network/grpc/grpc.go:82 +0x4f\n\ngoroutine 183 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network.(*Server).Subscribe.func1()\n\t/home/XXX/Code/blade/network/server.go:674 +0x165\ncreated by github.com/0xPolygon/polygon-edge/network.(*Server).Subscribe in goroutine 1\n\t/home/XXX/Code/blade/network/server.go:670 +0x10c\n\ngoroutine 184 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Accept(0xc0009e7740)\n\t/home/XXX/Code/blade/network/grpc/grpc.go:108 +0x7a\ngoogle.golang.org/grpc.(*Server).Serve(0xc000b3f000, {0x2e64fa0, 0xc0009e7740})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:885 +0x49e\ngithub.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Serve.func1()\n\t/home/XXX/Code/blade/network/grpc/grpc.go:83 +0x25\ncreated by github.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Serve in goroutine 1\n\t/home/XXX/Code/blade/network/grpc/grpc.go:82 +0x4f\n\ngoroutine 268 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc000d1e000, {0xc0010043d0, 0x1, 0x1})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0xc00c97c0a0?, {0xc0010043d0?, 0xae20ab?, 0x0?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc00091d600, {0xc0010043d0?, 0x8?, 0xc000cd3a28?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\nio.ReadAtLeast({0x7646a40400b8, 0xc00091d600}, {0xc0010043d0, 0x1, 0x1}, 0x1)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-msgio.(*simpleByteReader).ReadByte(0xc0010043c0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:185 +0x31\ngithub.com/multiformats/go-varint.ReadUvarint({0x2e4d620, 0xc0010043c0})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-varint@v0.0.7/varint.go:80 +0x51\ngithub.com/libp2p/go-msgio.(*varintReader).nextMsgLen(0xc00082a1c0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:119 +0x2a\ngithub.com/libp2p/go-msgio.(*varintReader).ReadMsg(0xc00082a1c0)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:149 +0xbb\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewStream(0xc00084a908, {0x2e82680, 0xc00091d600})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:66 +0x3d7\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).SetStreamHandler.func1({0x40f24b0?, 0x222e120?}, {0x7646a4040068?, 0xc00091d600?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:659 +0x82\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler(0xc000bfdc20, {0x2e82680, 0xc00091d600})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:487 +0x7e9\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1.1()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:142 +0xa7\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1 in goroutine 267\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:128 +0x1b7\n\ngoroutine 262 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Session).sendLoop(0xc001028100)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:583 +0x5a7\ngithub.com/libp2p/go-yamux/v4.(*Session).send(0xc001028100)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:512 +0x18\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 258\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:163 +0x556\n\ngoroutine 263 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Session).startMeasureRTT(0xc001028100)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:356 +0xca\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 258\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:164 +0x596\n\ngoroutine 188 [select]:\ngithub.com/0xPolygon/polygon-edge/network/discovery.(*DiscoveryService).startDiscovery(0xc000a79b90)\n\t/home/XXX/Code/blade/network/discovery/discovery.go:278 +0xfd\ncreated by github.com/0xPolygon/polygon-edge/network/discovery.(*DiscoveryService).Start in goroutine 1\n\t/home/XXX/Code/blade/network/discovery/discovery.go:116 +0x4f\n\ngoroutine 189 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network/dial.(*DialQueue).Wait(0xc000307c40, {0x2e69c00?, 0xc0007e0e60?})\n\t/home/XXX/Code/blade/network/dial/dial_queue.go:42 +0x85\ngithub.com/0xPolygon/polygon-edge/network.(*Server).runDial(0xc000b82340)\n\t/home/XXX/Code/blade/network/server.go:385 +0x1fa\ncreated by github.com/0xPolygon/polygon-edge/network.(*Server).Start in goroutine 1\n\t/home/XXX/Code/blade/network/server.go:270 +0x290\n\ngoroutine 190 [select]:\ngithub.com/0xPolygon/polygon-edge/network.(*Server).keepAliveMinimumPeerConnections(0xc000b82340)\n\t/home/XXX/Code/blade/network/server.go:331 +0x66\ncreated by github.com/0xPolygon/polygon-edge/network.(*Server).Start in goroutine 1\n\t/home/XXX/Code/blade/network/server.go:271 +0x2d2\n\ngoroutine 191 [select]:\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*FilterManager).Run(0xc000c84870)\n\t/home/XXX/Code/blade/jsonrpc/filter_manager.go:399 +0x26a\ncreated by github.com/0xPolygon/polygon-edge/jsonrpc.newDispatcher in goroutine 1\n\t/home/XXX/Code/blade/jsonrpc/dispatcher.go:90 +0x18f\n\ngoroutine 211 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network.(*Server).Subscribe.func1()\n\t/home/XXX/Code/blade/network/server.go:674 +0x165\ncreated by github.com/0xPolygon/polygon-edge/network.(*Server).Subscribe in goroutine 189\n\t/home/XXX/Code/blade/network/server.go:670 +0x10c\n\ngoroutine 26042 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00ac9f150, {0x2e69c00, 0xc00d2f80a0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 26040\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 261 [IO wait]:\ninternal/poll.runtime_pollWait(0x7646a555a3e0, 0x72)\n\t/home/XXX/sdk/go1.22.10/src/runtime/netpoll.go:345 +0x85\ninternal/poll.(*pollDesc).wait(0xc00091c280?, 0xc001027000?, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:84 +0x27\ninternal/poll.(*pollDesc).waitRead(...)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:89\ninternal/poll.(*FD).Read(0xc00091c280, {0xc001027000, 0x1000, 0x1000})\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_unix.go:164 +0x27a\nnet.(*netFD).Read(0xc00091c280, {0xc001027000?, 0xc0011f2058?, 0xc00d23b8e0?})\n\t/home/XXX/sdk/go1.22.10/src/net/fd_posix.go:55 +0x25\nnet.(*conn).Read(0xc00100e000, {0xc001027000?, 0xc00a697ca0?, 0xc00a697d50?})\n\t/home/XXX/sdk/go1.22.10/src/net/net.go:185 +0x45\nbufio.(*Reader).Read(0xc0010000c0, {0xc00027bb90, 0x2, 0xc00a697d80?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc0010000c0}, {0xc00027bb90, 0x2, 0x2}, 0x2)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-libp2p/p2p/security/noise.(*secureSession).readNextInsecureMsgLen(0xc00027bb00)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/security/noise/rw.go:133 +0x35\ngithub.com/libp2p/go-libp2p/p2p/security/noise.(*secureSession).Read(0xc00027bb00, {0xc0008b7cd0, 0xc, 0xc})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/security/noise/rw.go:52 +0x1fc\nio.ReadAtLeast({0x7646a40001c8, 0xc00027bb00}, {0xc0008b7cd0, 0xc, 0xc}, 0xc)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-yamux/v4.(*Session).recvLoop(0xc001028100)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:668 +0xf3\ngithub.com/libp2p/go-yamux/v4.(*Session).recv(0xc001028100)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:640 +0x18\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 258\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:162 +0x516\n\ngoroutine 26027 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc0011f1c40, {0x2e69c00, 0xc009e0f090})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 26025\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 192 [IO wait]:\ninternal/poll.runtime_pollWait(0x7646a555a4d8, 0x72)\n\t/home/XXX/sdk/go1.22.10/src/runtime/netpoll.go:345 +0x85\ninternal/poll.(*pollDesc).wait(0x12?, 0x1?, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:84 +0x27\ninternal/poll.(*pollDesc).waitRead(...)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:89\ninternal/poll.(*FD).Accept(0xc000bb7080)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_unix.go:611 +0x2ac\nnet.(*netFD).accept(0xc000bb7080)\n\t/home/XXX/sdk/go1.22.10/src/net/fd_unix.go:172 +0x29\nnet.(*TCPListener).accept(0xc000d017a0)\n\t/home/XXX/sdk/go1.22.10/src/net/tcpsock_posix.go:159 +0x1e\nnet.(*TCPListener).Accept(0xc000d017a0)\n\t/home/XXX/sdk/go1.22.10/src/net/tcpsock.go:327 +0x30\nnet/http.(*Server).Serve(0xc0002a7d10, {0x2e63ae0, 0xc000d017a0})\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:3260 +0x33e\ngithub.com/0xPolygon/polygon-edge/jsonrpc.(*JSONRPC).setupHTTP.func3()\n\t/home/XXX/Code/blade/jsonrpc/jsonrpc.go:155 +0x2c\ncreated by github.com/0xPolygon/polygon-edge/jsonrpc.(*JSONRPC).setupHTTP in goroutine 1\n\t/home/XXX/Code/blade/jsonrpc/jsonrpc.go:154 +0x79f\n\ngoroutine 193 [select]:\ngithub.com/0xPolygon/polygon-edge/syncer.(*syncPeerClient).startNewBlockProcess(0xc000984bd0)\n\t/home/XXX/Code/blade/syncer/client.go:244 +0xc5\ncreated by github.com/0xPolygon/polygon-edge/syncer.(*syncPeerClient).Start in goroutine 1\n\t/home/XXX/Code/blade/syncer/client.go:73 +0x58\n\ngoroutine 242 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/syncer.(*syncPeerClient).startPeerEventProcess(0xc000984bd0)\n\t/home/XXX/Code/blade/syncer/client.go:278 +0x19f\ncreated by github.com/0xPolygon/polygon-edge/syncer.(*syncPeerClient).Start in goroutine 1\n\t/home/XXX/Code/blade/syncer/client.go:74 +0x96\n\ngoroutine 243 [select]:\ngithub.com/libp2p/go-libp2p-pubsub.(*Subscription).Next(0xc000acdbd0, {0x2e69c00, 0xc000c8c000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/subscription.go:26 +0x85\ngithub.com/0xPolygon/polygon-edge/network.(*Topic).readLoop(0xc000acdb30, 0xc000acdbd0, 0xc000cafa40)\n\t/home/XXX/Code/blade/network/gossip.go:97 +0x114\ncreated by github.com/0xPolygon/polygon-edge/network.(*Topic).Subscribe in goroutine 1\n\t/home/XXX/Code/blade/network/gossip.go:80 +0xda\n\ngoroutine 244 [select]:\ngithub.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Accept(0xc0000c72c0)\n\t/home/XXX/Code/blade/network/grpc/grpc.go:108 +0x7a\ngoogle.golang.org/grpc.(*Server).Serve(0xc000b3f200, {0x2e64fa0, 0xc0000c72c0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:885 +0x49e\ngithub.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Serve.func1()\n\t/home/XXX/Code/blade/network/grpc/grpc.go:83 +0x25\ncreated by github.com/0xPolygon/polygon-edge/network/grpc.(*GrpcStream).Serve in goroutine 1\n\t/home/XXX/Code/blade/network/grpc/grpc.go:82 +0x4f\n\ngoroutine 245 [chan receive]:\ngithub.com/0xPolygon/polygon-edge/syncer.(*syncer).startPeerStatusUpdateProcess(0xc000bb6300)\n\t/home/XXX/Code/blade/syncer/syncer.go:109 +0x94\ncreated by github.com/0xPolygon/polygon-edge/syncer.(*syncer).Start in goroutine 1\n\t/home/XXX/Code/blade/syncer/syncer.go:82 +0x8a\n\ngoroutine 246 [chan receive, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/syncer.(*syncer).startPeerConnectionEventProcess(0xc000bb6300)\n\t/home/XXX/Code/blade/syncer/syncer.go:116 +0x47\ncreated by github.com/0xPolygon/polygon-edge/syncer.(*syncer).Start in goroutine 1\n\t/home/XXX/Code/blade/syncer/syncer.go:83 +0xc6\n\ngoroutine 247 [select]:\ngithub.com/0xPolygon/polygon-edge/syncer.(*syncer).Sync(0xc000bb6300, 0xc000b02040)\n\t/home/XXX/Code/blade/syncer/syncer.go:184 +0x194\ngithub.com/0xPolygon/polygon-edge/consensus/polybft.(*Polybft).Start.func1({0x42a165?, 0x43a56a0?})\n\t/home/XXX/Code/blade/consensus/polybft/polybft.go:549 +0x67\ngithub.com/0xPolygon/polygon-edge/helper/common.RetryForever.func1({0x2000005?, 0xc000e9b698?})\n\t/home/XXX/Code/blade/helper/common/common.go:40 +0x23\ngithub.com/sethvargo/go-retry.Do.func1({0x2e69b90?, 0x43a1fa0?})\n\t/home/XXX/go/pkg/mod/github.com/sethvargo/go-retry@v0.3.0/retry.go:101 +0x22\ngithub.com/sethvargo/go-retry.DoValue[...]({0x2e69b90?, 0x43a1fa0}, {0x2e4c320, 0xc000c86000}, 0xc00a94bf38)\n\t/home/XXX/go/pkg/mod/github.com/sethvargo/go-retry@v0.3.0/retry.go:63 +0x8a\ngithub.com/sethvargo/go-retry.Do({0x2e69b90?, 0x43a1fa0?}, {0x2e4c320?, 0xc000c86000?}, 0xe41d92a581b449a?)\n\t/home/XXX/go/pkg/mod/github.com/sethvargo/go-retry@v0.3.0/retry.go:100 +0x56\ngithub.com/0xPolygon/polygon-edge/helper/common.RetryForever({0x2e69b90, 0x43a1fa0}, 0x3b9aca00, 0xc000cafbc0)\n\t/home/XXX/Code/blade/helper/common/common.go:38 +0xa9\ncreated by github.com/0xPolygon/polygon-edge/consensus/polybft.(*Polybft).Start in goroutine 1\n\t/home/XXX/Code/blade/consensus/polybft/polybft.go:543 +0x1df\n\ngoroutine 248 [select]:\ngithub.com/0xPolygon/polygon-edge/consensus/polybft.(*Polybft).startConsensusProtocol(0xc0006621e0)\n\t/home/XXX/Code/blade/consensus/polybft/polybft.go:671 +0x677\ncreated by github.com/0xPolygon/polygon-edge/consensus/polybft.(*Polybft).startRuntime in goroutine 1\n\t/home/XXX/Code/blade/consensus/polybft/polybft.go:601 +0x4f\n\ngoroutine 249 [chan receive]:\ngithub.com/0xPolygon/polygon-edge/consensus/polybft.(*State).startStatsReleasing(0xc000c5c940)\n\t/home/XXX/Code/blade/consensus/polybft/stats.go:31 +0x16b\ncreated by github.com/0xPolygon/polygon-edge/consensus/polybft.(*Polybft).Start in goroutine 1\n\t/home/XXX/Code/blade/consensus/polybft/polybft.go:564 +0x278\n\ngoroutine 214 [chan receive, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network.(*Topic).readLoop.func1()\n\t/home/XXX/Code/blade/network/gossip.go:92 +0x26\ncreated by github.com/0xPolygon/polygon-edge/network.(*Topic).readLoop in goroutine 243\n\t/home/XXX/Code/blade/network/gossip.go:91 +0xfd\n\ngoroutine 26041 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00ac9f120, {0x2e69c00, 0xc00d2f8050})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 26040\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 196 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network.(*Server).Subscribe.func1()\n\t/home/XXX/Code/blade/network/server.go:674 +0x165\ncreated by github.com/0xPolygon/polygon-edge/network.(*Server).Subscribe in goroutine 242\n\t/home/XXX/Code/blade/network/server.go:670 +0x10c\n\ngoroutine 197 [chan receive, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/network.(*Server).SubscribeCh.func3()\n\t/home/XXX/Code/blade/network/server.go:717 +0x26\ncreated by github.com/0xPolygon/polygon-edge/network.(*Server).SubscribeCh in goroutine 242\n\t/home/XXX/Code/blade/network/server.go:716 +0x169\n\ngoroutine 250 [select, 449 minutes]:\ngithub.com/0xPolygon/polygon-edge/txpool.(*TxPool).gossipBatcher(0xc000a60000)\n\t/home/XXX/Code/blade/txpool/txpool.go:292 +0x170\ncreated by github.com/0xPolygon/polygon-edge/txpool.(*TxPool).startGossipBatchers in goroutine 1\n\t/home/XXX/Code/blade/txpool/txpool.go:269 +0x25\n\ngoroutine 251 [select, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/txpool.(*TxPool).Start.func1()\n\t/home/XXX/Code/blade/txpool/txpool.go:345 +0xa5\ncreated by github.com/0xPolygon/polygon-edge/txpool.(*TxPool).Start in goroutine 1\n\t/home/XXX/Code/blade/txpool/txpool.go:343 +0xb9\n\ngoroutine 252 [select, 449 minutes]:\ngithub.com/0xPolygon/polygon-edge/txpool.(*TxPool).Start.func2()\n\t/home/XXX/Code/blade/txpool/txpool.go:361 +0x96\ncreated by github.com/0xPolygon/polygon-edge/txpool.(*TxPool).Start in goroutine 1\n\t/home/XXX/Code/blade/txpool/txpool.go:359 +0xf6\n\ngoroutine 230 [chan receive, 487 minutes]:\ngithub.com/0xPolygon/polygon-edge/consensus/polybft.(*State).startStatsReleasing.func1()\n\t/home/XXX/Code/blade/consensus/polybft/stats.go:27 +0x26\ncreated by github.com/0xPolygon/polygon-edge/consensus/polybft.(*State).startStatsReleasing in goroutine 249\n\t/home/XXX/Code/blade/consensus/polybft/stats.go:26 +0xe9\n\ngoroutine 231 [syscall, 487 minutes]:\nos/signal.signal_recv()\n\t/home/XXX/sdk/go1.22.10/src/runtime/sigqueue.go:152 +0x29\nos/signal.loop()\n\t/home/XXX/sdk/go1.22.10/src/os/signal/signal_unix.go:23 +0x13\ncreated by os/signal.Notify.func1.1 in goroutine 1\n\t/home/XXX/sdk/go1.22.10/src/os/signal/signal.go:151 +0x1f\n\ngoroutine 198 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc0008a9370, {0x2e69c00, 0xc000634fa0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 266\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 254 [IO wait]:\ninternal/poll.runtime_pollWait(0x7646a555a2e8, 0x72)\n\t/home/XXX/sdk/go1.22.10/src/runtime/netpoll.go:345 +0x85\ninternal/poll.(*pollDesc).wait(0xc000328380?, 0xc000d33000?, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:84 +0x27\ninternal/poll.(*pollDesc).waitRead(...)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:89\ninternal/poll.(*FD).Read(0xc000328380, {0xc000d33000, 0x1000, 0x1000})\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_unix.go:164 +0x27a\nnet.(*netFD).Read(0xc000328380, {0xc000d33000?, 0xc000a26fb8?, 0xc0011b2100?})\n\t/home/XXX/sdk/go1.22.10/src/net/fd_posix.go:55 +0x25\nnet.(*conn).Read(0xc00100e068, {0xc000d33000?, 0xc000d27ca0?, 0xc000d27d50?})\n\t/home/XXX/sdk/go1.22.10/src/net/net.go:185 +0x45\nbufio.(*Reader).Read(0xc000d48000, {0xc000d341b0, 0x2, 0xc000d27d80?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc000d48000}, {0xc000d341b0, 0x2, 0x2}, 0x2)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-libp2p/p2p/security/noise.(*secureSession).readNextInsecureMsgLen(0xc000d34120)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/security/noise/rw.go:133 +0x35\ngithub.com/libp2p/go-libp2p/p2p/security/noise.(*secureSession).Read(0xc000d34120, {0xc000c1d650, 0xc, 0xc})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/security/noise/rw.go:52 +0x1fc\nio.ReadAtLeast({0x7646a40001c8, 0xc000d34120}, {0xc000c1d650, 0xc, 0xc}, 0xc)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-yamux/v4.(*Session).recvLoop(0xc000a64700)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:668 +0xf3\ngithub.com/libp2p/go-yamux/v4.(*Session).recv(0xc000a64700)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:640 +0x18\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 283\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:162 +0x516\n\ngoroutine 267 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Session).AcceptStream(0xc001028100)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:272 +0x106\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*conn).AcceptStream(0x44d540?)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/conn.go:43 +0x13\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:118 +0xa2\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start in goroutine 264\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:114 +0x4f\n\ngoroutine 307 [sync.Cond.Wait]:\nsync.runtime_notifyListWait(0xc000d36250, 0x1d7c2)\n\t/home/XXX/sdk/go1.22.10/src/runtime/sema.go:569 +0x159\nsync.(*Cond).Wait(0x2e69b90?)\n\t/home/XXX/sdk/go1.22.10/src/sync/cond.go:70 +0x85\ngithub.com/libp2p/go-libp2p-pubsub.(*rpcQueue).Pop(0xc000d36240, {0x2e69b90, 0x43a1fa0})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/rpc_queue.go:129 +0x1da\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handleSendingMessages(0xc000a64700?, {0x2e69b90, 0x43a1fa0}, {0x2e82710, 0xc000ae4380}, 0xc000d36240)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:178 +0x111\ncreated by github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer in goroutine 292\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:130 +0x2e5\n\ngoroutine 222 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00082e990, {0x2e69c00, 0xc000c8c320})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 221\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 219 [sync.Cond.Wait]:\nsync.runtime_notifyListWait(0xc000d8c190, 0x1d595)\n\t/home/XXX/sdk/go1.22.10/src/runtime/sema.go:569 +0x159\nsync.(*Cond).Wait(0x2e69b90?)\n\t/home/XXX/sdk/go1.22.10/src/sync/cond.go:70 +0x85\ngithub.com/libp2p/go-libp2p-pubsub.(*rpcQueue).Pop(0xc000d8c180, {0x2e69b90, 0x43a1fa0})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/rpc_queue.go:129 +0x1da\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handleSendingMessages(0xc00084a908?, {0x2e69b90, 0x43a1fa0}, {0x2e82710, 0xc00084c000}, 0xc000d8c180)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:178 +0x111\ncreated by github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer in goroutine 218\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:130 +0x2e5\n\ngoroutine 220 [select, 487 minutes]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc000da40e0, {0xc0009fbb27, 0x1, 0x1})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x4886bc?, {0xc0009fbb27?, 0x833b02870f306f?, 0xc000284234?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc000c3ad80, {0xc0009fbb27?, 0xa11d134184854dca?, 0x487385d609d5f3c3?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\ngithub.com/multiformats/go-multistream.(*lazyClientConn[...]).Read(0xc0000ad808?, {0xc0009fbb27?, 0x1?, 0x1?})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-multistream@v0.6.0/lazyClient.go:68 +0xad\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*streamWrapper).Read(0x102bd18678a5603b?, {0xc0009fbb27?, 0x1adf4f35b9141ecb?, 0x2e8cc6ecb823b5ff?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:1138 +0x22\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handlePeerDead(0xc00084a908, {0x2e82710, 0xc00084c000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:150 +0x73\ncreated by github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer in goroutine 218\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:131 +0x345\n\ngoroutine 170 [select, 487 minutes]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc000e201c0, {0xc000c869c0, 0x1, 0x1})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x4886bc?, {0xc000c869c0?, 0x100c000d0ee90?, 0xc001001e60?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc000077580, {0xc000c869c0?, 0x100c000d0ef08?, 0x40cd28?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\ngithub.com/multiformats/go-multistream.(*lazyClientConn[...]).Read(0xc000312808?, {0xc000c869c0?, 0x1?, 0x1?})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-multistream@v0.6.0/lazyClient.go:68 +0xad\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*streamWrapper).Read(0xc000d0efb0?, {0xc000c869c0?, 0xc0009f3590?, 0x692b6ef0600ed6d4?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:1138 +0x22\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handlePeerDead(0xc00084a908, {0x2e82710, 0xc000a1c200})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:150 +0x73\ncreated by github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer in goroutine 234\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:131 +0x345\n\ngoroutine 25993 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00d35d090, {0x2e69c00, 0xc00a3479a0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 25991\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 26026 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc0011f1c10, {0x2e69c00, 0xc009e0f040})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 26025\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 284 [IO wait]:\ninternal/poll.runtime_pollWait(0x7646a555a5d0, 0x72)\n\t/home/XXX/sdk/go1.22.10/src/runtime/netpoll.go:345 +0x85\ninternal/poll.(*pollDesc).wait(0xc000328200?, 0xc00106c000?, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:84 +0x27\ninternal/poll.(*pollDesc).waitRead(...)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:89\ninternal/poll.(*FD).Read(0xc000328200, {0xc00106c000, 0x1000, 0x1000})\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_unix.go:164 +0x27a\nnet.(*netFD).Read(0xc000328200, {0xc00106c000?, 0xc0009d15b8?, 0xc000b1c000?})\n\t/home/XXX/sdk/go1.22.10/src/net/fd_posix.go:55 +0x25\nnet.(*conn).Read(0xc00100e060, {0xc00106c000?, 0xc00dab6ca0?, 0xc00dab6d50?})\n\t/home/XXX/sdk/go1.22.10/src/net/net.go:185 +0x45\nbufio.(*Reader).Read(0xc0010015c0, {0xc000b8ae10, 0x2, 0xc00dab6d80?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc0010015c0}, {0xc000b8ae10, 0x2, 0x2}, 0x2)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-libp2p/p2p/security/noise.(*secureSession).readNextInsecureMsgLen(0xc000b8ad80)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/security/noise/rw.go:133 +0x35\ngithub.com/libp2p/go-libp2p/p2p/security/noise.(*secureSession).Read(0xc000b8ad80, {0xc0005b5320, 0xc, 0xc})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/security/noise/rw.go:52 +0x1fc\nio.ReadAtLeast({0x7646a40001c8, 0xc000b8ad80}, {0xc0005b5320, 0xc, 0xc}, 0xc)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-yamux/v4.(*Session).recvLoop(0xc001028200)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:668 +0xf3\ngithub.com/libp2p/go-yamux/v4.(*Session).recv(0xc001028200)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:640 +0x18\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 280\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:162 +0x516\n\ngoroutine 285 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Session).sendLoop(0xc001028200)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:583 +0x5a7\ngithub.com/libp2p/go-yamux/v4.(*Session).send(0xc001028200)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:512 +0x18\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 280\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:163 +0x556\n\ngoroutine 286 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Session).startMeasureRTT(0xc001028200)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:356 +0xca\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 280\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:164 +0x596\n\ngoroutine 338 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc000945b20, {0x2e69c00, 0xc0009f8050})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 289\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 586 [select, 7 minutes]:\ngoogle.golang.org/grpc/internal/transport.(*http2Server).keepalive(0xc000194d00)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:1180 +0x205\ncreated by google.golang.org/grpc/internal/transport.NewServerTransport in goroutine 584\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:356 +0x1945\n\ngoroutine 306 [select, 400 minutes]:\ngithub.com/libp2p/go-yamux/v4.(*Session).AcceptStream(0xc001028200)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:272 +0x106\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*conn).AcceptStream(0x44d540?)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/conn.go:43 +0x13\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:118 +0xa2\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start in goroutine 287\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:114 +0x4f\n\ngoroutine 255 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Session).sendLoop(0xc000a64700)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:583 +0x5a7\ngithub.com/libp2p/go-yamux/v4.(*Session).send(0xc000a64700)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:512 +0x18\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 283\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:163 +0x556\n\ngoroutine 256 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Session).startMeasureRTT(0xc000a64700)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:356 +0xca\ncreated by github.com/libp2p/go-yamux/v4.newSession in goroutine 283\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:164 +0x596\n\ngoroutine 308 [select, 487 minutes]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc000d1e2a0, {0xc000b9cf78, 0x1, 0x1})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x4886bc?, {0xc000b9cf78?, 0xc0001da768?, 0xc0003170a4?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc000b9b080, {0xc000b9cf78?, 0x0?, 0xc0009d0fc0?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\ngithub.com/multiformats/go-multistream.(*lazyClientConn[...]).Read(0xc000313008?, {0xc000b9cf78?, 0x1?, 0x1?})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-multistream@v0.6.0/lazyClient.go:68 +0xad\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*streamWrapper).Read(0xc0010137d0?, {0xc000b9cf78?, 0x0?, 0x0?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:1138 +0x22\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handlePeerDead(0xc00084a908, {0x2e82710, 0xc000ae4380})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:150 +0x73\ncreated by github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer in goroutine 292\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:131 +0x345\n\ngoroutine 363 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc0006cb6a0, {0x2e69c00, 0xc000a4e230})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 354\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 356 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Session).AcceptStream(0xc000a64700)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/session.go:272 +0x106\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*conn).AcceptStream(0x44d540?)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/conn.go:43 +0x13\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:118 +0xa2\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start in goroutine 257\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:114 +0x4f\n\ngoroutine 169 [sync.Cond.Wait]:\nsync.runtime_notifyListWait(0xc000e26190, 0x1dc71)\n\t/home/XXX/sdk/go1.22.10/src/runtime/sema.go:569 +0x159\nsync.(*Cond).Wait(0x2e69b90?)\n\t/home/XXX/sdk/go1.22.10/src/sync/cond.go:70 +0x85\ngithub.com/libp2p/go-libp2p-pubsub.(*rpcQueue).Pop(0xc000e26180, {0x2e69b90, 0x43a1fa0})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/rpc_queue.go:129 +0x1da\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handleSendingMessages(0xdd67c0?, {0x2e69b90, 0x43a1fa0}, {0x2e82710, 0xc000a1c200}, 0xc000e26180)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:178 +0x111\ncreated by github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer in goroutine 234\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:130 +0x2e5\n\ngoroutine 233 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc000e200e0, {0xc001004c28, 0x1, 0x1})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0xc00cd18500?, {0xc001004c28?, 0xae20ab?, 0x0?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc000077300, {0xc001004c28?, 0x474c79?, 0xc001063a28?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\nio.ReadAtLeast({0x7646a40400b8, 0xc000077300}, {0xc001004c28, 0x1, 0x1}, 0x1)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngithub.com/libp2p/go-msgio.(*simpleByteReader).ReadByte(0xc001004c18)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:185 +0x31\ngithub.com/multiformats/go-varint.ReadUvarint({0x2e4d620, 0xc001004c18})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-varint@v0.0.7/varint.go:80 +0x51\ngithub.com/libp2p/go-msgio.(*varintReader).nextMsgLen(0xc00082b500)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:119 +0x2a\ngithub.com/libp2p/go-msgio.(*varintReader).ReadMsg(0xc00082b500)\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:149 +0xbb\ngithub.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewStream(0xc00084a908, {0x2e82680, 0xc000077300})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.12.0/comm.go:66 +0x3d7\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).SetStreamHandler.func1({0x40f24b0?, 0x222e120?}, {0x7646a4040068?, 0xc000077300?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:659 +0x82\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler(0xc000bfdc20, {0x2e82680, 0xc000077300})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:487 +0x7e9\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1.1()\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:142 +0xa7\ncreated by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1 in goroutine 356\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_conn.go:128 +0x1b7\n\ngoroutine 475 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc000fc01c0, {0xc009b18000, 0x8000, 0x8000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x40d6b40?, {0xc009b18000?, 0xc000ea39a0?, 0xc0006d4b40?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc009a91580, {0xc009b18000?, 0xb?, 0x415dac0?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\ngithub.com/multiformats/go-multistream.(*lazyClientConn[...]).Read(0x0?, {0xc009b18000?, 0x0?, 0xc0012a5da8?})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-multistream@v0.6.0/lazyClient.go:68 +0xad\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*streamWrapper).Read(0x0?, {0xc009b18000?, 0x800010601?, 0xc000000000?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:1138 +0x22\nbufio.(*Reader).Read(0xc00109ee40, {0xc0001daf20, 0x9, 0xc0000ad808?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc00109ee40}, {0xc0001daf20, 0x9, 0x9}, 0x9)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngolang.org/x/net/http2.readFrameHeader({0xc0001daf20, 0x9, 0xc00d4345b8?}, {0x2e4c240?, 0xc00109ee40?})\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:237 +0x65\ngolang.org/x/net/http2.(*Framer).ReadFrame(0xc0001daee0)\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:501 +0x85\ngoogle.golang.org/grpc/internal/transport.(*http2Client).reader(0xc000cecfc8, 0xc00109eea0)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:1639 +0x1f0\ncreated by google.golang.org/grpc/internal/transport.NewHTTP2Client in goroutine 507\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:413 +0x1ed9\n\ngoroutine 369 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc000ed0140, {0x2e69c00, 0xc000a4e640})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 368\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 296 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00090d340, {0x2e69c00, 0xc000517a40})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 295\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 464 [select]:\ngoogle.golang.org/grpc/internal/transport.(*controlBuffer).get(0xc000ae1280, 0x1)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:412 +0x108\ngoogle.golang.org/grpc/internal/transport.(*loopyWriter).run(0xc0011d4680)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:575 +0x86\ngoogle.golang.org/grpc/internal/transport.NewServerTransport.func2()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:335 +0xde\ncreated by google.golang.org/grpc/internal/transport.NewServerTransport in goroutine 463\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:333 +0x18fe\n\ngoroutine 465 [select, 7 minutes]:\ngoogle.golang.org/grpc/internal/transport.(*http2Server).keepalive(0xc0007c4ea0)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:1180 +0x205\ncreated by google.golang.org/grpc/internal/transport.NewServerTransport in goroutine 463\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:356 +0x1945\n\ngoroutine 25992 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00d35d060, {0x2e69c00, 0xc00a347950})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 25991\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 238 [select]:\ngithub.com/0xPolygon/polygon-edge/consensus/polybft.(*Polybft).startConsensusProtocol.func1()\n\t/home/XXX/Code/blade/consensus/polybft/polybft.go:624 +0x96\ncreated by github.com/0xPolygon/polygon-edge/consensus/polybft.(*Polybft).startConsensusProtocol in goroutine 248\n\t/home/XXX/Code/blade/consensus/polybft/polybft.go:620 +0x1a5\n\ngoroutine 505 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc000fef5d0, {0x2e69c00, 0xc000fede00})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 503\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 587 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc0001db180, {0xc009d00000, 0x8000, 0x8000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x413a0f?, {0xc009d00000?, 0x41cec5?, 0xc001062b98?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc009b28500, {0xc009d00000?, 0x18?, 0x7646ec34c878?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\nbufio.(*Reader).Read(0xc009bf2240, {0xc000d1eba0, 0x9, 0xd6de0d?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc009bf2240}, {0xc000d1eba0, 0x9, 0x9}, 0x9)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngolang.org/x/net/http2.readFrameHeader({0xc000d1eba0, 0x9, 0xc00cc2de70?}, {0x2e4c240?, 0xc009bf2240?})\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:237 +0x65\ngolang.org/x/net/http2.(*Framer).ReadFrame(0xc000d1eb60)\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:501 +0x85\ngoogle.golang.org/grpc/internal/transport.(*http2Server).HandleStreams(0xc000194d00, {0x2e69bc8, 0xc00122fce0}, 0xc00122fd10)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:640 +0x10d\ngoogle.golang.org/grpc.(*Server).serveStreams(0xc000b3f000, {0x2e69b90?, 0x43a1fa0?}, {0x2e6a568, 0xc000194d00}, {0x2e77490?, 0xc009bf4110?})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:1024 +0x3b6\ngoogle.golang.org/grpc.(*Server).handleRawConn.func1()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:959 +0x56\ncreated by google.golang.org/grpc.(*Server).handleRawConn in goroutine 584\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:958 +0x1c6\n\ngoroutine 466 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc0010a39d0, {0x2e69c00, 0xc0098d8410})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 351\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 504 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc000fef5a0, {0x2e69c00, 0xc000feddb0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 503\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 460 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc000da48c0, {0xc0011f4000, 0x8000, 0x8000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x10?, {0xc0011f4000?, 0x41cec5?, 0xc00129eb98?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc0011d4380, {0xc0011f4000?, 0x18?, 0x7646ec34cd28?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\nbufio.(*Reader).Read(0xc000dbb7a0, {0xc000da49e0, 0x9, 0xd6de0d?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc000dbb7a0}, {0xc000da49e0, 0x9, 0x9}, 0x9)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngolang.org/x/net/http2.readFrameHeader({0xc000da49e0, 0x9, 0xc00c8d9430?}, {0x2e4c240?, 0xc000dbb7a0?})\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:237 +0x65\ngolang.org/x/net/http2.(*Framer).ReadFrame(0xc000da49a0)\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:501 +0x85\ngoogle.golang.org/grpc/internal/transport.(*http2Server).HandleStreams(0xc0007c4820, {0x2e69bc8, 0xc0011d2f00}, 0xc0011d2f30)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:640 +0x10d\ngoogle.golang.org/grpc.(*Server).serveStreams(0xc000b3f000, {0x2e69b90?, 0x43a1fa0?}, {0x2e6a568, 0xc0007c4820}, {0x2e77490?, 0xc0011f01e0?})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:1024 +0x3b6\ngoogle.golang.org/grpc.(*Server).handleRawConn.func1()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:959 +0x56\ncreated by google.golang.org/grpc.(*Server).handleRawConn in goroutine 457\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:958 +0x1c6\n\ngoroutine 352 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc0010a3970, {0x2e69c00, 0xc0098d8370})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 351\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 459 [select, 7 minutes]:\ngoogle.golang.org/grpc/internal/transport.(*http2Server).keepalive(0xc0007c4820)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:1180 +0x205\ncreated by google.golang.org/grpc/internal/transport.NewServerTransport in goroutine 457\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:356 +0x1945\n\ngoroutine 353 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc0010a39a0, {0x2e69c00, 0xc0098d83c0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 351\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 442 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc0001dae00, {0xc009910000, 0x8000, 0x8000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x40d6b40?, {0xc009910000?, 0xc0011ea320?, 0xc00a07b220?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc0010c3580, {0xc009910000?, 0xb?, 0x415dac0?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\ngithub.com/multiformats/go-multistream.(*lazyClientConn[...]).Read(0x0?, {0xc009910000?, 0x0?, 0xc001208da8?})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-multistream@v0.6.0/lazyClient.go:68 +0xad\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*streamWrapper).Read(0x0?, {0xc009910000?, 0x800010601?, 0xc000000000?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:1138 +0x22\nbufio.(*Reader).Read(0xc000ea1e60, {0xc00057cf20, 0x9, 0xc000312808?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc000ea1e60}, {0xc00057cf20, 0x9, 0x9}, 0x9)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngolang.org/x/net/http2.readFrameHeader({0xc00057cf20, 0x9, 0xc00d7a7c80?}, {0x2e4c240?, 0xc000ea1e60?})\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:237 +0x65\ngolang.org/x/net/http2.(*Framer).ReadFrame(0xc00057cee0)\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:501 +0x85\ngoogle.golang.org/grpc/internal/transport.(*http2Client).reader(0xc000ee4b48, 0xc000ea1ec0)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:1639 +0x1f0\ncreated by google.golang.org/grpc/internal/transport.NewHTTP2Client in goroutine 467\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:413 +0x1ed9\n\ngoroutine 458 [select]:\ngoogle.golang.org/grpc/internal/transport.(*controlBuffer).get(0xc000ae0f80, 0x1)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:412 +0x108\ngoogle.golang.org/grpc/internal/transport.(*loopyWriter).run(0xc0011d4480)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:575 +0x86\ngoogle.golang.org/grpc/internal/transport.NewServerTransport.func2()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:335 +0xde\ncreated by google.golang.org/grpc/internal/transport.NewServerTransport in goroutine 457\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:333 +0x18fe\n\ngoroutine 444 [select]:\ngoogle.golang.org/grpc/internal/transport.(*controlBuffer).get(0xc000b46400, 0x1)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:412 +0x108\ngoogle.golang.org/grpc/internal/transport.(*loopyWriter).run(0xc0011d4200)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:575 +0x86\ngoogle.golang.org/grpc/internal/transport.NewHTTP2Client.func6()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:471 +0xd2\ncreated by google.golang.org/grpc/internal/transport.NewHTTP2Client in goroutine 467\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:469 +0x24de\n\ngoroutine 482 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc000da4a80, {0xc009988000, 0x8000, 0x8000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x10?, {0xc009988000?, 0x41cec5?, 0xc0012a0b98?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc0011d4580, {0xc009988000?, 0x18?, 0x7646ec34ba68?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\nbufio.(*Reader).Read(0xc000dbb920, {0xc000da4ba0, 0x9, 0xd6de0d?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc000dbb920}, {0xc000da4ba0, 0x9, 0x9}, 0x9)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngolang.org/x/net/http2.readFrameHeader({0xc000da4ba0, 0x9, 0xc00ca215a0?}, {0x2e4c240?, 0xc000dbb920?})\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:237 +0x65\ngolang.org/x/net/http2.(*Framer).ReadFrame(0xc000da4b60)\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:501 +0x85\ngoogle.golang.org/grpc/internal/transport.(*http2Server).HandleStreams(0xc0007c4ea0, {0x2e69bc8, 0xc0011d35f0}, 0xc0011d3620)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:640 +0x10d\ngoogle.golang.org/grpc.(*Server).serveStreams(0xc000b3f000, {0x2e69b90?, 0x43a1fa0?}, {0x2e6a568, 0xc0007c4ea0}, {0x2e77490?, 0xc0011f0520?})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:1024 +0x3b6\ngoogle.golang.org/grpc.(*Server).handleRawConn.func1()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:959 +0x56\ncreated by google.golang.org/grpc.(*Server).handleRawConn in goroutine 463\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/server.go:958 +0x1c6\n\ngoroutine 506 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc000fef600, {0x2e69c00, 0xc000fede50})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 503\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 585 [select]:\ngoogle.golang.org/grpc/internal/transport.(*controlBuffer).get(0xc00123f640, 0x1)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:412 +0x108\ngoogle.golang.org/grpc/internal/transport.(*loopyWriter).run(0xc001248880)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:575 +0x86\ngoogle.golang.org/grpc/internal/transport.NewServerTransport.func2()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:335 +0xde\ncreated by google.golang.org/grpc/internal/transport.NewServerTransport in goroutine 584\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_server.go:333 +0x18fe\n\ngoroutine 432 [select]:\ngoogle.golang.org/grpc/internal/transport.(*controlBuffer).get(0xc009b00780, 0x1)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:412 +0x108\ngoogle.golang.org/grpc/internal/transport.(*loopyWriter).run(0xc009926480)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:575 +0x86\ngoogle.golang.org/grpc/internal/transport.NewHTTP2Client.func6()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:471 +0xd2\ncreated by google.golang.org/grpc/internal/transport.NewHTTP2Client in goroutine 507\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:469 +0x24de\n\ngoroutine 25984 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00dbd9bf0, {0x2e69c00, 0xc000ab2550})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 25982\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 687 [select]:\ngoogle.golang.org/grpc/internal/transport.(*controlBuffer).get(0xc00a2d1100, 0x1)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:412 +0x108\ngoogle.golang.org/grpc/internal/transport.(*loopyWriter).run(0xc00a0d3b80)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/controlbuf.go:575 +0x86\ngoogle.golang.org/grpc/internal/transport.NewHTTP2Client.func6()\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:471 +0xd2\ncreated by google.golang.org/grpc/internal/transport.NewHTTP2Client in goroutine 716\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:469 +0x24de\n\ngoroutine 26012 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00d6d04a0, {0x2e69c00, 0xc0008fc960})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 26011\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 713 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00a33d050, {0x2e69c00, 0xc00a682050})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 712\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 714 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00a33d080, {0x2e69c00, 0xc00a6820a0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 712\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 715 [select, 487 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00a33d0b0, {0x2e69c00, 0xc00a6820f0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 712\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 787 [select]:\ngithub.com/libp2p/go-yamux/v4.(*Stream).Read(0xc001042ee0, {0xc00a60a000, 0x8000, 0x8000})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.1/stream.go:111 +0x1a5\ngithub.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x40d6b40?, {0xc00a60a000?, 0xc009a53040?, 0xc00cb1fbd0?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/muxer/yamux/stream.go:17 +0x18\ngithub.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc00a37a700, {0xc00a60a000?, 0xb?, 0x415dac0?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/net/swarm/swarm_stream.go:58 +0x2d\ngithub.com/multiformats/go-multistream.(*lazyClientConn[...]).Read(0x0?, {0xc00a60a000?, 0x0?, 0xc000ec6da8?})\n\t/home/XXX/go/pkg/mod/github.com/multiformats/go-multistream@v0.6.0/lazyClient.go:68 +0xad\ngithub.com/libp2p/go-libp2p/p2p/host/basic.(*streamWrapper).Read(0x0?, {0xc00a60a000?, 0x800010601?, 0xc000000000?})\n\t/home/XXX/go/pkg/mod/github.com/libp2p/go-libp2p@v0.38.1/p2p/host/basic/basic_host.go:1138 +0x22\nbufio.(*Reader).Read(0xc00a608000, {0xc000d1f2a0, 0x9, 0xc000312808?})\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:241 +0x197\nio.ReadAtLeast({0x2e4c240, 0xc00a608000}, {0xc000d1f2a0, 0x9, 0x9}, 0x9)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:335 +0x90\nio.ReadFull(...)\n\t/home/XXX/sdk/go1.22.10/src/io/io.go:354\ngolang.org/x/net/http2.readFrameHeader({0xc000d1f2a0, 0x9, 0xc00db0e8e8?}, {0x2e4c240?, 0xc00a608000?})\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:237 +0x65\ngolang.org/x/net/http2.(*Framer).ReadFrame(0xc000d1f260)\n\t/home/XXX/go/pkg/mod/golang.org/x/net@v0.33.0/http2/frame.go:501 +0x85\ngoogle.golang.org/grpc/internal/transport.(*http2Client).reader(0xc00a2fe248, 0xc00a608060)\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:1639 +0x1f0\ncreated by google.golang.org/grpc/internal/transport.NewHTTP2Client in goroutine 716\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/transport/http2_client.go:413 +0x1ed9\n\ngoroutine 26013 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00d6d04d0, {0x2e69c00, 0xc0008fc9b0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 26011\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 410416 [select]:\ngithub.com/0xPolygon/go-ibft/messages.(*eventSubscription).runLoop(0xc00d078d40)\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/messages/event_subscription.go:31 +0xc5\ncreated by github.com/0xPolygon/go-ibft/messages.(*eventManager).subscribe in goroutine 410415\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/messages/event_manager.go:75 +0x21f\n\ngoroutine 410413 [select]:\ngithub.com/0xPolygon/go-ibft/core.(*IBFT).watchForFutureProposal(0xc000c847e0, {0x2e69c00, 0xc0006d4aa0})\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:235 +0x165\ncreated by github.com/0xPolygon/go-ibft/core.(*IBFT).RunSequence in goroutine 410411\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:341 +0x69e\n\ngoroutine 26035 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00ac9ecd0, {0x2e69c00, 0xc0007fb9a0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 26034\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 410415 [select]:\ngithub.com/0xPolygon/go-ibft/core.(*IBFT).runNewRound(0xc000c847e0, {0x2e69c00, 0xc0006d4aa0})\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:601 +0x1b2\ngithub.com/0xPolygon/go-ibft/core.(*IBFT).runStates(0xc000c847e0, {0x2e69c00, 0xc0006d4aa0})\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:560 +0x85\ngithub.com/0xPolygon/go-ibft/core.(*IBFT).startRound(0xc000c847e0, {0x2e69c00, 0xc0006d4aa0})\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:428 +0x1da\ncreated by github.com/0xPolygon/go-ibft/core.(*IBFT).RunSequence in goroutine 410411\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:347 +0x768\n\ngoroutine 25983 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00dbd9bc0, {0x2e69c00, 0xc000ab24b0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 25982\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 410430 [IO wait]:\ninternal/poll.runtime_pollWait(0x7646a5559d18, 0x72)\n\t/home/XXX/sdk/go1.22.10/src/runtime/netpoll.go:345 +0x85\ninternal/poll.(*pollDesc).wait(0xc0010c2a80?, 0xc00e205cf1?, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:84 +0x27\ninternal/poll.(*pollDesc).waitRead(...)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:89\ninternal/poll.(*FD).Read(0xc0010c2a80, {0xc00e205cf1, 0x1, 0x1})\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_unix.go:164 +0x27a\nnet.(*netFD).Read(0xc0010c2a80, {0xc00e205cf1?, 0xc000000002?, 0x232c59f?})\n\t/home/XXX/sdk/go1.22.10/src/net/fd_posix.go:55 +0x25\nnet.(*conn).Read(0xc00dd201f8, {0xc00e205cf1?, 0x232c59f?, 0xc001016fd0?})\n\t/home/XXX/sdk/go1.22.10/src/net/net.go:185 +0x45\nnet/http.(*connReader).backgroundRead(0xc00e205ce0)\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:681 +0x37\ncreated by net/http.(*connReader).startBackgroundRead in goroutine 410429\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:677 +0xba\n\ngoroutine 26036 [select, 457 minutes]:\ngoogle.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run(0xc00ac9ed00, {0x2e69c00, 0xc0007fb9f0})\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:88 +0x115\ncreated by google.golang.org/grpc/internal/grpcsync.NewCallbackSerializer in goroutine 26034\n\t/home/XXX/go/pkg/mod/google.golang.org/grpc@v1.69.2/internal/grpcsync/callback_serializer.go:52 +0x11a\n\ngoroutine 410470 [select]:\ngithub.com/0xPolygon/go-ibft/messages.(*eventSubscription).runLoop(0xc000efe140)\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/messages/event_subscription.go:31 +0xc5\ncreated by github.com/0xPolygon/go-ibft/messages.(*eventManager).subscribe in goroutine 410413\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/messages/event_manager.go:75 +0x21f\n\ngoroutine 410471 [select]:\ngithub.com/0xPolygon/go-ibft/messages.(*eventSubscription).runLoop(0xc000efe180)\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/messages/event_subscription.go:31 +0xc5\ncreated by github.com/0xPolygon/go-ibft/messages.(*eventManager).subscribe in goroutine 410414\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/messages/event_manager.go:75 +0x21f\n\ngoroutine 410411 [select]:\ngithub.com/0xPolygon/go-ibft/core.(*IBFT).RunSequence(0xc000c847e0, {0x2e69c00, 0xc0006d4a50}, 0x3804)\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:354 +0x8a5\ngithub.com/0xPolygon/polygon-edge/consensus/polybft.(*IBFTConsensusWrapper).runSequence.func1()\n\t/home/XXX/Code/blade/consensus/polybft/ibft_consensus.go:33 +0x38\ncreated by github.com/0xPolygon/polygon-edge/consensus/polybft.(*IBFTConsensusWrapper).runSequence in goroutine 248\n\t/home/XXX/Code/blade/consensus/polybft/ibft_consensus.go:32 +0xdd\n\ngoroutine 386451 [IO wait]:\ninternal/poll.runtime_pollWait(0x7646a555a000, 0x72)\n\t/home/XXX/sdk/go1.22.10/src/runtime/netpoll.go:345 +0x85\ninternal/poll.(*pollDesc).wait(0xc00cdbfe00?, 0xc00ad26000?, 0x0)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:84 +0x27\ninternal/poll.(*pollDesc).waitRead(...)\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_poll_runtime.go:89\ninternal/poll.(*FD).Read(0xc00cdbfe00, {0xc00ad26000, 0x1000, 0x1000})\n\t/home/XXX/sdk/go1.22.10/src/internal/poll/fd_unix.go:164 +0x27a\nnet.(*netFD).Read(0xc00cdbfe00, {0xc00ad26000?, 0xc000ddfa98?, 0x4f3ac5?})\n\t/home/XXX/sdk/go1.22.10/src/net/fd_posix.go:55 +0x25\nnet.(*conn).Read(0xc000a7c758, {0xc00ad26000?, 0x0?, 0xc00e204008?})\n\t/home/XXX/sdk/go1.22.10/src/net/net.go:185 +0x45\nnet/http.(*connReader).Read(0xc00e204000, {0xc00ad26000, 0x1000, 0x1000})\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:789 +0x14b\nbufio.(*Reader).fill(0xc00bb0cba0)\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:110 +0x103\nbufio.(*Reader).Peek(0xc00bb0cba0, 0x4)\n\t/home/XXX/sdk/go1.22.10/src/bufio/bufio.go:148 +0x53\nnet/http.(*conn).serve(0xc000c84480, {0x2e69bc8, 0xc0008b86f0})\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:2079 +0x749\ncreated by net/http.(*Server).Serve in goroutine 192\n\t/home/XXX/sdk/go1.22.10/src/net/http/server.go:3290 +0x4b4\n\ngoroutine 410412 [select]:\ngithub.com/0xPolygon/go-ibft/core.(*IBFT).startRoundTimer(0xc000c847e0, {0x2e69c00, 0xc0006d4aa0}, 0x0)\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:155 +0x12d\ncreated by github.com/0xPolygon/go-ibft/core.(*IBFT).RunSequence in goroutine 410411\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:338 +0x63b\n\ngoroutine 410414 [select]:\ngithub.com/0xPolygon/go-ibft/core.(*IBFT).watchForRoundChangeCertificates(0xc000c847e0, {0x2e69c00, 0xc0006d4aa0})\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:279 +0x1b0\ncreated by github.com/0xPolygon/go-ibft/core.(*IBFT).RunSequence in goroutine 410411\n\t/home/XXX/go/pkg/mod/github.com/0x!polygon/go-ibft@v0.4.1-0.20240621090555-e81a63ff50d7/core/ibft.go:344 +0x705\n"
}
```

</details>



## debug\_startCPUProfile

The StartCPUProfile method enables CPU profiling and writes profiling data to the specified file. This method is commonly used for performance analysis in Go applications, allowing developers to capture detailed CPU usage information for analysis.

#### Parameters

* file: String - the file path where the CPU profile data will be written. The path can be relative or absolute. If a relative path is provided, the method will convert it to an absolute path.

#### Returns

* String - the method returns the absolute path of the file where the CPU profile data is being written. This can be used to verify the location of the profile output.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_startCPUProfile","params":["profile.txt"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/profile.txt"
}
```

</details>



## debug\_startGoTrace

The StartGoTrace method enables Go execution tracing and writes the trace data to a specified file.

#### Parameters

* file: String - the file path where the execution trace data should be written. The method supports relative and absolute file paths.

#### Returns

* String - the absolute file path to which the trace data is written. This allows the caller to confirm the final location of the trace file.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_startGoTrace","params":["gotrace.txt"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/gotrace.txt"
}
```

</details>



## debug\_stopCPUProfile

The StopCPUProfile method stops an active CPU profiling session.

#### Parameters

None

#### Returns

None

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_stopCPUProfile","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": null
}
```

</details>



## debug\_stopGoTrace

The StopGoTrace method stops an ongoing Go runtime trace.

#### Parameters

None

#### Returns

None

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_stopGoTrace","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": null
}
```

</details>



## debug\_storageRangeAt

The method StorageRangeAt is used to retrieve the storage at a specific block height and transaction index. It takes several parameters to specify the block, transaction, contract address, storage key range, and result limits.

#### Parameters

* blockHash: DATA, 32 Bytes - hash of a block.
* txIndex: QUANTITY - this represents the transaction index in the specified block.
* contractAddress: DATA, 20 Bytes - the address of the contract whose storage is being queried.
* keyStart: Array - this is the starting key for the range of storage keys to be retrieved.
* maxResult: QUANTITY - this represents the transaction index in the specified block.

#### Returns

* Object - the StorageRangeResult object:

  * Storage: Array - a map where the key is of type type.Hash and the value is of type storageEntry. This holds the actual storage entries for the contract at the specified block and transaction index. Fields of the storageEntry object:
    + Key: Array - this key represents the hash of the storage item ([]byte).
    + Value: DATA, 32 Bytes - this holds the actual storage entry for a particular key.
  * NextKey: Array - this field indicates the next key to be queried. If the storage contains the last key in the trie, NextKey will be nil, signaling that there are no further keys to query.

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_storageRangeAt","params":["0x76f64c40d6493cf00426be2eecdaf3f968768619bfb21ce6c57921be497ab3f7", 0, "0xdafea492d9c6733ae3d56b7ed1adb60692c98bc5","0x0000000000000000000000000000000000000000000000000000000000000000", 1],"id":1}'
````

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "storage": {
      "0x0000000000000000000000000000000000000000000000000000000000000001": "0x0000000000000000000000000000000000000000000000000000000000000001",
      "0x0000000000000000000000000000000000000000000000000000000000000002": "0x0000000000000000000000000000000000000000000000000000000000000002"
    },
    "nextKey": "0x0000000000000000000000000000000000000000000000000000000000000065"
  }
}
```

</details>



## debug_traceBlock

Executes all transactions in the block given from the first argument with a tracer and returns the tracing result.

#### Parameters

* Object - an object containing block data
* Object - the tracer options:

  + enableMemory: Boolean - (optional, default: false) the flag indicating enabling memory capture.
  + disableStack: Boolean - (optional, default: false) the flag indicating disabling stack capture.
  + disableStorage: Boolean - (optional, default: false) the flag indicating disabling storage capture.
  + enableReturnData: Boolean - (optional, default: false) the flag indicating enabling return data capture.
  + timeOut: String - (optional, default: "5s") the timeout for cancellation of execution.
  + tracer: String - (default: "structTracer") defines the debug tracer used for given call. Supported values: structTracer, callTracer.

#### Returns

  * Array - array of trace objects with the following fields:

  * failed: Boolean - the tx is successful or not
  * gas: QUANTITY - the total consumed gas in the tx
  * returnValue: DATA - the return value of the executed contract call
  * structLogs: Array - the trace result of each step with the following fields:

    + pc: QUANTITY - the current index in bytecode
    + op: String - the name of current executing operation
    + gas: QUANTITY - the available gas ßin the execution
    + gasCost: QUANTITY - the gas cost of the operation
    + depth: QUANTITY - the number of levels of calling functions
    + error: String - the error of the execution
    + stack: Array - array of values in the current stack
    + memory: Array - array of values in the current memory
    + storage: Object - mapping of the current storage
    + refund: QUANTITY - the total of current refund value

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"debug_traceBlock","params":[{}, {}],"id":1}'
````



## debug\_traceBlockByHash

Executes all transactions in the block specified by block hash with a tracer and returns the tracing result.

#### Parameters

* DATA, 32 Bytes - hash of a block.
* Object - the tracer options:

  + enableMemory: Boolean - (optional, default: false) the flag indicating enabling memory capture.
  + disableStack: Boolean - (optional, default: false) the flag indicating disabling stack capture.
  + disableStorage: Boolean - (optional, default: false) the flag indicating disabling storage capture.
  + enableReturnData: Boolean - (optional, default: false) the flag indicating enabling return data capture.
  + timeOut: String - (optional, default: "5s") the timeout for cancellation of execution.
  + tracer: String - (default: "structTracer") defines the debug tracer used for given call. Supported values: structTracer, callTracer.


#### Returns

* Array - array of trace objects with the following fields:

  * failed: Boolean - the tx is successful or not
  * gas: QUANTITY - the total consumed gas in the tx
  * returnValue: DATA - the return value of the executed contract call
  * structLogs: Array - the trace result of each step with the following fields:

    + pc: QUANTITY - the current index in bytecode
    + op: String - the name of current executing operation
    + gas: QUANTITY - the available gas ßin the execution
    + gasCost: QUANTITY - the gas cost of the operation
    + depth: QUANTITY - the number of levels of calling functions
    + error: String - the error of the execution
    + stack: Array - array of values in the current stack
    + memory: Array - array of values in the current memory
    + storage: Object - mapping of the current storage
    + refund: QUANTITY - the total of current refund value

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_traceBlockByNumber","params":["0x1190f352179918be580bda87e6bbe563d48ac2949e6041e5bf445dbd80a6ce60", {}],"id":1}'
````



## debug\_traceBlockByNumber

Executes all transactions in the block specified by number with a tracer and returns the tracing result.

#### Parameters

* QUANTITY|TAG - integer of a block number, or the string "latest"
* Object - the tracer options:

  + enableMemory: Boolean - (optional, default: false) the flag indicating enabling memory capture.
  + disableStack: Boolean - (optional, default: false) the flag indicating disabling stack capture.
  + disableStorage: Boolean - (optional, default: false) the flag indicating disabling storage capture.
  + enableReturnData: Boolean - (optional, default: false) the flag indicating enabling return data capture.
  + timeOut: String - (optional, default: "5s") the timeout for cancellation of execution.
  + tracer: String - (default: "structTracer") defines the debug tracer used for given call. Supported values: structTracer, callTracer.

#### Returns

* Array - array of trace objects with the following fields:

  * failed: Boolean - the tx is successful or not
  * gas: QUANTITY - the total consumed gas in the tx
  * returnValue: DATA - the return value of the executed contract call
  * structLogs: Array - the trace result of each step with the following fields:

    + pc: QUANTITY - the current index in bytecode
    + op: String - the name of current executing operation
    + gas: QUANTITY - the available gas ßin the execution
    + gasCost: QUANTITY - the gas cost of the operation
    + depth: QUANTITY - the number of levels of calling functions
    + error: String - the error of the execution
    + stack: Array - array of values in the current stack
    + memory: Array - array of values in the current memory
    + storage: Object - mapping of the current storage
    + refund: QUANTITY - the total of current refund value

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_traceBlockByNumber","params":[10, {}],"id":1}'
````



## debug\_traceBlockFromFile

Executes all transactions in a block read from a file, using a tracer, and returns the trace result.

#### Parameters

* String - path to the file containing the serialized block data.
* Object - the tracer options:

  + enableMemory: Boolean - (optional, default: false) the flag indicating enabling memory capture.
  + disableStack: Boolean - (optional, default: false) the flag indicating disabling stack capture.
  + disableStorage: Boolean - (optional, default: false) the flag indicating disabling storage capture.
  + enableReturnData: Boolean - (optional, default: false) the flag indicating enabling return data capture.
  + timeOut: String - (optional, default: "5s") the timeout for cancellation of execution.
  + tracer: String - (default: "structTracer") defines the debug tracer used for given call. Supported values: structTracer, callTracer.

#### Returns

* Array - array of trace objects with the following fields:

  * failed: Boolean - the tx is successful or not
  * gas: QUANTITY - the total consumed gas in the tx
  * returnValue: DATA - the return value of the executed contract call
  * structLogs: Array - the trace result of each step with the following fields:

    + pc: QUANTITY - the current index in bytecode
    + op: String - the name of current executing operation
    + gas: QUANTITY - the available gas ßin the execution
    + gasCost: QUANTITY - the gas cost of the operation
    + depth: QUANTITY - the number of levels of calling functions
    + error: String - the error of the execution
    + stack: Array - array of values in the current stack
    + memory: Array - array of values in the current memory
    + storage: Object - mapping of the current storage
    + refund: QUANTITY - the total of current refund value

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"debug_traceBlockFromFile","params":["block.txt", {}],"id":1}'
````



## debug_traceCall

Executes a new message call with a tracer and returns the tracing result.

#### Parameters

* Object - the transaction call object

  + from: DATA, 20 Bytes - (optional) The address the transaction is sent from.
  + to: DATA, 20 Bytes - the address the transaction is directed to.
  + gas: QUANTITY - (optional) integer of the gas provided for the transaction execution. eth_call consumes zero gas, but this parameter may be needed by some executions.
  + gasPrice: QUANTITY - (optional) integer of the gasPrice used for each paid gas
  + value: QUANTITY - (optional) integer of the value sent with this transaction
  + data: DATA - (optional) hash of the method signature and encoded parameters. For details see Ethereum Contract ABI in the Solidity documentation

* QUANTITY|TAG - integer block number, or the string "latest"
* Object - the tracer options:

  + enableMemory: Boolean - (optional, default: false) the flag indicating enabling memory capture.
  + disableStack: Boolean - (optional, default: false) the flag indicating disabling stack capture.
  + disableStorage: Boolean - (optional, default: false) the flag indicating disabling storage capture.
  + enableReturnData: Boolean - (optional, default: false) the flag indicating enabling return data capture.
  + timeOut: String - (optional, default: "5s") the timeout for cancellation of execution.
  + tracer: String - (default: "structTracer") defines the debug tracer used for given call. Supported values: structTracer, callTracer.

#### Returns

* Object - trace object, with the following fields:

  * failed: Boolean - the tx is successful or not
  * gas: QUANTITY - the total consumed gas in the tx
  * returnValue: DATA - the return value of the executed contract call
  * structLogs: Array - the trace result of each step with the following fields:

    + pc: QUANTITY - the current index in bytecode
    + op: String - the name of current executing operation
    + gas: QUANTITY - the available gas ßin the execution
    + gasCost: QUANTITY - the gas cost of the operation
    + depth: QUANTITY - the number of levels of calling functions
    + error: String - the error of the execution
    + stack: Array - array of values in the current stack
    + memory: Array - array of values in the current memory
    + storage: Object - mapping of the current storage
    + refund: QUANTITY - the total of current refund value

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"debug_traceCall","params":[{"to": "0x1234", "data": "0x1234"}, "latest", {}],"id":1}'
````



## debug_traceChain

The TraceChain method traces a range of blocks within the blockchain and provides detailed tracing results for each block in the specified range. This includes tracing transaction execution and capturing intermediate states or any errors encountered during tracing.

#### Parameters

* start: QUANTITY|TAG - integer of a block number, or the string "latest"
* end: QUANTITY|TAG - integer of a block number, or the string "latest"
* Object - The tracer options:

  + enableMemory: Boolean - (optional, default: false) the flag indicating enabling memory capture.
  + disableStack: Boolean - (optional, default: false) the flag indicating disabling stack capture.
  + disableStorage: Boolean - (optional, default: false) the flag indicating disabling storage capture.
  + enableReturnData: Boolean - (optional, default: false) the flag indicating enabling return data capture.
  + timeOut: String - (optional, default: "5s") the timeout for cancellation of execution.
  + tracer: String - (default: "structTracer") defines the debug tracer used for given call. Supported values: structTracer, callTracer.

#### Returns
* Array - an array of BlockTraceResult objects with the following fields:

  * Block: DATA - block number corresponding to this trace
  * Hash: DATA, 32 Bytes - block hash corresponding to this trace
  * Array - array of trace objects with the following fields:

    + failed: Boolean - the tx is successful or not
    + gas: QUANTITY - the total consumed gas in the tx
    + returnValue: DATA - the return value of the executed contract call
    + structLogs: Array - the trace result of each step with the following fields:

      - pc: QUANTITY - the current index in bytecode
      - op: String - the name of current executing operation
      - gas: QUANTITY - the available gas ßin the execution
      - gasCost: QUANTITY - the gas cost of the operation
      - depth: QUANTITY - the number of levels of calling functions
      - error: String - the error of the execution
      - stack: Array - array of values in the current stack
      - memory: Array - array of values in the current memory
      - storage: Object - mapping of the current storage
      - refund: QUANTITY - the total of current refund value
  * Error: String - any error encountered during tracing for the specific block.

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_traceChain","params":[10, 10, {}],"id":1}'
````



## debug_traceTransaction

Executes the transaction specified by transaction hash with a tracer and returns the tracing result.

#### Parameters

* DATA, 32 Bytes </b> - hash of a transaction.
* Object </b> - the tracer options:

  + enableMemory: Boolean </b> - (optional, default: false) the flag indicating enabling memory capture.
  + disableStack: Boolean </b> - (optional, default: false) the flag indicating disabling stack capture.
  + disableStorage: Boolean </b> - (optional, default: false) the flag indicating disabling storage capture.
  + enableReturnData: Boolean </b> - (optional, default: false) the flag indicating enabling return data capture.
  + timeOut: String </b> - (optional, default: "5s") the timeout for cancellation of execution.
  + tracer: String </b> - (default: "structTracer") defines the debug tracer used for given call. Supported values: structTracer, callTracer.

#### Returns

* Object - trace objects, with the following fields:

  * failed: Boolean - the tx is successful or not
  * gas: QUANTITY - the total consumed gas in the tx
  * returnValue: DATA - the return value of the executed contract call
  * structLogs: Array - the trace result of each step with the following fields:

    + pc: QUANTITY - the current index in bytecode
    + op: String - the name of current executing operation
    + gas: QUANTITY - the available gas ßin the execution
    + gasCost: QUANTITY - the gas cost of the operation
    + depth: QUANTITY - the number of levels of calling functions
    + error: String - the error of the execution
    + stack: Array - array of values in the current stack
    + memory: Array - array of values in the current memory
    + storage: Object - mapping of the current storage
    + refund: QUANTITY - the total of current refund value

#### Example

````bash
curl https://rpc-endpoint.io:8545 -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"debug_traceTransaction","params":["0xdc0818cf78f21a8e70579cb46a43643f78291264dda342ae31049421c82d21ae", {}],"id":1}'
````



## debug\_verbosity

This method is used to set the global logging verbosity level of the application, impacting how much detail is logged. Higher verbosity levels generally provide more detailed logs, while lower levels limit logs to critical messages or warnings.

#### Parameters

* level: QUANTITY - specifies the logging verbosity level.
  + 0 - NoLevel
  + 1 - Trace
  + 2 - Debug
  + 3 - Info
  + 4 - Warn
  + 5 - Error
  + 6 - Off 

#### Returns

* String - contains the string representation of the new verbosity level.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_verbosity","params":[4],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "warn"
}
```

</details>



## debug\_writeBlockProfile

The WriteBlockProfile method writes a goroutine blocking profile to the specified file.

#### Parameters

* file: String - the path to the file where the blocking profile will be written. The path can be relative or absolute.

#### Returns

* String - the absolute path of the file where the blocking profile has been saved.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_writeBlockProfile","params":["blockprofile.txt"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/blockprofile.txt"
}
```

</details>



## debug\_writeMemProfile

The WriteMemProfile method writes an allocation profile (memory profile) to the specified file.

#### Parameters

* file: String - the path to the file where the memory profile will be written. The path can be relative or absolute.

#### Returns

* String - the absolute path of the file where the memory profile has been saved.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_writeMemProfile","params":["memprofile.txt"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/memprofile.txt"
}
```

</details>



## debug\_writeMutexProfile

The WriteMutexProfile method writes a goroutine blocking (mutex) profile to the specified file.

#### Parameters

* file: String - the path to the file where the mutex profile will be written. The path can be relative or absolute.

#### Returns

* String - the absolute path of the file where the mutex profile has been saved.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"debug_writeMutexProfile","params":["mutexprofile.txt"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "/home/blade/mutexprofile.txt"
}
```

</details>
