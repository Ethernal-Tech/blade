# EVM Compatible data structures

The Ethereum blockchain data structure was defined in the [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf), authored by Dr. Gavin Wood in April 2014. This foundational document established the core principles of Ethereum’s state transition system, including its use of Merkle Patricia Trees for efficient state management.

Ethereum’s data structure is based on an **account-based model** (as opposed to Bitcoin’s UTXO model), where each account maintains:

* **Balance**
* **Nonce**
* **Storage**
* **Code**

The blockchain itself is structured as a **linked list of blocks**, where each block contains:

* A **header**
* A **transaction list**
* **Receipts**

The state of the Ethereum Virtual Machine (EVM) is maintained using a **modified Merkle Patricia Trie**, enabling efficient verification and storage of account states.

These design choices allow Ethereum to support **smart contracts** and **decentralized applications (DApps)** while ensuring:

* **Security**
* **Transparency**
* **Scalability**
