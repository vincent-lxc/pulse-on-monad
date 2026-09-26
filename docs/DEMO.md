# Metropolis demo — Pulse on Monad

> 中文：这是给评委的英文演示稿，全程约 3 分钟。先在 Monad 测试网打一张回执，再在 receipts 页面打开同一笔交易。文中只有环境变量**名称**，不要写入私钥、助记词或 API key，录屏时也不要露出这些值。

Judges and the builder can follow this path in **3:00**. Default path: hard risk **PASS**, Jev **stub PASS** (no API key), live `stamp`, on-chain **MATCH**, then the receipts page. A real TypeSafe Jev call is optional.

## Shot list (3:00)

| Time | Shot | On screen |
| --- | --- | --- |
| 0:00–0:20 | One-liner | README title, or a title card |
| 0:20–0:30 | Architecture | README **Pipeline** diagram (ASCII or rendered mermaid) |
| 0:30–1:45 | Terminal stamp | `cd agent && go run ./cmd/pulse demo --live` |
| 1:45–2:25 | Receipts page | `make receipts` (or the Python server), then `/?id=N` |
| 2:25–2:40 | Coordinates | Contract, chain id, repo URL |
| 2:40–3:00 | Close | Out of scope, then why the receipt exists |

### 0:00–0:20 — One-liner

Pulse on Monad = **rules → hard risk (Go) → optional TypeSafe Jev soft gate → immutable PulseTradeStamp receipt on Monad testnet**.

The chain stores a receipt of the decision (hash, symbol, size hint, action, note). Settlement still happens off this demo.

### 0:20–0:30 — Architecture

Hold the README **Pipeline** section. Point along the loop: K-lines → rule candidates → hard risk (code only) → optional Jev → sim fill → audit JSONL → `decisionHash` → `PulseTradeStamp.stamp` on Monad. Ten seconds is enough. Do not redraw it.

### 0:30–1:45 — Terminal

From the repo root, with `.env` already loaded (see checklist; do this **off camera** so values never hit the shell scrollback you record):

```bash
cd agent && go run ./cmd/pulse demo --live
```

Scroll to these lines and leave them up:

```text
4. Hard risk  PASS  …
5. Jev        stub  PASS  jev_disabled_stub
8. Stamp      LIVE  to=0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4
              tx=0x…
              explorer=https://testnet.monadvision.com/tx/0x…
              receipt_id=N  on-chain decisionHash MATCH
```

- **PASS** on line 4 is the Go hard-risk gate. It runs before Jev.
- Line 5 is the soft gate. With no key, source is `stub` and the reason is `jev_disabled_stub`. That is still a recorded PASS.
- **`--jev` needs `TYPESAFE_API_KEY`** (alias `JEV_API_KEY`). The process exits if the key is missing; it will not stub. Optional on-camera variant, only when the key is already in the environment and **not** visible: `go run ./cmd/pulse demo --live --jev`. Expect `typesafe` instead of `stub` on line 5. Do not print, paste, or type the key.
- Line 8 is the broadcast. **MATCH** means `getReceipt(id).decisionHash` equals the audit keccak. Read `receipt_id=N` out loud; the browser step uses that id.
- `--live` spends **testnet MON**. One `stamp` call.

`make demo-live` from the repo root is the same command.

### 1:45–2:25 — Receipts page

Second terminal, repo root (the stamp terminal stays on **MATCH**):

```bash
make receipts
```

Equivalent:

```bash
cd web && python3 -m http.server 8080
```

Open `http://127.0.0.1:8080`. The new receipt is the latest row (symbol id `1` is labeled **BTC**; the chain stores the numeric id plus `note`).

Open the id from the CLI:

`http://127.0.0.1:8080/?id=N`

Click the explorer transaction link. It should be the same `tx=0x…` MonadVision page the CLI printed. No private key is used by this page.

### 2:25–2:40 — Coordinates

Leave these on screen (README **Deployed** table is fine):

| | |
| --- | --- |
| PulseTradeStamp | `0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4` |
| chainId | `10143` (Monad testnet) |
| Repo | https://github.com/vincent-lxc/pulse-on-monad |
| Explorer | https://testnet.monadvision.com/address/0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4 |

### 2:40–3:00 — Close

Out of scope for this demo: a CLOB, and live DEX execution. Kronos and Jev are not trained online in this repo.

Receipts matter for audit and settlement: the append-only JSONL line can be hashed again and checked against `getReceipt`. A third party does not have to trust the agent log by itself. Pitch and non-claims: [metropolis.md](metropolis.md).

## Spoken narration

Read this against the shot list. About 320 words, roughly three minutes at a calm pace.

Pulse on Monad is rules, then hard risk in Go, then an optional TypeSafe Jev soft gate, then an immutable PulseTradeStamp receipt on Monad testnet. The chain stores a receipt of that decision: the hash, the symbol, a size hint, the action, and a short note.

This diagram is the product. Quotes become rule candidates. Hard risk is ordinary Go code, and it always runs first. Feedback may keep those limits or tighten them. Jev is a soft gate after that, and it cannot loosen the code limits. The audit line is hashed. Stamp writes the decision hash on Monad testnet.

From the agent directory I run the live demo. The quote is a mock Bitcoin candle. Hard risk prints PASS. Jev prints PASS as a stub when no API key is set. The jev flag refuses that stub. It needs TYPESAFE_API_KEY already in the environment. The key stays off screen. The CLI then broadcasts stamp, prints the transaction hash and the explorer URL, and reads the receipt back. MATCH means the on-chain decision hash equals the audit keccak. This receipt id is the one we open next.

In a second terminal, from the repo root, I run make receipts. The same page is Python’s HTTP server in the web directory, on port 8080. The newest row is this stamp. Symbol id 1 is labeled BTC. Opening the receipt by id shows the action and the decision hash. The transaction link is MonadVision for the same hash.

The contract is 0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4, chain id 10143. The repository is github.com/vincent-lxc/pulse-on-monad.

Out of scope: a central limit order book, and live DEX execution. Receipts are the audit and settlement hook. Anyone can recompute the hash from the JSONL line and check the chain.

## Checklist

Do the prep **before** you start the timer. On camera, run only the two commands in the shot list.

- [ ] Go 1.22+ and `python3` on `PATH`.
- [ ] Warm the build off camera so `go run` is fast: `cd agent && go run ./cmd/pulse version`.
- [ ] Copy `.env.example` to `.env` at the repo root (gitignored). Fill values locally. Names only:
  - `PRIVATE_KEY` — throwaway Monad **testnet** key. Never a mainnet key, never a mnemonic in the file or the doc.
  - `MONAD_RPC_URL` — `https://testnet-rpc.monad.xyz` (public testnet RPC).
  - `MONAD_CHAIN_ID` — `10143`.
  - `PULSE_TRADE_STAMP` — `0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4`.
- [ ] The `PRIVATE_KEY` address holds **testnet MON** for gas. `--live` sends one `stamp` transaction. Faucet the throwaway address before recording.
- [ ] Load env off camera, without shell tracing: `set -a && source .env && set +a` from the repo root. Do not `cat .env`. Do not `set -x`. The CLI also reads `../.env` when started from `agent/`, and it does not print values.
- [ ] Optional real Jev: export `TYPESAFE_API_KEY` or `JEV_API_KEY` off camera. `demo --jev` exits if neither is set. Leave both unset for the default stub path. Never put the key value in this doc, a slide, or the recording.
- [ ] Second terminal ready at the repo root for `make receipts`.
- [ ] Browser ready for `http://127.0.0.1:8080` and `http://127.0.0.1:8080/?id=N`.
- [ ] README open to **Pipeline** for the 0:20 shot.
- [ ] Success lines to hit: hard risk `PASS`, `tx=0x…`, `decisionHash MATCH`, receipts page shows that id, explorer link opens.

If `MATCH` does not appear, stop. Check testnet MON balance, `MONAD_RPC_URL`, and `PULSE_TRADE_STAMP` before recording another take.

## Metropolis write-up

Paste this into the hackathon project profile (~190 words).

Pulse on Monad is an auditable off-chain trading agent for Metropolis Track 01, Onchain Finance and Trading. Rules read a quote and propose buy, sell, or hold. Hard risk is ordinary Go code and fail-closed. Labeled outcomes may keep those limits or tighten them. An optional TypeSafe Jev call is a soft gate after hard risk. With no API key, the gate records a stub pass so the demo still finishes. Jev is not trained online here.

A decision that clears the gates is hashed and written by PulseTradeStamp on Monad testnet, chain id 10143, contract 0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4. The chain stores the decision hash, symbol, size hint, action, and a short note: a receipt of the decision. Anyone can recompute keccak over the append-only JSONL line and compare it to getReceipt. A static receipts page reads that contract with no key and links each stamp to MonadVision.

The demo takes about three minutes: `cd agent && go run ./cmd/pulse demo --live`, then `make receipts` and open the new id. Repo: https://github.com/vincent-lxc/pulse-on-monad.

Out of scope: a central limit order book, and live DEX execution. Receipts let a third party check the decision on chain.
