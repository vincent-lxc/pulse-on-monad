// Read-only client for PulseTradeStamp (contracts/src/PulseTradeStamp.sol).
// getReceipt(uint256) returns one dynamic Receipt tuple:
//   (bytes32 decisionHash, uint256 symbolId, uint256 sizeHint, uint8 action,
//    string note, uint64 stampedAt, address stamper)
// eth_call prefixes that tuple with an offset word (0x20).

export const DEFAULTS = {
  rpc: "https://testnet-rpc.monad.xyz",
  chainId: 10143,
  contract: "0x6eC692C5792AD122c459AC8Df3FDF67eE80842F4",
  explorer: "https://testnet.monadvision.com",
  limit: 8,
};

export const NEXT_ID_DATA = "0x61b8ce8c";
export const GET_RECEIPT_SELECTOR = "0xb63e6ac3";
export const STAMPED_TOPIC0 =
  "0x8d3088137d8bea8bb136154f20c21e7785d98b9d2e1d0262909fb957051ae80e";
export const UNKNOWN_ID_SELECTOR = "0x48e73c8e";

// Demo quote in agent/internal/data/quote.go uses symbol id 1 for BTC.
// The chain only stores the numeric id.
const DEMO_SYMBOLS = { 1: "BTC" };

const ACTIONS = { 0: "hold", 1: "buy", 2: "sell" };
const MAX_LIMIT = 24;
const LOG_SPAN = 100;

export function actionLabel(action) {
  return ACTIONS[action] ?? `action ${action}`;
}

export function symbolLabel(symbolId) {
  const id = String(symbolId);
  const name = DEMO_SYMBOLS[id];
  return name ? `${name} · #${id}` : `#${id}`;
}

export function shortHex(hex, lead = 8, tail = 6) {
  const h = String(hex || "");
  if (!h.startsWith("0x") || h.length <= 2 + lead + tail + 1) return h;
  return `${h.slice(0, 2 + lead)}…${h.slice(-tail)}`;
}

export function formatStamp(unixSeconds) {
  const n = Number(unixSeconds);
  if (!Number.isFinite(n) || n <= 0) return "—";
  return new Date(n * 1000).toISOString().replace(".000Z", "Z");
}

export function explorerTx(explorer, hash) {
  if (!hash) return "";
  return `${explorer.replace(/\/$/, "")}/tx/${hash}`;
}

export function explorerAddress(explorer, address) {
  if (!address) return "";
  return `${explorer.replace(/\/$/, "")}/address/${address}`;
}

export function pageConfig(params, defaults = DEFAULTS) {
  const rpcRaw = (params.get("rpc") || defaults.rpc).trim();
  const contractRaw = (params.get("contract") || defaults.contract).trim();
  const chainRaw = (params.get("chainId") || String(defaults.chainId)).trim();
  const limitRaw = (params.get("n") || String(defaults.limit)).trim();
  const id = (params.get("id") || "").trim();

  let rpc;
  try {
    const url = new URL(rpcRaw);
    if (url.protocol !== "https:" && url.protocol !== "http:") {
      throw new Error("protocol");
    }
    rpc = url.toString();
  } catch {
    throw new Error("rpc must be an http(s) URL");
  }

  if (!/^0x[0-9a-fA-F]{40}$/.test(contractRaw)) {
    throw new Error("contract must be a 20-byte hex address");
  }

  const chainId = Number(chainRaw);
  if (!Number.isInteger(chainId) || chainId <= 0) {
    throw new Error("chainId must be a positive integer");
  }

  let limit = Number(limitRaw);
  if (!Number.isFinite(limit)) limit = defaults.limit;
  limit = Math.min(MAX_LIMIT, Math.max(1, Math.floor(limit)));

  return {
    rpc,
    contract: contractRaw,
    chainId,
    limit,
    id,
    explorer: defaults.explorer,
  };
}

export function encodeGetReceipt(id) {
  const n = BigInt(id);
  if (n <= 0n) throw new Error("receipt id must be positive");
  return GET_RECEIPT_SELECTOR + n.toString(16).padStart(64, "0");
}

export function decodeReceipt(hex) {
  const raw = hexToBytes(hex);
  if (raw.length < 32) throw new Error("empty getReceipt return");
  const offset = Number(wordToBig(raw, 0));
  if (!Number.isSafeInteger(offset) || offset < 32 || offset >= raw.length || offset % 32 !== 0) {
    throw new Error("bad getReceipt offset");
  }
  const body = raw.subarray(offset);
  if (body.length < 7 * 32) throw new Error("short getReceipt body");

  const noteOff = Number(wordToBig(body, 4));
  if (!Number.isSafeInteger(noteOff) || noteOff < 7 * 32 || noteOff + 32 > body.length) {
    throw new Error("bad note offset");
  }
  const noteLen = Number(wordToBig(body.subarray(noteOff), 0));
  if (!Number.isSafeInteger(noteLen) || noteLen < 0 || noteOff + 32 + noteLen > body.length) {
    throw new Error("bad note length");
  }

  const action = Number(wordToBig(body, 3));
  const stamped = wordToBig(body, 5);
  return {
    decisionHash: bytesToHex(body.subarray(0, 32)),
    symbolId: wordToBig(body, 1).toString(10),
    sizeHint: wordToBig(body, 2).toString(10),
    action,
    note: new TextDecoder().decode(body.subarray(noteOff + 32, noteOff + 32 + noteLen)),
    stampedAt: Number(stamped),
    stamper: bytesToHex(body.subarray(6 * 32 + 12, 7 * 32)),
  };
}

/** Inclusive ranges with (to - from) <= span. Monad testnet rejects wider eth_getLogs. */
export function logRanges(start, end, span = LOG_SPAN) {
  if (!Number.isInteger(start) || !Number.isInteger(end) || !Number.isInteger(span) || span < 1) {
    throw new Error("bad log range");
  }
  if (end < start) return [];
  const out = [];
  let from = start;
  while (from <= end) {
    const to = Math.min(from + span, end);
    out.push({ from, to });
    from = to + 1;
  }
  return out;
}

/**
 * Lowest block number whose timestamp is >= targetTs.
 * readTs(blockNumber) resolves to unix seconds. Returns null when head is older.
 */
export async function firstBlockAtOrAfter(targetTs, latest, readTs) {
  if (!Number.isInteger(latest) || latest < 0) throw new Error("bad head block");
  const target = Number(targetTs);
  const latestTs = await readTs(latest);
  if (latestTs < target) return null;

  let lo = 0;
  let hi = latest;
  let step = 2000;
  let cursor = latest;
  while (cursor > 0) {
    const ts = cursor === latest ? latestTs : await readTs(cursor);
    if (ts <= target) {
      lo = cursor;
      break;
    }
    hi = cursor;
    const next = cursor - step;
    if (next <= 0) break;
    cursor = next;
    if (step < 2_000_000) step *= 2;
  }

  while (lo < hi) {
    const mid = Math.floor((lo + hi) / 2);
    const ts = await readTs(mid);
    if (ts < target) lo = mid + 1;
    else hi = mid;
  }
  return lo;
}

export async function rpcCall(rpcUrl, method, params, fetchImpl = globalThis.fetch) {
  const res = await fetchImpl(rpcUrl, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", id: 1, method, params }),
  });
  const text = await res.text();
  let json;
  try {
    json = JSON.parse(text);
  } catch {
    throw new Error(`RPC ${method} returned HTTP ${res.status}`);
  }
  if (json.error) {
    const data = typeof json.error.data === "string" ? json.error.data : "";
    const err = new Error(json.error.message || `RPC ${method} failed`);
    err.data = data;
    err.code = json.error.code;
    if (data.toLowerCase().startsWith(UNKNOWN_ID_SELECTOR)) {
      err.message = "Unknown receipt id";
    }
    throw err;
  }
  if (!res.ok) throw new Error(`RPC ${method} HTTP ${res.status}`);
  return json.result;
}

export async function fetchChainId(rpcUrl, fetchImpl) {
  const raw = await rpcCall(rpcUrl, "eth_chainId", [], fetchImpl);
  return Number(BigInt(raw));
}

export async function fetchNextId(rpcUrl, contract, fetchImpl) {
  const raw = await rpcCall(
    rpcUrl,
    "eth_call",
    [{ to: contract, data: NEXT_ID_DATA }, "latest"],
    fetchImpl,
  );
  return BigInt(raw);
}

export async function fetchReceipt(rpcUrl, contract, id, fetchImpl) {
  const raw = await rpcCall(
    rpcUrl,
    "eth_call",
    [{ to: contract, data: encodeGetReceipt(id) }, "latest"],
    fetchImpl,
  );
  return decodeReceipt(raw);
}

export async function listReceipts(rpcUrl, contract, limit, fetchImpl) {
  const nextId = await fetchNextId(rpcUrl, contract, fetchImpl);
  const last = nextId - 1n;
  if (last < 1n) return { nextId, receipts: [] };
  const n = BigInt(limit);
  let start = last - n + 1n;
  if (start < 1n) start = 1n;
  const ids = [];
  for (let id = last; id >= start; id--) ids.push(id);
  const receipts = await Promise.all(
    ids.map(async (id) => ({ id, ...(await fetchReceipt(rpcUrl, contract, id, fetchImpl)) })),
  );
  return { nextId, receipts };
}

export function createBlockTimeReader(rpcUrl, fetchImpl) {
  const cache = new Map();
  return async function readTs(blockNumber) {
    if (cache.has(blockNumber)) return cache.get(blockNumber);
    const block = await rpcCall(
      rpcUrl,
      "eth_getBlockByNumber",
      ["0x" + blockNumber.toString(16), false],
      fetchImpl,
    );
    if (!block || !block.timestamp) throw new Error(`missing block ${blockNumber}`);
    const ts = Number(BigInt(block.timestamp));
    cache.set(blockNumber, ts);
    return ts;
  };
}

export async function findStampTx({
  rpcUrl,
  contract,
  id,
  stampedAt,
  readTs,
  fetchImpl,
  radii = [20, 200],
}) {
  const latest = Number(BigInt(await rpcCall(rpcUrl, "eth_blockNumber", [], fetchImpl)));
  const reader = readTs || createBlockTimeReader(rpcUrl, fetchImpl);
  const block = await firstBlockAtOrAfter(stampedAt, latest, reader);
  if (block == null) return null;
  const topic1 = "0x" + BigInt(id).toString(16).padStart(64, "0");
  for (const radius of radii) {
    const start = Math.max(0, block - radius);
    const end = block + radius;
    for (const range of logRanges(start, end, LOG_SPAN)) {
      const logs = await rpcCall(
        rpcUrl,
        "eth_getLogs",
        [
          {
            address: contract,
            topics: [STAMPED_TOPIC0, topic1],
            fromBlock: "0x" + range.from.toString(16),
            toBlock: "0x" + range.to.toString(16),
          },
        ],
        fetchImpl,
      );
      if (Array.isArray(logs) && logs.length > 0 && logs[0].transactionHash) {
        return logs[0].transactionHash;
      }
    }
  }
  return null;
}

function hexToBytes(hex) {
  const h = String(hex || "").replace(/^0x/i, "");
  if (h.length % 2 !== 0) throw new Error("odd hex");
  const out = new Uint8Array(h.length / 2);
  for (let i = 0; i < out.length; i++) {
    const byte = Number.parseInt(h.slice(i * 2, i * 2 + 2), 16);
    if (Number.isNaN(byte)) throw new Error("bad hex");
    out[i] = byte;
  }
  return out;
}

function wordToBig(bytes, wordIndex) {
  const start = wordIndex * 32;
  let n = 0n;
  for (let i = 0; i < 32; i++) n = (n << 8n) + BigInt(bytes[start + i]);
  return n;
}

function bytesToHex(bytes) {
  let hex = "0x";
  for (const b of bytes) hex += b.toString(16).padStart(2, "0");
  return hex;
}
