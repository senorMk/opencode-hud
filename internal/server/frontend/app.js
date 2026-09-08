"use strict";

const $ = (id) => document.getElementById(id);
let lastOk = 0;

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => (
    { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]
  ));
}

function compact(n) {
  n = Number(n) || 0;
  if (n >= 1e9) return (n / 1e9).toFixed(1) + "B";
  if (n >= 1e6) return (n / 1e6).toFixed(1) + "M";
  if (n >= 1e3) return (n / 1e3).toFixed(1) + "K";
  return String(n);
}

function ago(ms) {
  const s = Math.max(0, (Date.now() - ms) / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return Math.floor(s / 60) + "m ago";
  if (s < 86400) return Math.floor(s / 3600) + "h ago";
  return Math.floor(s / 86400) + "d ago";
}

function shortModel(m) {
  const i = String(m).lastIndexOf("/");
  return i >= 0 ? String(m).slice(i + 1) : String(m);
}

function render(snap) {
  const t = snap.today.totals;
  $("day").textContent = snap.today.day;
  $("c-cost").textContent = "$" + t.cost.toFixed(2);
  $("c-msgs").textContent = compact(t.messages);
  $("c-sess").textContent = t.sessions;
  $("c-in").textContent = compact(t.input);
  $("c-out").textContent = compact(t.output);
  $("c-cache").textContent = compact(t.cacheRead);

  const active = snap.recent.filter((s) => Date.now() - s.updatedMs < 5 * 60 * 1000);
  $("active").innerHTML = active.length === 0
    ? '<p class="muted">No active sessions.</p>'
    : active.map((s) => `<div class="srow"><div class="t"><span class="agodot">●</span>${esc(s.title || s.directory)}</div><div class="m">${esc(s.project)} · ${esc(shortModel(s.model))}</div></div>`).join("");

  const rows = snap.today.sessions.slice().sort((a, b) => b.updatedMs - a.updatedMs);
  $("sessions").innerHTML = rows.length === 0
    ? '<p class="muted">No sessions today yet.</p>'
    : rows.map((s) => `
      <div class="srow">
        <div class="t">${esc(s.title || s.directory)}</div>
        <div class="m">${esc(s.project)} · ${esc(shortModel(s.model))} · ${esc(s.agent)}</div>
        <div class="f"><span class="cost">$${s.cost.toFixed(2)}</span><span class="muted">${compact(s.input + s.output + s.reasoning)} tok · ${ago(s.updatedMs)}</span></div>
      </div>`).join("");

  $("updated").textContent = "Updated " + new Date(snap.at).toLocaleTimeString();
  $("live-dot").classList.remove("stale");
  lastOk = Date.now();
}

async function poll() {
  try {
    const r = await fetch("/api/snapshot", { cache: "no-store" });
    if (!r.ok) throw new Error(r.status);
    render(await r.json());
  } catch (e) {
    if (Date.now() - lastOk > 5000) $("live-dot").classList.add("stale");
  }
}

$("tab-today").onclick = () => {
  $("tab-today").classList.add("active"); $("tab-sessions").classList.remove("active");
  $("view-today").hidden = false; $("view-sessions").hidden = true;
};
$("tab-sessions").onclick = () => {
  $("tab-sessions").classList.add("active"); $("tab-today").classList.remove("active");
  $("view-sessions").hidden = false; $("view-today").hidden = true;
};

poll();
setInterval(poll, 2000);
