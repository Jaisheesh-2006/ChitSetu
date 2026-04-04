// License-Identifier: MIT
pragma solidity ^0.8.28;
import "./ChitFund.sol";
import "@openzeppelin/contracts/access/IAccessControl.sol";

contract ChitFundFactory {
      bytes32 public constant MANAGER_ROLE = keccak256("MANAGER_ROLE");

    struct FundInfo {
        address fundAddress;
        address creator;
        address token;
        string name;
        uint members;
        uint contribution;
    }

    FundInfo[] public allFunds;

    mapping(address => address[]) public userFunds;
    mapping(string => bool) public fundNameExists;

    event FundCreated(
        address indexed fund,
        address indexed creator,
        string name
    );

    function createFund(
        address token,
        uint members,
        uint contribution,
        string memory name
    ) external {

        require(members > 1, "Invalid members");
        require(contribution > 0, "Invalid contribution");
        require(!fundNameExists[name], "Name already used");

        fundNameExists[name] = true;

        ChitFund fund = new ChitFund(
            token,
            members,
            contribution,
            name,
            msg.sender
        );

        allFunds.push(FundInfo({
            fundAddress: address(fund),
            creator: msg.sender,
            token: token,
            name: name,
            members: members,
            contribution: contribution
        }));

        userFunds[msg.sender].push(address(fund));

        // Auto-grant MANAGER_ROLE to the new fund on the token
        try IAccessControl(token).grantRole(MANAGER_ROLE, address(fund)) {
            // Success
        } catch {
            // Logically, the factory must be an ADMIN on the token for this to work.
            // If it fails, we keep going (migration logic might handle it).
        }

        emit FundCreated(address(fund), msg.sender, name);
    }

    function getAllFunds() external view returns (FundInfo[] memory) {
        return allFunds;
    }

    function getUserFunds(address user) external view returns (address[] memory) {
        return userFunds[user];
    }

    function totalFunds() external view returns (uint) {
        return allFunds.length;
    }

    function getFund(uint index) external view returns (FundInfo memory) {
        return allFunds[index];
    }
}