// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// Uncomment this line to use console.log
// import "hardhat/console.sol";

// This contract is used to test the state of the EVM, specifically ensuring the
// StateDB Set/GetState methods behave as expected.
contract StateTest {
    uint256 public value1;
    uint256 public value2 = 0x1234;

    function tempChangeEmpty(uint256 _value) public {
        // Start from an empty state
        require(value1 == 0, "Value is should be empty");
        require(value2 != 0, "New value should be non-empty");

        // Change to a non-empty state
        value1 = _value;

        // Revert back to an empty state to create an overall no-op change.
        value1 = 0;

        // Ensure value1 is still empty
        require(value1 == 0, "Value should be empty");
    }

    function tempChangeNonEmpty(uint256 _value) public {
        require(value2 != 0, "Existing value should be non-empty");
        require(_value != 0, "New value should be non-empty");

        // Change to an empty state
        value2 = 0;

        // Revert back to an non-empty state
        value2 = _value;

        // Ensure value2 is still non-empty
        require(value2 != 0, "Value is empty");
    }
}
