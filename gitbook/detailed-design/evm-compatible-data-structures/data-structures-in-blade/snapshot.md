# Snapshot

A **Snapshot** represents the state of a **subtree of the World State Trie (WST)** at a specific moment in time.

* The **subtrie** is defined by specifying a **root hash**.
* It enables:
  * **Reading** a subset of blockchain state data
  * **Running transactions** against that state
  * **Committing** the updated changes back to storage

Snapshots are created using a `State` object, which provides the interface for snapshot initialization and manipulation.
