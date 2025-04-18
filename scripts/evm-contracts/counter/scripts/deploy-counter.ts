import { getSigners } from "hardhat";
import { Counter__factory } from "../typechain-types";

async function main() {
  const [owner] = getSigners(1);
  const deploy = await new Counter__factory(owner).deploy();
  const result = await deploy.deployed();
  console.log(result.address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
