import { ethers } from "hardhat";

export const contracts = {
  proxyContract: ethers.utils.getAddress(
    "0x9a0A02B296240D2620E339cCDE386Ff612f07Be5"
  ),
  eth: {
    oraiAddr: ethers.utils.getAddress(
      "0x4c11249814f11b9346808179Cf06e71ac328c1b5"
    ),
    routerAddr: ethers.utils.getAddress(
      "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"
    ), // uniswap router / pancakeswap
    wrapNativeAddr: ethers.utils.getAddress(
      "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
    ),
    gravityBridgeContract: ethers.utils.getAddress(
      "0x09Beeedf51AA45718F46837C94712d89B157a9D3"
    ),
  },
  bnb: {
    oraiAddr: ethers.utils.getAddress(
      "0xA325Ad6D9c92B55A3Fc5aD7e412B1518F96441C0"
    ),
    airiAddr: ethers.utils.getAddress(
      "0x7e2A35C746F2f7C240B664F1Da4DD100141AE71F"
    ),
    routerAddr: ethers.utils.getAddress(
      "0x10ED43C718714eb63d5aA57B78B54704E256024E"
    ), // uniswap router / pancakeswap
    wrapNativeAddr: ethers.utils.getAddress(
      "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"
    ),
    gravityBridgeContract: ethers.utils.getAddress(
      "0xb40C364e70bbD98E8aaab707A41a52A2eAF5733f"
    ),
  },
};
