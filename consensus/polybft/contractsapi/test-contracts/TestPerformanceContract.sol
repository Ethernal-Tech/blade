// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// import "hardhat/// console.sol";
// import "@openzeppelin/contracts/utils/Strings.sol";

contract TestPerformanceContract {
    struct SignedBatch {
        uint256 batchID;
        uint256 counter; // any data, can be used to simulate batches without quorum (which will stay in storage)
        uint256 validatorID; // instead of addresss; easier to write tests
        bytes signature; // can be anything, sc will not check
    }

    struct ConfirmedBatch {
        uint256 batchID;
        uint256 bitmap;
        uint256 counter;
        bytes[] signatures;
    }

    // hash -> multisig / bls signatures
    mapping(bytes32 => bytes[]) private signatures;

    mapping(bytes32 => uint256) private bitmap;

    // hash -> validator ID (uint8) -> true/false
    mapping(bytes32 => mapping(uint256 => bool)) private hasVoted; // for resubmit

    uint256 private hashesCount;

    ConfirmedBatch[] private confirmedBatches;

    uint256 private quorumCnt;

    constructor(uint256 _quorumCnt) {
        quorumCnt = _quorumCnt;
    }

    function submitSignedBatch(SignedBatch calldata _signedBatch) external {
        // console.log("Batch has been submitted!");
        // console.log(Strings.toString(_signedBatch.validatorID));
        // console.log(Strings.toString(_signedBatch.batchID));
        // console.log(Strings.toString(_signedBatch.counter));

        uint256 _validatorID = _signedBatch.validatorID;
        uint256 _batchID = _signedBatch.batchID;
        uint256 _counter = _signedBatch.counter;

        bytes32 _sbHash = keccak256(abi.encodePacked(_batchID, _counter));

        // check if caller already voted for same hash
        if (hasVoted[_sbHash][_validatorID]) {
            return;
        }

        uint256 _numberOfVotes = signatures[_sbHash].length;

        if (_numberOfVotes == quorumCnt) {
            // console.log("Quorum is already reached");
            return;
        }

        hasVoted[_sbHash][_validatorID] = true;

        // increment hashes count if this hash is not seen before
        if (_numberOfVotes == 0) {
            // console.log("New hash has been created");
            hashesCount++;
        }

        signatures[_sbHash].push(_signedBatch.signature);
        unchecked {
            bitmap[_sbHash] = bitmap[_sbHash] | (1 << _validatorID);
        }

        // check if quorum reached (+1 is last vote)
        if (_numberOfVotes + 1 >= quorumCnt) {
            // console.log("Quorum has been reached!");

            confirmedBatches.push(
                ConfirmedBatch(
                    _batchID,
                    bitmap[_sbHash],
                    _counter,
                    signatures[_sbHash]
                )
            );

            // remove from storage but for that exactly hash
            delete signatures[_sbHash];
            delete bitmap[_sbHash];
        }
    }

    function getConfirmedBatches()
        external
        view
        returns (ConfirmedBatch[] memory)
    {
        ConfirmedBatch[] memory batches = new ConfirmedBatch[](
            confirmedBatches.length
        );

        for (uint i = 0; i < confirmedBatches.length; i++) {
            batches[i] = confirmedBatches[i];
        }

        return batches;
    }

    function getHashesCount() external view returns (uint256) {
        return hashesCount;
    }
}
