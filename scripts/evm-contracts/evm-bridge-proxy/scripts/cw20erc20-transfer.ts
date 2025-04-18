import { getSigners } from "hardhat";
import { CW20ERC20Token__factory } from "../typechain-types";

async function main() {
  const erc20Address = process.env.ERC20_ADDRESS!;
  const [owner] = getSigners(1);
  const deployed = CW20ERC20Token__factory.connect(erc20Address, owner);
  const result = await deployed.transfer(erc20Address, "1", {
    gasLimit: 90000000,
  });
  const receipt = await result.wait();
  console.log(receipt);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });