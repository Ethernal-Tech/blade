# Predicate contracts

## RootTokenPredicate

The `RootTokenPredicate` contract is responsible for the deposit and withdrawal of funds on the root chain.

### `deposit()` Function

The `deposit()` call receives the source token and destination address to transfer the funds to and performs the following:

* **Mapping** of the root token to the child (destination) token.
* **Safely transfers tokens** to the contract address for safekeeping during the deposit operation.
  * The tokens are held at the contract address until a withdrawal is requested from the external network to the internal network.
* **Invokes `Gateway::sendBridgeMessage()`**, which emits the `BridgeMsg` event used by the Relayer to transfer funds to the client chain.

### `_withdraw()` Function

The `_withdraw()` function is called from the `Gateway` contract during message execution on the destination chain. It receives:

* Root token address
* Withdrawer’s address
* Receiver’s address
* Amount

It performs the following actions:

* Retrieves the **mapped child token** from the mapping.
* **Safely transfers tokens** from the contract address to the receiver’s address.

***

## ChildTokenPredicate

The `ChildTokenPredicate` contract is responsible for receiving tokens on the external chain. Transfer is initiated via the `onStateReceive` call from the `Gateway` and has the following roles:

* **Validates** the sender address and message type.
* **Maps** the root token to the child token.
* **Mints** the requested amount of tokens.

### `withdraw()` Function

The `withdraw()` call is responsible for withdrawing the tokens to the internal chain and consists of the following steps:

* **Burns** the withdrawn tokens.
* **Invokes `Gateway::sendBridgeMessage()`**, which emits the `BridgeMsg` event used by the Relayer to complete the transfer.

> **Note:** Minting and burning of tokens is performed **only on the external chain**.
