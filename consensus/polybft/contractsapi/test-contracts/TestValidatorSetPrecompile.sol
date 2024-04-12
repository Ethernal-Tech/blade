// SPDX-License-Identifier: MIT
// TestValidatorSetPrecompile.sol
// Contract which testst ValidatorSet precompile
pragma solidity ^0.8.0;

contract TestValidatorSetPrecompile {
    address constant VALIDATOR_SET_PRECOMPILE = 0x0000000000000000000000000000000000002040;
    uint256 constant VALIDATOR_SET_PRECOMPILE_GAS = 240000;

    mapping(address => bool) voteMap;
    address[] votes;

    modifier onlyValidator() {
        (bool callSuccess, bytes memory returnData) = VALIDATOR_SET_PRECOMPILE.staticcall{
            gas: VALIDATOR_SET_PRECOMPILE_GAS
        }(abi.encode(msg.sender));
        require(callSuccess && abi.decode(returnData, (bool)), "validator");
        _;
    }

    function inc() public onlyValidator {
        if (!voteMap[msg.sender]) {
            votes.push(msg.sender);
            voteMap[msg.sender] = true;
        }
    }

    function hasQuorum() public view returns (bool) {
        (bool callSuccess, bytes memory returnData) = VALIDATOR_SET_PRECOMPILE.staticcall{
            gas: VALIDATOR_SET_PRECOMPILE_GAS
        }(abi.encode(votes));
        return callSuccess && abi.decode(returnData, (bool));
    }
}
