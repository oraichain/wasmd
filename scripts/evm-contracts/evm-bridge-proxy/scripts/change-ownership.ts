import { ethers, getSigners } from "hardhat";
import { Bridge__factory } from "../typechain-types";
import { contracts } from "../constants";

// mainnet eth & bsc
const { proxyContract } = contracts;

async function main() {
  const [owner] = getSigners(1);
  const chainId = await owner.getChainId();
  console.log("chain id: ", chainId);
  // const ownerAddress = await owner.getAddress();
  const deploy = new Bridge__factory(owner).attach(proxyContract);
  const contract = await deploy.deployed();
  await contract.transferOwnership(
    ethers.utils.getAddress("0xD7F771664541b3f647CBA2be9Ab1Bc121bEEC913") // multi-sig
  );
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
