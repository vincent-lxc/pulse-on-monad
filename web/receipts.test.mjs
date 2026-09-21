import assert from "node:assert/strict";
import test from "node:test";

import {
  actionLabel,
  decodeReceipt,
  encodeGetReceipt,
  firstBlockAtOrAfter,
  formatStamp,
  logRanges,
  pageConfig,
  rpcCall,
  symbolLabel,
} from "./receipts.js";

// Live eth_call of getReceipt(3) on PulseTradeStamp 0x6eC6…42F4 (Monad testnet).
const RECEIPT_3 =
  "0x0000000000000000000000000000000000000000000000000000000000000020" +
  "f488506599d41cd2763fb389d6839f3f85055eb09548272eea32b7c0e44839b1" +
  "0000000000000000000000000000000000000000000000000000000000000001" +
  "0000000000000000000000000000000000000000000000000000000000000064" +
  "0000000000000000000000000000000000000000000000000000000000000001" +
  "00000000000000000000000000000000000000000000000000000000000000e0" +
  "000000000000000000000000000000000000000000000000000000006ab0c86f" +
  "0000000000000000000000002e24006d3b0ad37687d71185efaff165087ac776" +
  "0000000000000000000000000000000000000000000000000000000000000029" +
  "6167656e743d70756c73652d64656d6f2072756e3d72756e2d39363732633133663261633032643236" +
  "000000000000000000000000000000000000000000000000";

test("decodeReceipt reads the on-chain Receipt tuple", () => {
  const got = decodeReceipt(RECEIPT_3);
  assert.equal(
    got.decisionHash,
    "0xf488506599d41cd2763fb389d6839f3f85055eb09548272eea32b7c0e44839b1",
  );
  assert.equal(got.symbolId, "1");
  assert.equal(got.sizeHint, "100");
  assert.equal(got.action, 1);
  assert.equal(got.note, "agent=pulse-demo run=run-9672c13f2ac02d26");
  assert.equal(got.stampedAt, 1789970543);
  assert.equal(got.stamper, "0x2e24006d3b0ad37687d71185efaff165087ac776");
  assert.equal(symbolLabel(got.symbolId), "BTC · #1");
  assert.equal(actionLabel(got.action), "buy");
  assert.equal(formatStamp(1_700_000_000), "2023-11-14T22:13:20Z");
});

test("decodeReceipt rejects a short payload", () => {
  assert.throws(() => decodeReceipt("0x01"), /empty|odd/);
});

test("encodeGetReceipt packs the uint256 id", () => {
  assert.equal(
    encodeGetReceipt(3),
    "0xb63e6ac3" + "3".padStart(64, "0"),
  );
  assert.throws(() => encodeGetReceipt(0), /positive/);
});

test("logRanges stay inside the 100-block eth_getLogs cap", () => {
  const ranges = logRanges(10, 250, 100);
  assert.deepEqual(ranges, [
    { from: 10, to: 110 },
    { from: 111, to: 211 },
    { from: 212, to: 250 },
  ]);
  for (const range of ranges) assert.ok(range.to - range.from <= 100);
});

test("firstBlockAtOrAfter is a timestamp lower bound", async () => {
  const stamps = [10, 10, 11, 14, 14, 20];
  const read = async (n) => stamps[n];
  assert.equal(await firstBlockAtOrAfter(10, 5, read), 0);
  assert.equal(await firstBlockAtOrAfter(11, 5, read), 2);
  assert.equal(await firstBlockAtOrAfter(12, 5, read), 3);
  assert.equal(await firstBlockAtOrAfter(20, 5, read), 5);
  assert.equal(await firstBlockAtOrAfter(21, 5, read), null);

  const readLinear = async (n) => 1_000_000 + n;
  assert.equal(await firstBlockAtOrAfter(1_150_000, 200_000, readLinear), 150_000);
});

test("rpcCall maps UnknownId reverts", async () => {
  const fetchImpl = async () => ({
    ok: true,
    status: 200,
    async text() {
      return JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        error: { code: 3, message: "execution reverted", data: "0x48e73c8e" },
      });
    },
  });
  await assert.rejects(
    () => rpcCall("https://example.test/rpc", "eth_call", [], fetchImpl),
    /Unknown receipt id/,
  );
});

test("pageConfig applies query overrides and rejects bad inputs", () => {
  const cfg = pageConfig(
    new URLSearchParams(
      "rpc=https://example.test/rpc&contract=0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4&chainId=10143&n=3&id=4",
    ),
  );
  assert.equal(cfg.rpc, "https://example.test/rpc");
  assert.equal(cfg.chainId, 10143);
  assert.equal(cfg.limit, 3);
  assert.equal(cfg.id, "4");
  assert.equal(pageConfig(new URLSearchParams("n=999")).limit, 24);
  assert.throws(() => pageConfig(new URLSearchParams("rpc=file:///tmp")), /http/);
  assert.throws(() => pageConfig(new URLSearchParams("contract=0x123")), /contract/);
});
