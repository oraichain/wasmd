
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

address constant BANK_PRECOMPILE_ADDRESS = 0x9000000000000000000000000000000000000004;

IBank constant BANK_CONTRACT = IBank(
    BANK_PRECOMPILE_ADDRESS
);

interface IBank {
    // Transactions
    function send(
        address toAddress,
        string memory denom,
        uint256 amount
    ) external returns (bool success);

    // Queries
    function balance(
        address acc,
        string memory denom
    ) external view returns (uint256 amount);

    function name(
        string memory denom
    ) external view returns (string memory response);

    function symbol(
        string memory denom
    ) external view returns (string memory response);

    function decimals(
        string memory denom
    ) external view returns (uint8 response);

    function supply(
        string memory denom
    ) external view returns (uint256 response);
}
