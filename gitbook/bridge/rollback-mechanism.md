# Rollback Mechanism

In the bridging system, there are two types of messages:

* **Ordinary messages**
* **Rollback messages**

The message type is determined by the `isRollback` flag.

***

## Ordinary Messages

* Represent **original messages** created as a result of a user’s intent to transfer assets or data across chains.
* Emitted by the **Gateway** via the `sendBridgeMsg` method.
* Have `isRollback = false`.
* Handled on the destination chain by the `executeBridgeMessage` method.

***

## Rollback Messages

* Triggered when the **execution of an ordinary message fails** on the destination chain.
* Treated as the **inverse** of ordinary messages:
  * Same message content
  * Source and destination chain IDs are **swapped**
  * `isRollback = true`
* Always succeed upon execution.
* Handled on the destination chain by the `executeRollbackBridgeMessage` method.

***

## Message Buckets in Blade

The Blade node classifies and stores messages into two buckets:

### 1. Ordinary Bucket

* A message enters this bucket when a `BridgeMsg` event is emitted.
* A message is **removed** from this bucket upon:
  * Successful execution (via `BridgeMessageResult` event), or
  * Unsuccessful execution → message is moved to the rollback bucket.

### 2. Rollback Bucket

* A message enters this bucket when execution of the ordinary message fails.
* The message remains in this bucket until it is **committed** to `BridgeStorage`.

> Note: While ordinary messages could also be deleted upon commitment, they remain in the ordinary bucket in case of failure and a need to transition to rollback.

***

## Execution and Commitment Rules

### Similarities

* Ordinary and rollback messages are treated **equally** during:
  * Batch creation
  * Batch commitment
  * Batch execution
* Each message (ordinary or rollback) can be **committed and executed only once**.

### Commitment Tracking

* Handled via:
  * `rollbackedE2I`
  * `rollbackedI2E`
* Key: **External chain ID**
* Value: **List of committed rollback message IDs**

### Execution Tracking

* Handled via `processedEventsRollback` map.
* Key: **Rollback message ID**
* Value: `true` / `false` indicating whether the message has been executed.

***
