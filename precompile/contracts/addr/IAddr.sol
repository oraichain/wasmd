// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IAddr {
    // Queries
    function getCosmosAddr(
        address addr
    ) external view returns (string memory response);

    function getEvmAddr(
        string memory addr
    ) external view returns (address response);

    function associate(
        string memory v,
        string memory r,
        string memory s,
        string memory customMessage
    ) external returns (string memory cosmosAddr, address evmAddr);

    function associatePubKey(
        string memory pubKeyHex
    ) external returns (string memory cosmosAddr, address evmAddr);
}
