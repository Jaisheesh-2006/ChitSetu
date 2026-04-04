import type { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const config: HardhatUserConfig = {
  solidity: "0.8.28",
  networks:{
    amoy:{
      url: process.env.AMOY_RPC_URL || "",
      accounts: process.env.AMOY_PRIVATE_KEY !== undefined ? [process.env.AMOY_PRIVATE_KEY] : [],
    },
    localhost:{
      url:"http://localhost:8545",
      accounts: process.env.LOCALHOST_PRIVATE_KEY !== undefined ? [process.env.LOCALHOST_PRIVATE_KEY] : [],
    }
  }
};

export default config;
