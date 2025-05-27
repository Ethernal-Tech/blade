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

* [Block Structure](detailed-design/block-structure.md)
* [World State](detailed-design/world-state.md)
* [Transaction Pool](detailed-design/transaction-pool-txpool.md)
* [Block Creation Process](detailed-design/block-creation-process.md)
* [Consensus Mechanism](detailed-design/consensus-mechanism/README.md)
  * [IBFT 2.0 Consensus Algorithm](detailed-design/consensus-mechanism/ibft-2.0-consensus-algorithm/README.md)
    * [Initialization](detailed-design/consensus-mechanism/ibft-2.0-consensus-algorithm/initialization.md)
    * [Implementation](detailed-design/consensus-mechanism/ibft-2.0-consensus-algorithm/implementation.md)
    * [Consensus Backend](detailed-design/consensus-mechanism/ibft-2.0-consensus-algorithm/backend.md)
* [Synchronization](detailed-design/synchronization.md)
* [Networking](detailed-design/networking.md)

## Bridge

* [Overview](bridge/overview.md)
* [High-Level Architecture](bridge/high-level-architecture.md)
* [Components](bridge/components/README.md)
  * [BridgeStorage](bridge/components/bridgestorage.md)
  * [Gateway](bridge/components/gateway.md)
  * [Predicate contracts](bridge/components/predicate-contracts.md)
  * [ValidatorSetStorage](bridge/components/validatorsetstorage.md)
  * [Validator](bridge/components/validator/README.md)
    * [Building the Bridge Batch](bridge/components/validator/building-the-bridge-batch.md)
    * [Batch Voting](bridge/components/validator/batch-voting.md)
    * [Bridge Transaction](bridge/components/validator/bridge-transaction.md)
  * [Relayer](bridge/components/relayer.md)
* [Asset Transfer Workflow](bridge/asset-transfer-workflow.md)
* [Retry Mechanism](bridge/retry-mechanism.md)
* [Rollback Mechanism](bridge/rollback-mechanism.md)
* [Data Structures](bridge/data-structures.md)
* [Process of Token Transfer via Bridge](bridge/process-of-token-transfer-via-bridge.md)
