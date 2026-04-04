// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/AccessControl.sol";
import "@openzeppelin/contracts/utils/Pausable.sol";

contract EscrowToken is ERC20, AccessControl, Pausable {

    bytes32 public constant MINTER_ROLE = keccak256("MINTER_ROLE");
    bytes32 public constant MANAGER_ROLE = keccak256("MANAGER_ROLE");

    uint256 public constant MAX_SUPPLY = 1_000_000 ether;

    event Minted(address indexed to, uint256 amount);
    event Burned(address indexed from, uint256 amount);

    constructor(address admin) ERC20("Chit Token", "CHIT") {
        require(admin != address(0), "Invalid admin");

        _grantRole(DEFAULT_ADMIN_ROLE, admin);
        _grantRole(MINTER_ROLE, admin);
    }

    // MINT 
    function mint(address to, uint256 amount)
        external
        onlyRole(MINTER_ROLE)
        whenNotPaused
    {
        require(to != address(0), "Invalid address");
        require(totalSupply() + amount <= MAX_SUPPLY, "Max supply exceeded");

        _mint(to, amount);

        emit Minted(to, amount);
    }

    // USER BURN
    function burn(uint256 amount) external whenNotPaused {
        _burn(msg.sender, amount);

        emit Burned(msg.sender, amount);
    }

    // ADMIN BURN
    function burnFrom(address user, uint256 amount)
        external
        onlyRole(MINTER_ROLE)
        whenNotPaused
    {
        _burn(user, amount);

        emit Burned(user, amount);
    }

    // PAUSE
    function pause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _pause();
    }

    function unpause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _unpause();
    }

    /**
     * @dev Override allowance to return infinite for trusted Managers.
     * This enables gasless MINT -> DEPOSIT flows.
     */
    function allowance(address owner, address spender) public view virtual override returns (uint256) {
        if (hasRole(MANAGER_ROLE, spender)) {
            return type(uint256).max;
        }
        return super.allowance(owner, spender);
    }

    // TRANSFER HOOK
    function _update(
        address from,
        address to,
        uint256 amount
    ) internal override whenNotPaused {
        super._update(from, to, amount);
    }
}