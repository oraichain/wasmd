// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IWasmd {
    // Transactions
    function instantiate(
        uint64 codeID,
        string memory admin,
        bytes memory payload,
        string memory label,
        bytes memory coins
    ) external returns (string memory contractAddr, bytes memory data);

    function execute(
        string memory contractAddress,
        bytes memory payload,
        bytes memory coins
    ) external returns (bytes memory response);

    // Queries
    function query(
        string memory contractAddress,
        bytes memory req
    ) external view returns (bytes memory response);
}