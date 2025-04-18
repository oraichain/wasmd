import { ethers, getSigners } from "hardhat";
import { ERC20Token__factory } from "../typechain-types";

async function main() {
  const [owner] = getSigners(1);
  const deploy = await new ERC20Token__factory(owner).deploy(
    "USDT token",
    "USDT",
    ethers.BigNumber.from("10000000000000000000000000")
  );
  const result = await deploy.deployed();
  console.log(result.address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
