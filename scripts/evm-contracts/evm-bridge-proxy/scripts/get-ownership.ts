import { ethers, getSigners } from "hardhat";
import { Bridge__factory } from "../typechain-types";
import assert from "assert";
import { contracts } from "../constants";

// mainnet eth & bsc
const { proxyContract } = contracts;

async function main() {
  const [owner] = getSigners(1);
  const chainId = await owner.getChainId();
  console.log("chain id: ", chainId);
  const deploy = new Bridge__factory(owner).attach(proxyContract);
  const contract = await deploy.deployed();
  const ownership = await contract.owner();
  assert.equal(
    ownership,
    ethers.utils.getAddress("0xD7F771664541b3f647CBA2be9Ab1Bc121bEEC913")
  );
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
