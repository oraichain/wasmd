import { getSigners } from "hardhat";

// mainnet eth & bsc

async function main() {
  const [owner] = getSigners(1);
  const ownerBalance = await owner.getBalance();
  console.log("owner balance: ", ownerBalance.toString());
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
