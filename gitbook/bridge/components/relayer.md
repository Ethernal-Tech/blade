# Relayer

The **relayer** plays a crucial role in **transferring batches from the internal to the external network**.\
It operates as an **independent node** that periodically queries the `BridgeStorage` contract on the **Blade network** for newly stored batches.

***

## Responsibilities

* **Detect new batches** on the internal (Blade) network.
* **Forward batches** to the corresponding external network.
* **Wait for transaction execution** on the external chain.
* Ensure **correct sequencing**: batches are processed in the **exact order** they appear in `BridgeStorage`.

***

## Types of Batches

The relayer handles two kinds of batches:

1. **Asset Transfer Batches**
   * Contain token transfers and related messages.
2. **Validator Set Batches**
   * Required for signature verification on external chains.

***

## Relayer Architecture

* Each **external network** has its own **dedicated relayer**.
* The number of **relayer nodes** must match the number of **external networks**.
* Every relayer maintains its own **BoltDB** instance to store the ID of the **last processed batch**.
  * This allows the relayer to **resume from the last executed batch** in case of a restart.

***

## Batch Transfer Workflow

1. The relayer calls `BridgeStorage::getCommittedBatches` with the **ID of the last processed batch**.
2. `BridgeStorage` returns an **array of all batches** starting from that ID.
3. For each received batch:
   * The relayer checks if it's a **validator set batch** or a **regular asset batch**.
   * If **validator set batch**:
     * The relayer calls `BridgeStorage` to retrieve the `ValidatorSetBatch`.
     * Sends it to the external network.
   * If **regular asset batch**:
     * Sends the batch **directly to the external chain**.
     * Waits for the **transaction receipt**.
