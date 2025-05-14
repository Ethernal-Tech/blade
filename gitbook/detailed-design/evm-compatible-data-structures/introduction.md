# Introduction

There are four types of state tries:

* **World State Trie**
* **Transaction Trie**
* **Transaction Receipt Trie**
* **Account Storage Trie**

Each state trie is constructed using a **Merkle Patricia Trie** (MPT) \[Ref].

Each block stores three central state tries:

* **World State Trie**
* **Transaction Trie**
* **Receipt Trie**

The **Account Storage Trie** (also known as the account storage contents trie) constructs the leaf nodes in the world state trie.

***

## Why Use Merkle Patricia Trie?

The Merkle Patricia Trie data structure is used for the following reasons:

* **Efficient Lookups**:\
  Tries allow for efficient key-value lookups, insertions, and deletions with `O(log(n))` performance.
* **Cryptographic Verification**:\
  The Merkle tree structure enables cryptographic proofs of the data's integrity. Each node in the trie is hashed, and the **root hash** (stored in the block header) serves as a fingerprint of the entire state.
* **Incremental Updates**:\
  Only the parts of the trie that are modified need to be updated, making it efficient for frequent state changes.
