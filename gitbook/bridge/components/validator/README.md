# Validator

From the perspective of **bridge functionality**, the **validator node** is responsible for:

* Preparing batches
* Proposing them for commitment to the blockchain
* Voting
* Applying bridge transactions

Each of these steps involves multiple validations to ensure **safe and secure operation**.

***

## BridgeManager Component

The key component in a validator that handles most bridge-related actions is the **BridgeManager**.\
There is a **dedicated BridgeManager instance for each bridged blockchain**.

Each BridgeManager hosts:

* An instance of **EventTracker**
* An instance of **EventProvider**

These are used to **track events on both external and internal chains**.

***

## Event Tracking and Processing

* The **EventTracker** monitors events and, for each `BridgeMsgEvent`, it invokes:
  * `BridgeEventManager::AddLog()` — entry point for processing events from **external chains**
  * `BridgeEventManager::ProcessLog()` — entry point for processing events from the **internal chain (Blade)**

***

## Event Processing Steps

1. **Validation**
   * Check chain ID
   * Validate event type
   * Verify event format
2. **Persistence**
   * Add the event to a **local (but configurable) database**
   * **BoltDB** is used for this purpose
3. **Batching**
   * Instruct the **BridgeManager** to batch a set of transactions
   * These transactions are processed together as a **single bridge transaction**, referred to as a **bridge batch**
