# ValidatorSetStorage

Signature verification plays a crucial role in the `Gateway` and `BridgeStorage` contracts, with `ValidatorSetStorage` serving as the foundation for this process.\
Both `BridgeStorage` and `Gateway` inherit from `ValidatorSetStorage` and rely on it to validate batch signatures.\
This ensures the **integrity of batches**, confirming that the data remains **untampered** and originates from **trusted validators**.

***

## Maintaining the Validator Set

Equally important is maintaining an **up-to-date and reliable validator set**.\
The `commitValidatorSet` function in the `ValidatorSetStorage` contract allows updates whenever necessary, ensuring that the latest validator set is always stored.

* When a new batch of validators is introduced:
  * The existing validators verify a **metadata signature** included with the new batch.
  * This ensures the **correctness of the new validator set**.

***

## Summary

By enforcing **strict signature verification** and continuously **updating the validator set**,\
the `BridgeStorage` and `Gateway` contracts uphold the **security and reliability** of **cross-network transactions**.
