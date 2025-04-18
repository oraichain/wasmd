import { ethers, getSigners } from "hardhat";

const main = async () => {
  const [sender] = getSigners(1);

  const tx = await sender.sendTransaction({
    to: "0x11E6017Def04B1D70eD10eB903392F504589b041",
    value: ethers.utils.parseUnits("10", "ether"),
  });

  const receipt = await tx.wait();
  console.log("receipt: ", receipt);
};

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.log(error);
    process.exit(1);
  });
