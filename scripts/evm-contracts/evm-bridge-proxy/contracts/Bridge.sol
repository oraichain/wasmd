// SPDX-License-Identifier: BSD-3-Clause
pragma solidity 0.8.16;

import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/math/SafeMathUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/ERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@uniswap/lib/contracts/libraries/TransferHelper.sol";
import "@uniswap/v2-periphery/contracts/interfaces/IUniswapV2Router02.sol";
import "./IGravity.sol";

contract Bridge is Initializable, OwnableUpgradeable {
    event Amount(string description, uint amount);
    event Address(string description, address addy);

    address public gravityBridgeContract;
    address public swapRouter; // swap router, can be uniswap / pancakeswap. Eg: uniswap v2: 0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D
    address public wrapNativeAddress; // erc20 version of the native token on the deployed chain. Eg, WETH is 0xc778417E063141139Fce010982780140Aa0cD5Ab

    constructor(
        address _gravityBridgeContract,
        address _swapRouter,
        address _wrapNativeAddress
    ) {
        gravityBridgeContract = _gravityBridgeContract;
        swapRouter = _swapRouter;
        wrapNativeAddress = _wrapNativeAddress;
        initialize();
    }

    function initialize() public initializer {
        __Ownable_init_unchained();
    }

    function sendToCosmosInternal(
        address _tokenContract,
        string calldata _destination,
        uint256 _amount
    ) private {
        TransferHelper.safeApprove(
            _tokenContract,
            gravityBridgeContract,
            _amount
        );
        IGravity(gravityBridgeContract).sendToCosmos(
            _tokenContract,
            _destination,
            _amount
        );
    }

    function sendToCosmos(
        address _tokenContract,
        string calldata _destination,
        uint256 _amount
    ) external {
        TransferHelper.safeTransferFrom(
            _tokenContract,
            msg.sender,
            address(this),
            _amount
        );
        sendToCosmosInternal(_tokenContract, _destination, _amount);
    }

    function bridgeFromETH(
        address _tokenOut,
        uint _amountOutMin,
        string calldata _destination
    ) external payable {
        AddressUpgradeable.sendValue(payable(wrapNativeAddress), msg.value); //wrap ETH as WETH (comes back automatically)
        // case 1: token out = wrapNativeAddress -> we don't swap anything, keep the same msg.value sent as it maps 1-1 with the native coin.
        uint bridgeAmount = msg.value;
        // case 2: token out is not wrap native -> we swap and update amount out
        if (_tokenOut != wrapNativeAddress) {
            bridgeAmount = swap(
            wrapNativeAddress,
            _tokenOut,
            msg.value,
            _amountOutMin
        );
        }
        if (bytes(_destination).length == 0) {
            TransferHelper.safeTransfer(_tokenOut, msg.sender, bridgeAmount);
            return;
        }

        sendToCosmosInternal(_tokenOut, _destination, bridgeAmount);
    }

    function bridgeFromERC20(
        address _tokenIn,
        address _tokenOut,
        uint _amountIn,
        uint _amountOutMin,
        string calldata _destination
    ) external {
        TransferHelper.safeTransferFrom(
            _tokenIn,
            msg.sender,
            address(this),
            _amountIn
        );
        uint amountToReturn = swap(
            _tokenIn,
            _tokenOut,
            _amountIn,
            _amountOutMin
        );
        if (bytes(_destination).length == 0) {
            TransferHelper.safeTransfer(_tokenOut, msg.sender, amountToReturn);
            return;
        }

        sendToCosmosInternal(_tokenOut, _destination, amountToReturn);
    }

    function swap(
        address _tokenIn,
        address _tokenOut,
        uint _amountIn,
        uint _amountOutMin
    ) private returns (uint amountOfTokenOut) {
        TransferHelper.safeApprove(_tokenIn, swapRouter, _amountIn);

        address[] memory path;
        if (_tokenIn == wrapNativeAddress || _tokenOut == wrapNativeAddress) {
            path = new address[](2);
            path[0] = _tokenIn;
            path[1] = _tokenOut;
        } else {
            path = new address[](3);
            path[0] = _tokenIn;
            path[1] = wrapNativeAddress;
            path[2] = _tokenOut;
        }

        uint[] memory amounts = IUniswapV2Router02(swapRouter)
            .swapExactTokensForTokens(
                _amountIn,
                _amountOutMin,
                path,
                address(this),
                block.timestamp
            );

        uint amountToReturn = amounts[amounts.length - 1]; // if multiple hops were involved, the final amount will be in the last element

        return amountToReturn;
    }

    function sendTokenBalanceToOwner(address tokenToWithdraw) external {
        ERC20Upgradeable tokenAsContract = ERC20Upgradeable(tokenToWithdraw);
        TransferHelper.safeApprove(
            tokenToWithdraw,
            address(this),
            tokenAsContract.balanceOf(address(this))
        );
        TransferHelper.safeTransferFrom(
            tokenToWithdraw,
            address(this),
            payable(owner()),
            tokenAsContract.balanceOf(address(this))
        );
    }
}
