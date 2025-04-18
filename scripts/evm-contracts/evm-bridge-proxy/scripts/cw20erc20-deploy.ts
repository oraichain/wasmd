import { ethers, getSigners } from "hardhat";
import { CW20ERC20Token__factory } from "../typechain-types";

async function main() {
  const cw20Address = process.env.CW20_ADDRESS!;
  const [owner] = getSigners(1);
  console.log("cw20 address:", cw20Address);
  const deploy = await new CW20ERC20Token__factory(owner).deploy(
    cw20Address,
    "USDT token",
    "USDT"
  );
  const result = await deploy.deployed();
  console.log(result.address);
  const decimals = await result.decimals();
  console.log(decimals);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
