// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

//import "hardhat/console.sol";
//import "@openzeppelin/contracts/utils/Strings.sol";

contract TestPerformanceContract {
    struct SignedBatch {
        uint256 batchID;
        uint256 validatorID; // instead of addresss; easier to write tests
        bytes rawTx;
        bytes signature; // can be anything, sc will not check
    }

    struct ConfirmedBatch {
        uint256 batchID;
        uint256 bitmap;
        bytes rawTx;
        bytes[] signatures;
    }

    // hash -> multisig / bls signatures
    mapping(bytes32 => bytes[]) private signatures;

    // hash -> validator ID (uint8) -> true/false
    mapping(bytes32 => uint256) private bitmap;

    uint256 private hashesCount;

    ConfirmedBatch private lastConfirmedBatch;

    uint256 private lastBatchID;

    uint256 private quorumCnt;

    bool private checkBatchID;

    bool private deleteTemporaryMappingsAfterQuorum;

    constructor(uint256 _quorumCnt, bool _checkBatchID, bool _deleteTemporaryMappingsAfterQuorum) {
        quorumCnt = _quorumCnt;
        checkBatchID = _checkBatchID;
        deleteTemporaryMappingsAfterQuorum = _deleteTemporaryMappingsAfterQuorum;
    }

    function submitSignedBatch(SignedBatch calldata _signedBatch) external {
        //console.log("Batch has been submitted!");
        //console.log(Strings.toString(_signedBatch.validatorID));
        //console.log(Strings.toString(_signedBatch.batchID));

        uint256 _batchID = _signedBatch.batchID;

        if (checkBatchID && lastBatchID + 1 != _batchID) {
            //console.log("Invalid batch ID!");
            //console.log(Strings.toString(lastBatchID + 1));
            return;
        }

        bytes32 _sbHash = keccak256(abi.encodePacked(_batchID, _signedBatch.rawTx));
        uint256 _oldBitmapVal = bitmap[_sbHash];
        uint256 _newBitmapVal;
        unchecked {
            _newBitmapVal = _oldBitmapVal | (1 << _signedBatch.validatorID);
        }

        // check if caller already voted for same hash
        if (_newBitmapVal == _oldBitmapVal) {
            //console.log("Validator already voted!");
            return;
        }

        uint256 _numberOfVotes = signatures[_sbHash].length;

        if (_numberOfVotes == quorumCnt) {
            // console.log("Quorum is already reached");
            return;
        }

        // increment hashes count if this hash is not seen before
        if (_numberOfVotes == 0) {
            //console.log("New hash has been created");
            hashesCount++;
        }

        signatures[_sbHash].push(_signedBatch.signature);
        bitmap[_sbHash] = _newBitmapVal;

        // check if quorum reached (+1 is last vote)
        if (_numberOfVotes + 1 >= quorumCnt) {
            //console.log("Quorum has been reached!");
            lastConfirmedBatch = ConfirmedBatch(_batchID, bitmap[_sbHash], _signedBatch.rawTx, signatures[_sbHash]);
            //console.log("ConfirmedBatch has been updated!");
            if (lastBatchID < _batchID) {
                lastBatchID = _batchID;
            }

            if (deleteTemporaryMappingsAfterQuorum) {
                // remove from storage but for that exactly hash
                for (uint256 i = 0; i < signatures[_sbHash].length; ++i) {
                    delete signatures[_sbHash][i];
                }
                delete signatures[_sbHash];
                delete bitmap[_sbHash];
            }
        }
        //console.log("Finished!");
    }

    function getConfirmedBatch() external view returns (ConfirmedBatch memory) {
        return lastConfirmedBatch;
    }

    function getHashesCount() external view returns (uint256) {
        return hashesCount;
    }

    function getLastBatchID() external view returns (uint256) {
        return lastBatchID;
    }
}
