## txpool_content

Returns a list with the exact details of all the transactions currently pending for inclusion in the next block(s), as well as the ones that are being scheduled for future execution only.

### Parameters

None

### Example
````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"txpool_content","params":[],"id":1}'
````
<details>
<summary><b>JSON result ↓</b></summary>

````bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "pending": {},
    "queued": {}
  }
}
````
</details>
<br>

## txpool_contentFrom

Returns a list with the exact details of all the transactions sent from the address currently pending for inclusion in the next block(s), as well as the ones that are being scheduled for future execution only.

### Parameters

* <b> DATA, 20 Bytes </b> - address of the sender.

### Example
````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"txpool_contentFrom","params":["0x85da99c8a7c2c95964c8efd687e95e632fc533d6"],"id":1}'
````
<details>
<summary><b>JSON result ↓</b></summary>

````bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "pending": {},
    "queued": {}
  }
}
````
</details>
<br>

## txpool_inspect

Returns a list with a textual summary of all the transactions currently pending for inclusion in the next block(s), as well as the ones that are being scheduled for future execution only. This is a method specifically tailored to developers to quickly see the transactions in the pool and find any potential issues.

### Parameters

None

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"txpool_inspect","params":[],"id":1}'
````
<details>
<summary><b>JSON result ↓</b></summary>

````bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "pending": {},
    "queued": {},
    "currentCapacity": 0,
    "maxCapacity": 20000000
  }
}
````
</details>
<br>

## txpool_status

Returns the number of transactions currently pending for inclusion in the next block(s), as well as the ones that are being scheduled for future execution only.

### Parameters

None

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"txpool_status","params":[],"id":1}'
````
<details>
<summary><b>JSON result ↓</b></summary>

````bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "pending": 0,
    "queued": 0
  }
}
````
</details>
