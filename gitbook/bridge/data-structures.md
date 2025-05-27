# Data Structures

***

## BridgeMessage

Data from every emitted `BridgeMsg` event on the source network is mapped to the `BridgeMessage` data type.

### Fields:

* `id (uint256)`: ID of the `BridgeMsg` event. Each bridge and each side (internal and external) maintains its own counter.
* `sourceChainId (uint256)`: ID of the source chain.
* `destinationChainId (uint256)`: ID of the destination chain.
* `sender (address)`: Address of the transaction sender.
* `receiver (address)`: Address of the predicate contract on the destination chain.
* `isRollback (bool)`: Indicates if this is a rollback message.
* `payload (bytes)`: The message payload.

***

## BridgeMessageBatch

Primary data structure used by Blade to communicate with other chains.

### Fields:

* `messages (BridgeMessage[])`: An array of `BridgeMessage` objects. All messages must share the same source and destination chain IDs.
* `sourceChainId (uint256)`: ID of the source chain.
* `destinationChainId (uint256)`: ID of the destination chain.
* `threshold (uint256)`: Block number before which this batch must be executed. If not, the batch goes into retry.
* `commitCounter (uint256)`: Counter indicating how many times this batch has been committed (including retries).

***

## SignedBridgeMessageBatch

Wraps a `BridgeMessageBatch` and includes metadata necessary for signature verification.

### Fields:

* `batch (BridgeMessageBatch[])`: Array of `BridgeMessageBatch` objects.
* `signature ([2]uint256)`: Signature of the batch field.
* `bitmap (bytes)`: Bitmap indicating which validators signed the batch.
* `validatorSetBatchId (uint256)`: If greater than zero, represents a validator set batch. Used by relayers for ordering.

***

## Validator

Used by Gateway and BridgeStorage contracts for batch signature verification.

### Fields:

* `_address (address)`: Address of the validator.
* `blsKey ([4]uint256)`: Public BLS key.
* `votingPower (uint256)`: Validator's assigned voting power.

***

## BlockMetadata

Used to verify signatures for new validator sets; derived from epoch blocks.

### Fields:

* `blockHash (uint256)`: Hash of the block.
* `blockRound (uint256)`: Round number of the block.
* `epochNumber (uint256)`: Epoch number.

***

## SignedValidatorSet

Committed at the beginning of a new epoch if the validator set changes.

### Fields:

* `newValidatorSet (Validator[])`: Array of new validator set members.
* `signature ([2]uint256)`: Signature of the validator set.
* `bitmap (bytes)`: Bitmap indicating which validators signed the set.
* `blockMetadata (BlockMetadata)`: Metadata from the block used for verifying the validator signatures.

***
