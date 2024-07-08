// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract EIP161Test {
    function selfDestructTo(address target) public payable {
        // This contract will self-destruct and send its balance to the target address
        selfdestruct(payable(target));
    }

    function selfDestructToRevert(address target) public payable {
        // This contract will self-destruct and send its balance to the target address
        selfdestruct(payable(target));

        revert();
    }

    function callAccount(address target) public payable returns (bytes memory) {
        (bool success, bytes memory data) = target.call{value: msg.value}("");
        require(success, "Failed to call empty account with value");

        return data;
    }

    function callAccountRevert(address target) public payable {
        callAccount(target);
        revert();
    }

    function createContract() public payable returns (address) {
        address newContract;
        bytes memory bytecode = hex"3859818153F3";
        assembly {
            newContract := create(0, add(bytecode, 0x20), mload(bytecode))
            if iszero(extcodesize(newContract)) {
                revert(0, 0)
            }
        }
        return newContract;
    }

    function transferValue(address target) public payable {
        payable(target).transfer(msg.value);
    }

    function transferValueRevert(address target) public payable {
        transferValue(target);
        revert();
    }
}
