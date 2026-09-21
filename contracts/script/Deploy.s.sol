// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console2} from "forge-std/Script.sol";
import {PulseTradeStamp} from "../src/PulseTradeStamp.sol";

contract Deploy is Script {
    function run() external {
        uint256 pk = vm.envUint("PRIVATE_KEY");
        uint256 expected = vm.envUint("EXPECTED_CHAIN_ID");
        require(block.chainid == expected, "wrong chain");

        vm.startBroadcast(pk);
        PulseTradeStamp c = new PulseTradeStamp();
        vm.stopBroadcast();

        console2.log("PulseTradeStamp", address(c));
        console2.log("chainId", block.chainid);
        console2.log("deployer", vm.addr(pk));
    }
}
