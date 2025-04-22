# net

## net\_listening

Whether the client is actively listening for network connections.

#### Parameters

None

#### Returns

* Boolean - true when listening, otherwise false.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"net_listening","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": true
}
```

</details>



## net\_peerCount

Returns number of peers currently connected to the client.

#### Parameters

None

#### Returns

* QUANTITY - number of connected peers in hexadecimal.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "0x6"
}
```

</details>



## net\_version

Returns the current network id.

#### Parameters

None

#### Returns

* String - the current network id.

#### Example

```bash
curl https://rpc-endpoint.io:8545 --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "100"
}
```

</details>
