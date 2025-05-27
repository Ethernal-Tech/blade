# BridgeStorage

Alongside the `Gateway`, the `BridgeStorage` contract plays a central role in asset transfers between two different networks.

## Key Responsibilities

* **Storing signed batches** and **returning requested batches** upon a relayer's request.
* **Tracking message IDs** that are stored within the batches.

***

## Batch Verification

* Several **checks are performed** on incoming batches and the messages they contain.
* If all checks pass, the batch is considered **verified**.
  * **Verification** occurs only on the **first commitment** of a given batch.
* The **signed batch** is validated for **signature authenticity**.
* Upon successful verification:
  * The contract emits the `NewBatch(id)` event.
  * The batch is stored in the contract’s mapping.
  * Associated **event IDs** are stored in appropriate mappings depending on the **event type**:
    * **Rollback**
    * **Ordinary**

***

## External to Internal Batch Handling

* In cases where batches are transitioned **from external to internal**, `BridgeStorage` retrieves the batch from the **internal `Gateway` smart contract**.
* Since it is possible to **send a batch directly** to the internal `Gateway` contract:
  * There is **no need to first store** the batch in `BridgeStorage` before forwarding it through a relayer to the other network.
