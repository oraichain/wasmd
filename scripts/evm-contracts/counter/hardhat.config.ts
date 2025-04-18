import "dotenv/config";
import "@nomiclabs/hardhat-ethers";
import "@openzeppelin/hardhat-upgrades";
import "@typechain/hardhat";
import "@nomiclabs/hardhat-etherscan";
import "hardhat-contract-sizer";
import { ethers } from "ethers";
import { extendEnvironment } from "hardhat/config";
import { HardhatUserConfig } from "hardhat/types";

let accounts: string[] = [];

if (process.env.MNEMONIC) {
  accounts = [];
  for (let i = 0; i < 20; ++i) {
    accounts.push(
      ethers.Wallet.fromMnemonic(process.env.MNEMONIC, `m/44'/60'/0'/${i}`)
        .privateKey
    );
  }
} else if (process.env.PRIVATE_KEY) {
  accounts = process.env.PRIVATE_KEY.split(/\s*,\s*/);
}

const config: HardhatUserConfig = {
  defaultNetwork: "hardhat",

  networks: {
    testing: {
      url: "http://0.0.0.0:8545",
      accounts,
    },
    hardhat: {
      chainId: 31337,
      accounts: accounts?.map((privateKey) => ({
        privateKey,
        balance: "1000000000000000000000000000000",
      })),
      forking: {
        url: "https://rpc.ankr.com/eth",
        blockNumber: 17914136,
      },
    },
    rinkeby: {
      url: `https://eth-rinkeby.alchemyapi.io/v2/${process.env.ALCHEMY_API_KEY}`,
      accounts,
    },
    eth: {
      url: `https://rpc.ankr.com/eth`,
      accounts,
    },
  },
  paths: {
    sources: "contracts",
  },
  etherscan: {
    // Your API key for Etherscan
    // Obtain one at https://etherscan.io/
    apiKey: process.env.ETHERSCAN_API_KEY,
  },
  solidity: {
    compilers: [
      {
        version: "0.8.16",
        settings: {
          optimizer: {
            enabled: true, // Default: false
            runs: 1000, // Default: 200
          },
        },
      },
    ],
  },
  mocha: {
    timeout: process.env.TIMEOUT || 100000,
  },
};

declare module "hardhat/types/runtime" {
  export interface HardhatRuntimeEnvironment {
    provider: ethers.providers.Web3Provider;
    getSigner: (
      addressOrIndex?: string | number
    ) => ethers.providers.JsonRpcSigner;
    getSigners: (num?: number) => ethers.providers.JsonRpcSigner[];
  }
}

extendEnvironment((hre) => {
  // @ts-ignore
  hre.provider = new ethers.providers.Web3Provider(hre.network.provider);
  hre.getSigners = (num = 20) =>
    [...new Array(num)].map((_, i) => hre.provider.getSigner(i));
  hre.getSigner = (addressOrIndex) => hre.provider.getSigner(addressOrIndex);
});

export default config;
