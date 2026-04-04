import type { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const config: HardhatUserConfig = {
  solidity: "0.8.28",
  networks:{
    amoy:{
      url: process.env.WEB3_RPC_URL || "",
      accounts: process.env.WEB3_PRIVATE_KEY !== undefined ? [process.env.WEB3_PRIVATE_KEY] : [],
    },
    localhost:{
      url:"http://localhost:8545",
      accounts: process.env.LOCALHOST_PRIVATE_KEY !== undefined ? [process.env.LOCALHOST_PRIVATE_KEY] : [],
    }
  }
};

export default config;
