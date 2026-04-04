import {HardhartUserConfig} from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";
import "dotenv/config";
import "@nomicfoundation/hardhat-verify";

import * as dotenv from "dotenv";
import { solidity } from "hardhat";
import { url } from "node:inspector";

dotenv.config({path: '../.env'});

const config: HardhartUserConfig = {
  solidity: "0.8.18",

  networks:{
    amoy:{
      url: process.env.AMOY_RPC_URL || "",
      accounts:process.env.PRIVATE_KEY !== undefined ? [process.env.PRIVATE_KEY]:[],

    },
    localhost:{
      url:"http://127.0.0.1:8545",
      accounts:[
        process.env.PRIVATE_KEY !== undefined ? process.env.PRIVATE_KEY : "",
      ],
    },
  }

};
export default config;