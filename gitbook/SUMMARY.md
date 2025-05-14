# Table of contents

## Architecture

* [Overview](README.md)
* [High-level Architecture](architecture/high-level-architecture.md)

## SYSTEM SECURITY

* [Considerations](system-security/considerations/README.md)
  * [Data Classification](system-security/considerations/data-classification.md)
  * [Risk Based Methodology](system-security/considerations/risk-based-methodology.md)
  * [Key Management](system-security/considerations/key-management.md)
* [Enterprise Integration](system-security/enterprise-integration.md)

## DETAILED DESIGN

* [Transaction Pool](detailed-design/transaction-pool-txpool.md)
* [Block Creation Process](detailed-design/block-creation-process.md)
* [Consensus Mechanism](detailed-design/consensus-mechanism/README.md)
  * [IBFT 2.0 Consensus Algorithm](detailed-design/consensus-mechanism/ibft-2.0-consensus-algorithm/README.md)
    * [Initialization](detailed-design/consensus-mechanism/ibft-2.0-consensus-algorithm/initialization.md)
    * [Implementation](detailed-design/consensus-mechanism/ibft-2.0-consensus-algorithm/implementation.md)
    * [Consensus Backend](detailed-design/consensus-mechanism/ibft-2.0-consensus-algorithm/backend.md)
* [Synchronization](detailed-design/synchronization.md)
* [Networking](detailed-design/networking.md)
* [EVM Compatible data structures](detailed-design/evm-compatible-data-structures/README.md)
  * [Introduction](detailed-design/evm-compatible-data-structures/introduction.md)
  * [Block Structure](detailed-design/evm-compatible-data-structures/block-structure.md)
  * [State Tries](detailed-design/evm-compatible-data-structures/state-tries.md)
  * [Blade](detailed-design/evm-compatible-data-structures/blade/README.md)
    * [Storage](detailed-design/evm-compatible-data-structures/blade/storage.md)
    * [State](detailed-design/evm-compatible-data-structures/blade/state.md)
    * [Snapshot](detailed-design/evm-compatible-data-structures/blade/snapshot.md)
    * [State Txn](detailed-design/evm-compatible-data-structures/blade/state-txn.md)
    * [State Transition](detailed-design/evm-compatible-data-structures/blade/state-transition.md)
  * [World State Trie Update Dataflow](detailed-design/evm-compatible-data-structures/world-state-trie-update-dataflow.md)
