import {
  actionLabel,
  createBlockTimeReader,
  explorerAddress,
  explorerTx,
  fetchChainId,
  fetchReceipt,
  findStampTx,
  formatStamp,
  listReceipts,
  pageConfig,
  shortHex,
  symbolLabel,
} from "./receipts.js";

const statusEl = document.querySelector("#status");
const rowsEl = document.querySelector("#rows");
const emptyEl = document.querySelector("#empty");
const focusEl = document.querySelector("#focus");
const metaEl = document.querySelector("#meta");
const form = document.querySelector("#lookup");
const idInput = document.querySelector("#receipt-id");
const contractLink = document.querySelector("#contract-link");
const chainEl = document.querySelector("#chain");

const cfg = readConfig();
let listed = [];
let nextId = null;
let listGen = 0;
let lookupGen = 0;

boot();

function readConfig() {
  try {
    return pageConfig(new URLSearchParams(location.search));
  } catch (err) {
    setStatus(err.message, true);
    throw err;
  }
}

async function boot() {
  chainEl.textContent = `chain ${cfg.chainId}`;
  contractLink.textContent = shortHex(cfg.contract, 6, 4);
  contractLink.href = explorerAddress(cfg.explorer, cfg.contract);
  metaEl.textContent = rpcHost(cfg.rpc);
  if (cfg.id) idInput.value = cfg.id;
  form.addEventListener("submit", (ev) => {
    ev.preventDefault();
    const id = idInput.value.trim();
    setQueryId(id);
    lookup(id);
  });
  document.querySelector("#refresh").addEventListener("click", () => loadList());
  rowsEl.addEventListener("click", onRowClick);
  await loadList();
  if (cfg.id) await lookup(cfg.id);
}

async function loadList() {
  const gen = ++listGen;
  const lookupAtStart = lookupGen;
  setStatus("Reading receipts…");
  rowsEl.replaceChildren();
  emptyEl.hidden = true;
  try {
    const onchain = await fetchChainId(cfg.rpc);
    if (onchain !== cfg.chainId) {
      setStatus(`RPC chain id ${onchain} does not match ${cfg.chainId}.`, true);
    }
    const page = await listReceipts(cfg.rpc, cfg.contract, cfg.limit);
    if (gen !== listGen) return;
    nextId = page.nextId;
    listed = page.receipts;
    renderRows(listed);
    const shown = listed.length;
    const head = nextId > 1n ? `next id ${nextId}` : "no stamps yet";
    if (onchain === cfg.chainId && lookupAtStart === lookupGen) {
      setStatus(`${head} · showing ${shown}`);
    }
    emptyEl.hidden = shown > 0;
    resolveTxs(listed, gen);
  } catch (err) {
    setStatus(err.message || "Could not read receipts", true);
  }
}

async function lookup(raw) {
  const gen = ++lookupGen;
  if (!raw) {
    focusEl.hidden = true;
    focusEl.replaceChildren();
    markActive("");
    return;
  }
  if (!/^[1-9]\d*$/.test(raw)) {
    setStatus("Receipt id must be a positive integer.", true);
    return;
  }
  markActive(raw);
  const known = listed.find((r) => r.id.toString() === raw);
  if (known) {
    renderFocus(known);
    document.getElementById(`receipt-${raw}`)?.scrollIntoView({ block: "nearest" });
  }
  setStatus(`Reading receipt ${raw}…`);
  try {
    const receipt = known || { id: BigInt(raw), ...(await fetchReceipt(cfg.rpc, cfg.contract, raw)) };
    if (gen !== lookupGen) return;
    renderFocus(receipt);
    setStatus(`Receipt ${raw} · ${actionLabel(receipt.action)} · ${symbolLabel(receipt.symbolId)}`);
    if (!receipt.txHash) {
      const tx = await findStampTx({
        rpcUrl: cfg.rpc,
        contract: cfg.contract,
        id: receipt.id,
        stampedAt: receipt.stampedAt,
      });
      if (gen !== lookupGen) return;
      receipt.txHash = tx || "";
      renderFocus(receipt);
      const row = listed.find((r) => r.id === receipt.id);
      if (row) {
        row.txHash = receipt.txHash;
        paintTx(row);
      }
    }
  } catch (err) {
    if (gen !== lookupGen) return;
    setStatus(err.message || "Lookup failed", true);
  }
}

function renderRows(receipts) {
  rowsEl.replaceChildren(
    ...receipts.map((r) => {
      const tr = document.createElement("tr");
      tr.id = `receipt-${r.id}`;
      tr.append(
        cell(r.id.toString(), "id", "Id"),
        symbolCell(r),
        actionCell(r.action),
        hashCell(r.decisionHash),
        txCell(r),
        cell(formatStamp(r.stampedAt), "time", "Stamped"),
      );
      return tr;
    }),
  );
}

function renderFocus(r) {
  focusEl.hidden = false;
  focusEl.replaceChildren();
  const title = document.createElement("h2");
  title.textContent = `Receipt ${r.id}`;
  const dl = document.createElement("dl");
  addDef(dl, "Symbol / note", `${symbolLabel(r.symbolId)} — ${r.note || "—"}`);
  addDef(dl, "Action", actionLabel(r.action));
  addDef(dl, "decisionHash", r.decisionHash);
  addDef(dl, "Size hint", r.sizeHint);
  addDef(dl, "Stamped", formatStamp(r.stampedAt));
  addDef(dl, "Stamper", r.stamper);
  const txRow = document.createElement("div");
  txRow.className = "tx-line";
  txRow.append(document.createTextNode("Tx "));
  txRow.append(txAnchor(r));
  focusEl.append(title, dl, txRow);
}

function addDef(dl, term, value) {
  const dt = document.createElement("dt");
  dt.textContent = term;
  const dd = document.createElement("dd");
  dd.textContent = value;
  dl.append(dt, dd);
}

function cell(text, className, label) {
  const td = document.createElement("td");
  td.className = className;
  td.dataset.label = label;
  td.textContent = text;
  return td;
}

function symbolCell(r) {
  const td = document.createElement("td");
  td.dataset.label = "Symbol / note";
  const strong = document.createElement("strong");
  strong.textContent = symbolLabel(r.symbolId);
  const note = document.createElement("span");
  note.className = "note";
  note.textContent = r.note || "—";
  td.append(strong, note);
  return td;
}

function actionCell(action) {
  const td = document.createElement("td");
  td.dataset.label = "Action";
  const span = document.createElement("span");
  const label = actionLabel(action);
  span.className = `act ${label}`;
  span.textContent = label;
  td.append(span);
  return td;
}

function hashCell(hash) {
  const td = document.createElement("td");
  td.dataset.label = "decisionHash";
  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = "hash";
  btn.dataset.copy = hash;
  btn.textContent = shortHex(hash);
  btn.title = "Copy decisionHash";
  td.append(btn);
  return td;
}

function txCell(r) {
  const td = document.createElement("td");
  td.className = "tx";
  td.dataset.label = "Tx";
  td.dataset.id = r.id.toString();
  td.append(txAnchor(r));
  return td;
}

function txAnchor(r) {
  if (!r.txHash) {
    const span = document.createElement("span");
    span.className = "pending";
    span.textContent = r.txHash === "" ? "—" : "finding";
    return span;
  }
  const a = document.createElement("a");
  a.href = explorerTx(cfg.explorer, r.txHash);
  a.target = "_blank";
  a.rel = "noopener noreferrer";
  a.textContent = shortHex(r.txHash, 6, 4);
  return a;
}

function paintTx(r) {
  const td = rowsEl.querySelector(`td.tx[data-id="${CSS.escape(r.id.toString())}"]`);
  if (!td) return;
  td.replaceChildren(txAnchor(r));
}

async function resolveTxs(receipts, gen) {
  const readTs = createBlockTimeReader(cfg.rpc);
  await mapPool(receipts, 2, async (r) => {
    if (gen !== listGen) return;
    try {
      const tx = await findStampTx({
        rpcUrl: cfg.rpc,
        contract: cfg.contract,
        id: r.id,
        stampedAt: r.stampedAt,
        readTs,
      });
      if (gen !== listGen) return;
      r.txHash = tx || "";
    } catch {
      if (gen !== listGen) return;
      r.txHash = "";
    }
    paintTx(r);
    if (!focusEl.hidden && focusEl.querySelector("h2")?.textContent === `Receipt ${r.id}`) {
      renderFocus(r);
    }
  });
}

async function mapPool(items, limit, fn) {
  let cursor = 0;
  async function worker() {
    while (cursor < items.length) {
      const idx = cursor++;
      await fn(items[idx], idx);
    }
  }
  const n = Math.min(limit, items.length);
  await Promise.all(Array.from({ length: n }, () => worker()));
}

function onRowClick(ev) {
  const btn = ev.target.closest("button[data-copy]");
  if (!btn) return;
  copyText(btn.dataset.copy, btn);
}

async function copyText(text, btn) {
  const previous = btn.textContent;
  try {
    await navigator.clipboard.writeText(text);
    btn.textContent = "copied";
  } catch {
    btn.textContent = "copy failed";
  }
  setTimeout(() => {
    btn.textContent = previous;
  }, 900);
}

function markActive(id) {
  for (const tr of rowsEl.querySelectorAll("tr")) {
    tr.classList.toggle("active", tr.id === `receipt-${id}`);
  }
}

function setQueryId(id) {
  const url = new URL(location.href);
  if (id) url.searchParams.set("id", id);
  else url.searchParams.delete("id");
  history.replaceState(null, "", url);
}

function setStatus(text, isError = false) {
  statusEl.textContent = text;
  statusEl.classList.toggle("error", isError);
}

function rpcHost(rpc) {
  try {
    return new URL(rpc).host;
  } catch {
    return rpc;
  }
}
