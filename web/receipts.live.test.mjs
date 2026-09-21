import assert from "node:assert/strict";
import test from "node:test";

import { DEFAULTS, fetchReceipt, findStampTx, listReceipts } from "./receipts.js";

const live = process.env.PULSE_LIVE_SMOKE === "1";

test("live testnet receipt 3 and 4", { skip: !live, timeout: 90_000 }, async () => {
  const third = await fetchReceipt(DEFAULTS.rpc, DEFAULTS.contract, 3);
  assert.equal(third.action, 1);
  assert.equal(third.symbolId, "1");
  assert.equal(
    third.decisionHash,
    "0xf488506599d41cd2763fb389d6839f3f85055eb09548272eea32b7c0e44839b1",
  );
  assert.match(third.note, /pulse-demo|Pulse/);

  const fourth = await fetchReceipt(DEFAULTS.rpc, DEFAULTS.contract, 4);
  assert.equal(fourth.decisionHash.length, 66);
  assert.notEqual(fourth.decisionHash, "0x" + "0".repeat(64));

  const page = await listReceipts(DEFAULTS.rpc, DEFAULTS.contract, 8);
  const ids = page.receipts.map((r) => r.id.toString());
  assert.ok(ids.includes("3") || ids.includes("4"), `listed ${ids.join(",")}`);

  const tx = await findStampTx({
    rpcUrl: DEFAULTS.rpc,
    contract: DEFAULTS.contract,
    id: 4n,
    stampedAt: fourth.stampedAt,
  });
  assert.match(tx, /^0x[0-9a-fA-F]{64}$/);
});
