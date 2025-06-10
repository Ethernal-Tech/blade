# Storage

The entire **WST** is stored in a **key-value database** persisted on disk.

Storage serves as an abstraction layer for accessing the databases used to persist blockchain data.\
Blade supports both GoLevelDb and Pebble database for persisting blockchain data. Pebble is set by default.

The following image presents:

* The main **Storage API** functions
* A **high-level design diagram** of the storage layer

<figure><img src="../../../.gitbook/assets/storage.png" alt=""><figcaption><p>Storage example</p></figcaption></figure>
