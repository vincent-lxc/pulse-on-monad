# Monad Metropolis — Track 01 pitch

**Pulse on Monad** is an auditable off-chain trading agent that stamps
immutable **decision receipts** on Monad.

## Track

**Onchain Finance & Trading** (Metropolis Track 01).

The product is a **policy loop + receipt**, not a CLOB:

- Rules propose `buy` / `sell` / `hold` from K-lines (optional Kronos features).
- **Hard risk is code** and fail-closed. Feedback cannot loosen it.
- Optional Jev is a *soft* gate only (prompt + stub if no API key).
- After a risk pass, `PulseTradeStamp.stamp` writes `decisionHash`, symbol,
  size hint, action, and a note that includes `agentId`.
- Outcomes (PnL or gate-block) label the run and update **rule weights**.

That is the closed loop judges can replay from JSONL + explorer events.

## Why Monad

- Already deployed on **Monad testnet** (chainId `10143`):
  [`0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4`](https://testnet.monadvision.com/address/0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4)
- Cheap, fast receipts make “stamp every decision” practical.
- ERC-8004 Identity Registry is live on the same chain
  (`0x8004A818BFB912233c491871b3d84c89A494BD9e`) as an **optional identity
  shell** — register an `agentId`, point `agentURI` at the agent card, keep
  `agentWallet` for settlement. Pulse does **not** require a live mint to
  stamp or to pass CI.

## What we are not claiming

- Not a full perp / CLOB engine.
- Not live funded DEX trading.
- Kronos and Jev are **not** online-trained here. Rules adapt; models do not.

## Demo path (under 3 minutes)

```bash
cd contracts && forge test
cd ../agent && go test ./... && go run ./cmd/pulse demo
# live (needs PRIVATE_KEY + testnet MON; never commit the key):
# go run ./cmd/pulse demo --live
```

Dry-run prints calldata. `--live` / `PULSE_STAMP_LIVE=1` broadcasts
`stamp()` and prints the [MonadVision](https://testnet.monadvision.com) tx
URL plus a `getReceipt` hash match. Compare `data/decisions.jsonl`
`decision_hash` to the on-chain receipt.

## Fallback narrative

If a judge asks about AI infra rather than trading: the same receipt is
**agent decision provenance** — a keccak of the audit record, on-chain,
with an optional ERC-8004 handle.
