# PulseTradeStamp (Foundry)

Immutable decision receipts for the Pulse agent on Monad.

```solidity
function stamp(
    bytes32 decisionHash,
    uint256 symbolId,
    uint256 sizeHint,
    uint8 action,          // 0 hold / 1 buy / 2 sell
    string calldata note   // include agentId; max 280 bytes
) external returns (uint256 id);
```

`decisionHash` is keccak256 of the agent's canonical audit record (see `docs/audit-schema.md`).

## Testnet

See [DEPLOYED-TESTNET.md](./DEPLOYED-TESTNET.md).

```
chainId            10143
PulseTradeStamp    0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4
```

## Commands

```bash
forge test
forge script script/Deploy.s.sol:Deploy --rpc-url monad_testnet --broadcast
```

`PRIVATE_KEY` and `EXPECTED_CHAIN_ID` are read from the environment. Do not commit `.env` or `broadcast/` folders that contain keys.
