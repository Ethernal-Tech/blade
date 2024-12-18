# Getting Started

In this section we'll initialize and start a single Blade node with PolyBFT consensus.

## System requirements
The hardware requirements for running a Blade node depend upon the node configuration and can change over time as upgrades to the network are implemented. Typical requirements are listed below.

### AWS requirements
Recommended instance for running Blade is c6a.2xlarge.

### Alibaba requirements
Recommended instance for running Blade is ecs.c6a.2xlarge.

### Other/Custom requirements
In general it is recommended to run Blade on machines with 16 GB RAM and 8 CPUs to achieve maximal performances. If low traffic is expected most of the time it is also acceptable to go with lower number of CPUs, for instance 4 and 8 GB RAM.

## Install Blade
<details>
<summary><b>Run Blade from a Docker image</b></summary>
<br>
Blade provides a Docker image to run a Blade node in a Docker container. Use this Docker image to run a single Blade node without installing Blade.

#### Prerequisites
* Docker
* Linux or MacOS
> **Info**<br>
> The Docker image doesn't run on Windows.

#### Pull the image
 ```bash
docker pull 0xethernal/blade:latest
```
</details>

<details>
<summary><b>Install Blade from packaged binaries</b></summary>

#### Linux
Download the Blade packaged binaries. Unpack the downloaded files and change into blade-`<release>` directory. Display Blade command line help to confirm installation from blade-`<release>` directory:
```bash
./blade
```
</details>

## Initialization (required only before the 1st run)
### Secrets
New accounts and corresponding validator keys for signing will be generated with a command:

<details>
<summary><b>Docker</b></summary>

```bash
docker run -v <your-local-directory>:/container-dir -w /container-dir 0xethernal/blade:latest secrets init --data-dir data --insecure
```
</details>

<details>
<summary><b>Binaries</b></summary>

```bash
./blade secrets init --data-dir data --insecure
```
</details>
<br>
<details>
<summary>Output example ↓</summary>

```bash
[WARNING: INSECURE LOCAL SECRETS - SHOULD NOT BE RUN IN PRODUCTION]

[SECRETS GENERATED]
network-key, validator-key, validator-bls-key

[SECRETS INIT]
Public key (address) = 0x61324166B0202DB1E7502924326262274Fa4358F
BLS Public key       = 06d8d9e6af67c28e85ac400b72c2e635e83234f8a380865e050a206554049a222c4792120d84977a6ca669df56ff3a1cf1cfeccddb650e7aacff4ed6c1d4e37b055858209f80117b3c0a6e7a28e456d4caf2270f430f9df2ba37221f23e9bbd313c9ef488e1849cc5c40d18284d019dde5ed86770309b9c24b70ceff6167a6ca
Node ID              = 16Uiu2HAmMYyzK7c649Tnn6XdqFLP7fpPB2QWdck1Ee9vj5a7Nhg8
```
</details>

#### Understand the Generated Secrets

The generated secrets include the following information for a validator node:

- **ECDSA Private and Public Keys**: These keys are used to sign and verify transactions on the blockchain.
- **BLS Private and Public Keys**: These keys are used in the Byzantine fault-tolerant (BFT) consensus protocol to aggregate and verify signatures efficiently.
- **P2P Networking Node ID**: This is a unique identifier for each validator node in the network, allowing them to establish and maintain connections with other nodes.

> **Info**<br>
> The secrets output can be retrieved again if needed by running the following command: `docker run -v <your-local-directory>:/container-dir -w /container-dir 0xethernal/blade:latest secrets output --data-dir data`

### Genesis
Within `<your-local-directory>` there should be genesis.json file, default path is in the root of working directory. In order to generate genesis.json run following command:

<details>
<summary><b>Docker</b></summary>

```bash
docker run -v <your-local-directory>:/container-dir -w /container-dir 0xethernal/blade:latest genesis --reward-wallet 0xDEADBEEF --premine 0x0000000000000000000000000000000000000000 --proxy-contracts-admin 0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed --blade-admin 0x61324166B0202DB1E7502924326262274Fa4358F --validators /ip4/127.0.0.1/tcp/1478/p2p/16Uiu2HAmMYyzK7c649Tnn6XdqFLP7fpPB2QWdck1Ee9vj5a7Nhg8:0x61324166B0202DB1E7502924326262274Fa4358F:06d8d9e6af67c28e85ac400b72c2e635e83234f8a380865e050a206554049a222c4792120d84977a6ca669df56ff3a1cf1cfeccddb650e7aacff4ed6c1d4e37b055858209f80117b3c0a6e7a28e456d4caf2270f430f9df2ba37221f23e9bbd313c9ef488e1849cc5c40d18284d019dde5ed86770309b9c24b70ceff6167a6ca
```
</details>
<details>
<summary><b>Binaries</b></summary>

```bash
./blade genesis --reward-wallet 0xDEADBEEF --premine 0x0000000000000000000000000000000000000000 --proxy-contracts-admin 0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed --blade-admin 0x61324166B0202DB1E7502924326262274Fa4358F --validators /ip4/127.0.0.1/tcp/1478/p2p/16Uiu2HAmMYyzK7c649Tnn6XdqFLP7fpPB2QWdck1Ee9vj5a7Nhg8:0x61324166B0202DB1E7502924326262274Fa4358F:06d8d9e6af67c28e85ac400b72c2e635e83234f8a380865e050a206554049a222c4792120d84977a6ca669df56ff3a1cf1cfeccddb650e7aacff4ed6c1d4e37b055858209f80117b3c0a6e7a28e456d4caf2270f430f9df2ba37221f23e9bbd313c9ef488e1849cc5c40d18284d019dde5ed86770309b9c24b70ceff6167a6ca
```
</details>

<br>

* proxy-contracts-admin and blade-admin are some Ethereum accounts
* validators is array of validators in the network in the format `<P2P multi address>:<ECDSA address>:<public BLS key>`. If param bootnode is omitted then genesis validators will be also bootnodes. In the example above a local validator will be a bootnode as well, hence genesis is generated with 127.0.0.1. Other validators connecting to that bootnode/validator should have set private/public IP address of the bootnode/validator during genesis generation. There can be multiple bootnodes (up to the total number of validators).
> **Info**<br>
> More information about Blade configuration parameters can be found in section CLI.
>
> **Warning**  
> Permission over `<your-local-directory>` should be set to 777 (only for Docker deployment).

## Start Blade
<details>
<summary><b>Docker</b></summary>
<br>
Default exposed ports are:
* 8545 - json rpc port
* 9632 - grpc
* 1478 - p2p discovery
* 5001 - prometheus

If you don’t have to change default ports start blade with:
```bash
docker run 0xethernal/blade:latest
```

If you want to change ports then start Blade with:
```bash
docker run -p <localportJSON-RPC>:8545 -p <localportGRPC>:9632 -p <localportP2P>:1478 0xethernal/blade:latest
```

Minimal docker command would be
```bash
docker run --name blade -v <your-local-dir>:/container-dir -w /container-dir 0xethernal/blade:latest server --data-dir data
```
* --name is optional and that will be docker container name, otherwise default is used
* -v mounts `<your-local-dir>` as a container directory, container-dir in the example
* -w sets mounted container directory as a working container directory
* --data-dir sets path to data folder within container working directory

### Stop Blade and clean up resources
When done running a node, you can shut down the node container without deleting resources or you can delete the container after stopping it. Run `docker container ls` and `docker volume ls` to get the container and volume names.

To stop a container:
```bash
docker stop <container-name>
```

To delete a container:
```bash
docker rm <container-name>
```
</details>

<details>
<summary><b>Binaries</b></summary>
<br>

Use the blade command with the required command line flags to start a node:
```bash
./blade server --data-dir data
```
</details>