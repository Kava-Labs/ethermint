// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// This contract is used to test the behavior of EIP-2929, excluding SLOAD.
// Primarily for ensuring custom precompile addresses are added to the accesslist
// correctly. Refer to x/evm/keeper/state_transition_test.go for the corresponding
// test cases.
contract EIP2929Test {
    function callAccount(address target) public payable returns (bytes memory) {
        (bool success, bytes memory data) = target.call{value: msg.value}("");
        require(success, "Failed to call empty account with value");

        return data;
    }

    function getAccountBalance(address target) public view returns (uint256) {
        return target.balance;
    }

    function getAccountCodeSize(address target) public view returns (uint32) {
        uint32 size;
        assembly {
            size := extcodesize(target)
        }

        return size;
    }
}
