import { ethers, getSigners } from "hardhat";
import assert from "assert";
import {
  Bridge,
  Bridge__factory,
  IERC20Upgradeable,
  IERC20Upgradeable__factory,
  IUniswapV2Router02,
  IUniswapV2Router02__factory,
} from "../typechain-types";
import { contracts } from "../constants";

describe("Bridge", () => {
  const [owner] = getSigners(1);
  let ownerAddress: string;
  let uniswapRouter: IUniswapV2Router02;
  let bridge: Bridge;
  let oraiContract: IERC20Upgradeable;
  let wrapNativeContract: IERC20Upgradeable;
  const destination = "0x8754032Ac7966A909e2E753308dF56bb08DabD69";
  const { eth, bnb } = contracts;
  let { oraiAddr, routerAddr, wrapNativeAddr, gravityBridgeContract } = eth;
  const gravityInterface = new ethers.utils.Interface([
    "event SendToCosmosEvent(address indexed _tokenContract, address indexed _sender, string _destination, uint256 _amount, uint256 _eventNonce)",
  ]);
  const approveInterface = new ethers.utils.Interface([
    "event Approval(address indexed owner, address indexed spender, uint256 value);",
  ]);
  const transferInterface = new ethers.utils.Interface([
    "event Transfer(address indexed from, address indexed to, uint256 value)",
  ]);

  beforeEach(async function () {
    ownerAddress = await owner.getAddress();
    const chainId = await owner.getChainId();

    switch (chainId) {
      case 1: // eth
      case 31337: // fork eth
        break;
      case 56: // bnb
      case 5600: // fork bnb
        // bsc
        oraiAddr = bnb.oraiAddr;
        routerAddr = bnb.routerAddr;
        wrapNativeAddr = bnb.wrapNativeAddr;
        gravityBridgeContract = bnb.gravityBridgeContract;
        break;
      default:
        return;
    }

    bridge = await new Bridge__factory(owner).deploy(
      gravityBridgeContract,
      routerAddr,
      wrapNativeAddr
    );
    uniswapRouter = IUniswapV2Router02__factory.connect(routerAddr, owner);
    oraiContract = IERC20Upgradeable__factory.connect(oraiAddr, owner);
    wrapNativeContract = IERC20Upgradeable__factory.connect(
      wrapNativeAddr,
      owner
    );
  });

  it("retrieve returns contract addresses", async function () {
    // Store a value
    const contract = await bridge.gravityBridgeContract();
    const router = await bridge.swapRouter();
    const wrapped = await bridge.wrapNativeAddress();
    assert.strictEqual(contract, gravityBridgeContract);
    assert.strictEqual(router, routerAddr);
    assert.strictEqual(wrapped, wrapNativeAddr);
  });

  it("bridgeFromETH swap eth to orai", async function () {
    const res = await bridge.bridgeFromETH(oraiAddr, "1", "", {
      value: "1",
    });

    // use should have more than 0 orai
    const oraiBalance = await oraiContract.balanceOf(ownerAddress);
    assert.equal(oraiBalance.gt("0"), true);
    const { events } = await res.wait();

    // there should be no gravity contract event
    assert.equal(
      events?.some((e) => e.address === gravityBridgeContract),
      false
    );

    const erc20Events = events?.filter((e) => e.address === oraiAddr)!;
    // length is 2 because the first call is transferFrom when calling uniswap
    // 2nd one is transfer back to the sender's wallet
    assert.equal(erc20Events.length, 2);

    const transferLog = transferInterface.decodeEventLog(
      "Transfer",
      erc20Events[1].data,
      erc20Events[1].topics
    );
    assert.equal(transferLog[0], bridge.address);
    assert.equal(transferLog[1], ownerAddress);
    assert.equal(BigInt(transferLog[2]).toString(), "733");
  });

  it("bridgeFromETH swap eth to orai then send to cosmos", async function () {
    const res = await bridge.bridgeFromETH(oraiAddr, 1, destination, {
      value: "1",
    });
    const { events } = await res.wait();
    const bridgeEvent = events?.find(
      (e) => e.address === gravityBridgeContract
    )!;
    const eventLog = gravityInterface.decodeEventLog(
      "SendToCosmosEvent",
      bridgeEvent.data,
      bridgeEvent.topics
    );
    assert.strictEqual(destination, eventLog._destination);
    assert.strictEqual(bridge.address, eventLog._sender);
    // assert.equal(BigInt(eventLog[3]).toString(), "733");

    const erc20Events = events?.filter((e) => e.address === oraiAddr)!;
    // length is 3 because the first call is the swap call
    // 2nd one is to approve for the gravity bridge contract to transfer token from the proxy contract
    // last one is the actual transfer from token called by the gravity bridge contract
    // assert.equal(erc20Events.length, 3);

    const approveLog = approveInterface.decodeEventLog(
      "Approval",
      erc20Events[1].data,
      erc20Events[1].topics
    );
    assert.equal(approveLog[0], bridge.address);
    assert.equal(approveLog[1], gravityBridgeContract);

    const transferLog = transferInterface.decodeEventLog(
      "Transfer",
      erc20Events[2].data,
      erc20Events[2].topics
    );
    assert.equal(transferLog[0], bridge.address);
    assert.equal(transferLog[1], gravityBridgeContract);
    // magic number 733. At the fork height, 1 amount bnb / eth = 733 orai
    assert.equal(transferLog[2], BigInt(733));
  });

  it("bridgeFromETH swap eth to weth then send weth to cosmos", async function () {
    const res = await bridge.bridgeFromETH(wrapNativeAddr, 1000, destination, {
      value: "1",
    });
    const { events } = await res.wait();
    const bridgeEvent = events?.find(
      (e) => e.address === gravityBridgeContract
    )!;
    const eventLog = gravityInterface.decodeEventLog(
      "SendToCosmosEvent",
      bridgeEvent.data,
      bridgeEvent.topics
    );
    assert.strictEqual(destination, eventLog._destination);
    assert.strictEqual(bridge.address, eventLog._sender);
    // assert.equal(BigInt(eventLog[3]).toString(), "733");

    const erc20Events = events?.filter((e) => e.address === wrapNativeAddr)!;
    // length is 3 because the first call is the swap call
    // 2nd one is to approve for the gravity bridge contract to transfer token from the proxy contract
    // last one is the actual transfer from token called by the gravity bridge contract
    // assert.equal(erc20Events.length, 3);

    const approveLog = approveInterface.decodeEventLog(
      "Approval",
      erc20Events[1].data,
      erc20Events[1].topics
    );
    assert.equal(approveLog[0], bridge.address);
    assert.equal(approveLog[1], gravityBridgeContract);

    const transferLog = transferInterface.decodeEventLog(
      "Transfer",
      erc20Events[2].data,
      erc20Events[2].topics
    );
    assert.equal(transferLog[0], bridge.address);
    assert.equal(transferLog[1], gravityBridgeContract);
    assert.equal(transferLog[2], BigInt(1));
  });

  it("bridgeFromETH swap eth to weth", async function () {
    // use should have more than 0 orai
    let wrapNativeBalance = await wrapNativeContract.balanceOf(ownerAddress);
    assert.equal(wrapNativeBalance.eq("0"), true);

    const res = await bridge.bridgeFromETH(wrapNativeAddr, "10000", "", {
      value: "1",
    });
    const { events } = await res.wait();

    // there should be no gravity contract event
    assert.equal(
      events?.some((e) => e.address === gravityBridgeContract),
      false
    );

    const erc20Events = events?.filter((e) => e.address === wrapNativeAddr)!;
    // length is 2 because the first call is transferFrom when calling uniswap
    // 2nd one is transfer back to the sender's wallet
    assert.equal(erc20Events.length, 2);

    const transferLog = transferInterface.decodeEventLog(
      "Transfer",
      erc20Events[1].data,
      erc20Events[1].topics
    );
    assert.equal(transferLog[0], bridge.address);
    assert.equal(transferLog[1], ownerAddress);
    assert.equal(transferLog[2], BigInt(1));

    wrapNativeBalance = await wrapNativeContract.balanceOf(ownerAddress);
    assert.equal(wrapNativeBalance.eq("1"), true);
  });

  it("bridgeFromERC20 swap eth to orai then swap orai to weth", async function () {
    const oraiAmount = 1000;
    const nativeEthAmount = "1000000000";
    await bridge.bridgeFromETH(oraiAddr, oraiAmount, "", {
      value: nativeEthAmount,
    });
    await oraiContract.approve(bridge.address, oraiAmount);
    const res = await bridge.bridgeFromERC20(
      oraiAddr,
      wrapNativeAddr,
      oraiAmount,
      1,
      ""
    );
    const { events } = await res.wait();
    // there should be no gravity contract event
    assert.equal(
      events?.some((e) => e.address === gravityBridgeContract),
      false
    );
    let erc20Events = events?.filter((e) => e.address === wrapNativeAddr)!;
    // length is 2 because token out is weth, which has only two calls
    // 1st one when calling uniswap
    // last one is transfer back to the sender's wallet
    assert.equal(erc20Events.length, 2);

    const transferLog = transferInterface.decodeEventLog(
      "Transfer",
      erc20Events[1].data,
      erc20Events[1].topics
    );
    assert.equal(transferLog[0], bridge.address);
    assert.equal(transferLog[1], ownerAddress);
    // assert.equal(BigInt(transferLog[2]).toString(), "1");
  });

  it("bridgeFromERC20 swap eth to orai then swap orai to weth then send to cosmos", async function () {
    let oraiAmount = 1000;
    const nativeEthAmount = "1000000000";
    await bridge.bridgeFromETH(oraiAddr, oraiAmount, "", {
      value: nativeEthAmount,
    });
    await oraiContract.approve(bridge.address, oraiAmount);
    await bridge.sendToCosmos(oraiAddr, destination, 1);
    oraiAmount -= 1;

    const res = await bridge.bridgeFromERC20(
      oraiAddr,
      wrapNativeAddr,
      oraiAmount,
      1,
      destination
    );
    const { events } = await res.wait();
    const bridgeEvent = events?.find(
      (e) => e.address === gravityBridgeContract
    )!;
    const eventLog = gravityInterface.decodeEventLog(
      "SendToCosmosEvent",
      bridgeEvent.data,
      bridgeEvent.topics
    );
    assert.strictEqual(destination, eventLog._destination);
    assert.strictEqual(bridge.address, eventLog._sender);
    // assert.equal(BigInt(eventLog[3]).toString(), "1");

    const erc20Events = events?.filter((e) => e.address === wrapNativeAddr)!;
    // length is 3 because token out is weth, the first call is the swap call
    // 2nd is to approve for the gravity bridge contract to transfer token from the proxy contract
    // last one is the actual transfer from token called by the gravity bridge contract
    assert.equal(erc20Events.length, 3);

    const approveLog = approveInterface.decodeEventLog(
      "Approval",
      erc20Events[1].data,
      erc20Events[1].topics
    );
    assert.equal(approveLog[0], bridge.address);
    assert.equal(approveLog[1], gravityBridgeContract);

    const transferLog = transferInterface.decodeEventLog(
      "Transfer",
      erc20Events[2].data,
      erc20Events[2].topics
    );
    assert.equal(transferLog[0], bridge.address);
    assert.equal(transferLog[1], gravityBridgeContract);
    // assert.equal(BigInt(transferLog[2]).toString(), "1");
  });

  async function simulateRoute(
    route: string[],
    simulateAmount: string
  ): Promise<string> {
    const swapRouterV2 = IUniswapV2Router02__factory.connect(routerAddr, owner);
    try {
      let outs = await swapRouterV2.getAmountsOut(simulateAmount, route);
      console.log("route out amount: ", route, outs.slice(-1)[0]);
      return outs.slice(-1)[0].toString();
    } catch (error) {
      console.log("error simulating: ", error);
      return "0";
    }
  }

  it("simulate-bnb-swap-orai-to-airi", async () => {
    const chainId = await owner.getChainId();
    if (chainId === 56) {
      const simulateAmount = BigInt(1 * 10 ** 18).toString();
      assert.equal(
        await simulateRoute(
          [contracts.bnb.airiAddr, wrapNativeAddr],
          simulateAmount
        ),
        "11522495640377"
      );
      assert.equal(
        await simulateRoute([oraiAddr, wrapNativeAddr], simulateAmount),
        "9683120229072611"
      );
      assert.equal(
        await simulateRoute(
          [oraiAddr, wrapNativeAddr, contracts.bnb.airiAddr],
          simulateAmount
        ),
        "836163475812578803334"
      );
    } else {
      // dont care. this test is for bsc
    }
  });
});
