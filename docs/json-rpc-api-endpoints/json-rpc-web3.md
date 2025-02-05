# web3

## web3\_clientVersion

Returns the current client version.

#### Parameters

None

#### Returns

* String - the current client version.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "blade/<version>"
}
```

</details>



## web3\_sha3

Returns Keccak-256 (not the standardized SHA3-256) of the given data.

#### Parameters

* DATA - the data to convert into a SHA3 hash.

#### Returns

* DATA - the SHA3 result of the given string.

#### Example

```bash
curl https://rpc-endpoint.io:8545 -X POST --data '{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":1}'
```

<details>

<summary>JSON result ↓</summary>

```bash
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"
}
```

</details>
