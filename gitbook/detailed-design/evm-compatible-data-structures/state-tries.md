# State Tries

## World State Trie

The **World State Trie** (WST), also called the **State Trie** or **Global State Trie**, is a critical data structure that represents the current state of all accounts on the Ethereum network.

* The trie is persisted to disk.
* All of the values are **RLP-serialized accounts**.

The figure below shows an example of world state trie with detailed node structure.

<figure><img src="../../.gitbook/assets/system_architecture-world state.drawio(4).png" alt=""><figcaption><p>World State Trie Example</p></figcaption></figure>

***

## Account Storage Trie

The **Account Storage Trie** is a data structure used to store the persistent data of a smart contract. It is a **Merkle Patricia Trie**, like the World State Trie, and it maps storage keys to storage values for a specific contract account.

More details:

* Each contract account on Ethereum has its **own Account Storage Trie**, which stores the contract's persistent state variables.
* The storage trie is specific to a single contract and is **separate from the World State Trie**, which stores the global state of all accounts.
* The **root hash** of the Account Storage Trie is stored in the `storageRoot` field of the contract's account state (in the World State Trie).

***

## Transaction Trie

In the context of Ethereum, the **Transaction Trie** is a data structure used to store and organize all the transactions included in a specific block.

* Each block contains a Transaction Trie that holds all the transactions in that block.
* The transactions are indexed by their position in the block and are RLP-encoded.
* The root hash of this trie is stored in the block header (`transactionsRoot` field).

***

## Transaction Receipt Trie

The **Transaction Receipt Trie** records the **receipts (outcomes)** of transactions.

* A **receipt** is the result of a transaction that was executed successfully.
* Each receipt includes:
  * The **transaction hash**
  * **Block number**
  * **Amount of gas used**
  * **Contract address** (if a new contract was created)
  * Other execution metadata
* Like the Transaction Trie, this is a Merkle Patricia Trie whose root hash is stored in the block header (`receiptsRoot` field).
