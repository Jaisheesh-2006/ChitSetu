// License-Identifier: MIT
pragma solidity ^0.8.28;


import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";


contract ChitFund {
    using SafeERC20 for IERC20;

    IERC20 public token;

    uint public totalMembers;
    uint public contributionAmount;
    uint public currentRound;
    uint public totalPool;
    // uint public reservePool; // Removed logic for optional dividends
    // bool public dividendEnabled; // Removed logic for optional dividends

    address public manager;
    string public fundName;

    address[] public members;

    mapping(address => bool) public joined;
    mapping(address => bool) public hasWon;
    mapping(address => bool) public hasPaid;
    mapping(uint => address) public winners;

    event MemberJoined(address indexed member);
    event ContributionPaid(address indexed member);
    event BidPlaced(address indexed bidder, uint amount);
    event AuctionWinner(uint round, address winner, uint payout, uint discount);
    event FundCompleted();

    modifier onlyManager() {
        require(msg.sender == manager, "Only manager");
        _;
    }

    constructor(
        address _token,
        uint _members,
        uint _contribution,
        string memory _name,
        address _creator
    ) {
        require(_members > 1, "Minimum 2 members");
        require(_contribution > 0, "Invalid contribution");

        token = IERC20(_token);
        totalMembers = _members;
        contributionAmount = _contribution;
        manager = _creator;
        fundName = _name;
    }

    // JOIN
    function joinFund(address member) external onlyManager {
        require(!joined[member], "Already joined");
        require(members.length < totalMembers, "Fund full");

        joined[member] = true;
        members.push(member);

        emit MemberJoined(member);
    }

    // CONTRIBUTION
    function depositContribution(address member) external onlyManager {
        require(joined[member], "Not member");
        require(!hasPaid[member], "Already paid");

        // The manager (backend) triggers the token transfer from the member's wallet
        token.safeTransferFrom(
            member, // <--- PULL FROM THE MEMBER
            address(this),
            contributionAmount
        );

        hasPaid[member] = true;
        totalPool += contributionAmount;

        emit ContributionPaid(member);
    }

    uint public lastAuctionTimestamp;

    /**
     * @dev Finalizes the current auction/round by rewarding the winner and distributing dividends.
     * Only the backend manager can call this once the off-chain bidding timer ends.
     */
    function finalizeAuction(address winner, uint256 discount) external onlyManager {
        require(members.length == totalMembers, "Fund not full");
        require(currentRound < totalMembers, "All rounds done");
        require(
            block.timestamp >= lastAuctionTimestamp + 20 hours, 
            "Cooldown: One auction per 20 hours"
        );
        require(
            totalPool == totalMembers * contributionAmount,
            "Incomplete contributions"
        );
        require(winner != address(0), "No eligible winner found");
        require(joined[winner], "Winner must be a member");
        require(hasPaid[winner], "Winner has not paid");
        require(!hasWon[winner], "Winner already won a previous round");
        require(discount <= totalPool, "Discount exceeds pool");

        uint payout = totalPool - discount;
        
        winners[currentRound] = winner;
        hasWon[winner] = true;
        lastAuctionTimestamp = block.timestamp;

        // Transfer payout
        token.safeTransfer(winner, payout);

        // Mandatory dividend distribution
        if (discount > 0 && totalMembers > 1) {
            uint numMembersToPay = totalMembers - 1;
            uint dividend = discount / numMembersToPay;
            uint remainder = discount % numMembersToPay;

            for (uint i = 0; i < members.length; i++) {
                address m = members[i];
                if (m != winner) {
                    token.safeTransfer(m, dividend);
                }
            }

            // Dust Handler: Send remainder (dust) to manager to keep contract balance clean
            if (remainder > 0) {
                token.safeTransfer(manager, remainder);
            }
        }

        emit AuctionWinner(currentRound, winner, payout, discount);

        currentRound++;

        // Reset for next round
        totalPool = 0;

        for (uint i = 0; i < members.length; i++) {
            address m = members[i];
            hasPaid[m] = false;
        }

        // Final completion
        if (currentRound == totalMembers) {
            emit FundCompleted();
        }
    }

    function getMembers() external view returns (address[] memory) {
        return members;
    }
}