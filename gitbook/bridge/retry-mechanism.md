# Retry Mechanism

In bridging from an **internal** to an **external** chain, the **relayer** is responsible for transferring and executing batches. However, committing a batch to `BridgeStorage` **does not guarantee** immediate execution on the external chain. Delays or failures may occur.

While some recovery is possible (if the validator set hasn’t changed), a **retry mechanism** is necessary to protect against malicious behavior and ensure eventual execution.

***

## Batch Identification

Each batch is uniquely identified by a **base hash**, computed as the hash of:

* **Source chain ID**
* **Destination chain ID**
* **List of message IDs** (in the current implementation, complete messages are used instead)

***

## Retry Logic Overview

1. **Wait** a specific period for batch execution on the external chain.
2. If not executed by then, **initiate retry**.
3. Modify:
   * **Threshold** (block number deadline on the external chain)
   * **Commit counter** (sequence number of the commitment)
4. Trigger **new round** of voting, commitment, and waiting.
5. Repeat until the batch is successfully executed.

***

## Additional Metadata Per Batch

Two additional fields are required for retry support:

* **Threshold**:\
  Block number on the external chain until which the system waits.\
  If exceeded, retry is initiated.\
  If execution is attempted after threshold, the batch is rejected.
* **Commit Counter**:\
  Indicates the number of times a batch has been committed.
  * Original batch → `commitCounter = 1`
  * On each retry → incremented by 1
  * Helps differentiate between:
    * **Original batches** (apply message-related checks)
    * **Retry batches** (skip message checks)

To prevent **infinite retries** under the same validator set, the system enforces that the `commitCounter` must **strictly increase** for the same batch.\
This is tracked using the `batchCommitCounter` map in `BridgeStorage`, where:

* **Key**: Unique batch identifier (base hash)
* **Value**: Last commit counter used for that batch

***

## Internal (Blade) Logic for Retry Support

### Threshold & Commit Counter Calculation

* **Threshold**:\
  Derived by reading the latest block number on the external chain + a configurable buffer value.
* **Commit Counter**:
  * If original batch → set to 1
  * If retry → read last value from `BridgeStorage` using base hash, increment by 1

***

## Retry Tracking

To manage retry flow, two **mutually exclusive** data structures are used:

1. **unexecutedBatches**:
   * Holds batches **waiting for execution**
   * Added on commitment to `BridgeStorage`
   * Removed when executed or expired (moved to retry list)
2. **retryBatches** (implemented as a map):
   * Holds batches for which **retry has been initiated**
   * Remains until a retry version is successfully committed
   * Then removed and re-added to `unexecutedBatches`

***

### Threshold Expiration Check

* Triggered on **PostBlock** (on each new internal chain block)
* Reads the **external chain's block number**
* Compares with each batch's threshold in `unexecutedBatches`
* Accounts for potential **block reorgs**
* Expired batches are moved to `retryBatches`

***

## Retry Mechanism Flow

1. Create original batch → **vote** → **commit to BridgeStorage**
2. Verify batch on BridgeStorage:
   * Signature
   * Message commitment
   * `commitCounter > 0`
3. Add to `unexecutedBatches`, wait for threshold
4. On **threshold expiration**, move batch to `retryBatches`
5. Initiate retry:
   * Set new threshold
   * Set `commitCounter = 2`
6. **Vote** → **commit retry batch to BridgeStorage**
7. Verify on BridgeStorage:
   * Signature
   * `commitCounter > 1` (skip message checks)
8. Remove from `retryBatches`, insert into `unexecutedBatches`, wait for threshold again
9. **Repeat steps 4–8** until the batch is executed
