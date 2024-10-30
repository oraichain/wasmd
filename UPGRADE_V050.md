# Upgrading to v0.50.0

## New Oraichain mainnet codebase repository
During the upgrade of our mainnet from `v0.42.4` to `v0.50.0`, we transitioned our mainnet repository codebase to [Oraichain’s wasmd repository](https://github.com/oraichain/wasmd). This repository, a fork of the [CosmWasm wasmd module](https://github.com/CosmWasm/wasmd), supports the integration and execution of smart contracts within the Cosmos ecosystem and is now tailored for Oraichain's mainnet. It includes tools and configurations specifically optimized for Oraichain's blockchain, supporting WebAssembly (WASM) smart contracts, enhancing interoperability, and enabling decentralized applications. Additionally, it contains essential modules, configurations, and test scripts for our mainnet platform.

## Why we change our mainnet repository to wasmd
When upgrading our chain, we identified several advantages in transitioning to the wasmd codebase:
- Reduced Maintenance Burden: As our mainnet doesn’t rely on unique modules, switching to the `wasmd` repo allows us to avoid developing and maintaining a separate repository, saving resources and reducing code management.
- Simplified Synchronization with Cosmos SDK Updates: The `wasmd` repo is actively maintained by the community, allowing easy synchronization with the latest Cosmos SDK updates, saving time and effort.
- Upstream Logic Improvements: `wasmd` offers continuous logic optimizations and fixes from upstream, so our team only needs to focus on syncing updates without handling independent code adjustments.

## Step to build new oraid binary
To build the binary for `Oraichain v0.50.0`, follow these steps. Only proceed after an upgrade proposal has been created, passed, and the upgrade height is reached.
- First, stop your node. Consider backing up your node data in case of issues when applying the new binary.
- Clone the new `wasmd` mainnet repo, navigate to the `wasmd` folder, check out the `v0.50.0` tag, and build the binary:
```shell
git clone https://github.com/oraichain/wasmd
cd wasmd
git checkout v0.50.0
make build
```
- Verify the new binary version:
```shell
oraid version
# Expected output: v0.50.0
```
- Restart your node to apply the new binary.