---
description: >-
  This section gives an overview of the Ethernal Blade blockchain high-level
  architecture.
---

# High-level Architecture

To better understand the Blade system, the high-level architecture diagram of components is illustrated below. A node in the diagram is any instance of Blade software connected to other computers running Blade software, forming a network. We also refer to this node as the Blade client or client for short.&#x20;

<figure><img src="../.gitbook/assets/system_architecture-high-level arch with syncer.drawio(3).png" alt=""><figcaption><p>High-level Architecture Component Diagram</p></figcaption></figure>

The client exposes predefined RPC methods through an API (JSON-RPC within the `RPC` component). Among other things, this enables the sending of transactions to the blockchain and the reading of current blockchain data.

Sent transactions are submitted to the transaction pool (represented by `TxPool` component). For reading a block from the blockchain, a transaction from a block, or the current state of an account, a request is submitted to the `Storage` component. The `Storage` component is a general abstraction for different types of storage holding various data like the blockchain itself, the world state, bridge data, etc.

The transaction pool prioritizes received transactions, usually by giving an advantage to transactions with higher gas prices, but the applied criteria may differ. Each received transaction from the transaction pool is sent to the rest of the nodes/peers in peer-to-peer network using the `libp2p` communication component. This component uses gossip mechanism (gossipsub) to spread received messages through the network.

There are two types of Blade clients: full nodes and validators. Full nodes store the entire blockchain and they do not participate in block validation. Validators, as their name suggests, participate in block validation.

If a client node is in the role of a validator, it takes part in the block creation process, and the consensus mechanism is an integral part of its operational behavior. The consensus mechanism, which is essentially an algorithm using state transitions based on the received confirmations, helps validators come to an agreement regarding the validity of a proposed block. The `Consensus` component (the algorithm) is supported by the `Consensus Backend` component.&#x20;

As the algorithm assumes that only one validator from all consensus participants/validators is a block proposer, the `Consensus Backend` also holds the logic to determine whether the validator is a proposer or not. For example, when the consensus algorithm starts, it checks the proposer status of the validator, and if the status is true, it calls the `Block Management` component, which takes transactions from the `TxPool` to form a proposed block. The consensus relies on a number of messages (such as PREPARE, COMMIT, etc.) during different stages of the block creation process. These messages are created by the `Consensus Backend` as well. As all validators equally participate in the consensus, the messages have to be distributed to all of them. The communication layer of the `Consensus Backend` ensures that all messages are sent to all peers through `libp2p`. Messages work in pub/sub manner meaning that only subscribed nodes receive messages for the given topic.&#x20;

Once all messages are exchanged through different consensus phases, and assuming that consensus is reached (meaning that the proposed block is valid and ready to be added to the blockchain), the next step is to apply all block transactions and add the block.&#x20;

The `Block Management` component takes the block and sends each transaction for execution. The `Executor` component determines whether it is a smart contract call or an external owned account (EOA) balance change, and applies them accordingly. In the case of a smart contract, the contract is retrieved from storage and sent to the `EVM` (Ethereum Virtual Machine) for execution. Consequently, the `World State` is updated by the `EVM`, adjusting the balance of the contract account and its storage. For EOAs, the `World State` is directly updated by adjusting the balance of the account. After all transactions are executed and the `World State` is updated, the block's header is also updated with the latest `World State` hash and the block is appended to the blockchain (saved in `Blocks` storage). The block is also sent to all peers using `Syncer` component.

When a node is a validator, both the `Consensus` and `Syncer` components are up and running. On the other hand, for full nodes, only the `Syncer` is running.  As mentioned, the `Consensus` serves to create blocks, while the `Syncer`'s purpose is to synchronize the node's blocks with the rest of the network. The `Syncer` connects to another node and fetches blockchain differences (i.e. blocks) by using an internal protocol that compares the indices of the latest blocks on the nodes. After a newer block is received, the `Syncer` sends it to `Block Management`, which in turn checks if the block already exists in the chain, because there is a chance that the `Consensus` created it. Otherwise, the newer block is applied by `Block Management` (the block is validated, its transactions are executed, and the updated block is appended to the blockchain). At the same time, currently running consensus execution is halted, as it is stale.

