// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IAuthz {
    function setGrant(
        address grantee,
        string memory denom,
        uint256 amount
    ) external returns (bool success);

    function execGrant(
        address granter,
        address recipient,
        string memory denom,
        uint256 amount
    ) external returns (bool success);

    function grant(
        address granter,
        address grantee,
        string memory denom
    ) external view returns (uint256 amount);
}
