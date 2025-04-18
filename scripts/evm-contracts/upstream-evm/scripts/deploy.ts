import { ethers, getSigners } from "hardhat";
import { Bridge__factory } from "../typechain-types";
import { contracts } from "../constants";

// eth
const { eth, bnb } = contracts;
let { routerAddr, wrapNativeAddr, gravityBridgeContract } = eth;

async function main() {
  const [owner] = getSigners(1);
  const chainId = await owner.getChainId();
  console.log("chain id: ", chainId);
  switch (chainId) {
    case 1:
      break;
    case 56:
      // bsc
      routerAddr = bnb.routerAddr;
      wrapNativeAddr = bnb.wrapNativeAddr;
      gravityBridgeContract = bnb.gravityBridgeContract;
      break;
    default:
      break;
  }
  const deploy = await new Bridge__factory(owner).deploy(
    gravityBridgeContract,
    routerAddr,
    wrapNativeAddr
  );
  const result = await deploy.deployed();
  console.log("deployed: ", result.address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
