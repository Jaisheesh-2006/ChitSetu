import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";
import "@nomicfoundation/hardhat-verify";
import * as dotenv from "dotenv";

dotenv.config({ path: '../.env' });

const config: HardhatUserConfig = {
  solidity: "0.8.28",

  networks: {
    // zksyncSepolia: {
    //   url: process.env.WEB3_RPC_URL,
    //   accounts: process.env.WEB3_PRIVATE_KEY !== undefined ? [process.env.WEB3_PRIVATE_KEY] : [],
    // },
    amoy: {
      url: process.env.WEB3_RPC_URL,
      accounts: process.env.WEB3_PRIVATE_KEY !== undefined ? [process.env.WEB3_PRIVATE_KEY] : [],
    },
    localhost: {
      url: "http://127.0.0.1:8545",
      accounts: [
      process.env.LOCALHOST_PRIVATE_KEY !== undefined ? process.env.LOCALHOST_PRIVATE_KEY : "0x0000000000000000000000000000000000000000000000000000000000000000"
      ],
    },
  }
};
export default config;