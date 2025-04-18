import { getSigners } from "hardhat";
import { Counter__factory } from "../typechain-types";

async function main() {
  const counterAddress = process.env.COUNTER_ADDRESS!;
  const [owner] = getSigners(1);
  const deployed = new Counter__factory(owner).attach(counterAddress);
  const count = await deployed.count();
  console.log(count.toNumber());
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
