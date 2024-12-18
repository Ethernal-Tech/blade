## net_listening

Whether the client is actively listening for network connections.

### Parameters

None

### Returns

* <b> Boolean </b> - true when listening, otherwise false.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"net_listening","params":[],"id":1}'
````
<details>
<summary><b>JSON result ↓</b></summary>

````bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": true
}
````
</details>
<br>

## net_peerCount

Returns number of peers currently connected to the client.

### Parameters

None

### Returns

* <b> QUANTITY </b> - number of connected peers in hexadecimal.

### Example

````bash
curl  https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}'
````
<details>
<summary><b>JSON result ↓</b></summary>

````bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "0x6"
}
````
</details>
<br>

## net_version

Returns the current network id.

### Parameters

None

### Returns

* <b> String </b> - the current network id.

### Example

````bash
curl  https://rpc-endpoint.io:8545 --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}'
````
<details>
<summary><b>JSON result ↓</b></summary>

````bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "100"
}
````
</details>
