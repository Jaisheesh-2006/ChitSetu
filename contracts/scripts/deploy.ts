import { ethers } from "hardhat";
import * as dotenv from "dotenv";
dotenv.config();
async function main() {
    console.log("Starting deployment...");

    const [deployer] = await ethers.getSigners();

    // 1. Deploy the ERC20 Token first
    const Token = await ethers.getContractFactory("EscrowToken");
    const token = await Token.deploy(deployer.address);
    await token.waitForDeployment();
    const tokenAddress = await token.getAddress();
    console.log("EscrowToken deployed to:", tokenAddress);

    // 2. Deploy the Factory
    const Factory = await ethers.getContractFactory("ChitFundFactory");
    const factory = await Factory.deploy();
    await factory.waitForDeployment();
    const factoryAddress = await factory.getAddress();
    console.log("ChitFundFactory deployed to:", factoryAddress);

    // 3. Setup Roles
    console.log("\nSetting up roles...");
    const DEFAULT_ADMIN_ROLE = await token.DEFAULT_ADMIN_ROLE();
    const MANAGER_ROLE = await token.MANAGER_ROLE();

    // Grant Factory the ability to grant MANAGER_ROLE to children
    await token.grantRole(DEFAULT_ADMIN_ROLE, factoryAddress);
    console.log("Granted DEFAULT_ADMIN_ROLE to Factory");

    // Grant Factory MANAGER_ROLE for itself if needed
    await token.grantRole(MANAGER_ROLE, factoryAddress);
    console.log("Granted MANAGER_ROLE to Factory");

    // Optional: Grant Deployer MANAGER_ROLE for manual testing
    await token.grantRole(MANAGER_ROLE, deployer.address);
    console.log("Granted MANAGER_ROLE to Deployer");

    console.log("\nIMPORTANT: Add these to your backend .env file:");
    console.log(`TOKEN_CONTRACT_ADDRESS=${tokenAddress}`);
    console.log(`FACTORY_CONTRACT_ADDRESS=${factoryAddress}`);
}

main().catch((error) => {
    console.error("Error:", error);
    process.exitCode = 1;
});