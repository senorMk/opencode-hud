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

function renderDay(rep, recent) {
  const t = rep.totals;
  $("c-cost").textContent = "$" + t.cost.toFixed(2);
  $("c-msgs").textContent = compact(t.messages);
  $("c-sess").textContent = t.sessions;
  $("c-in").textContent = compact(t.input);
  $("c-out").textContent = compact(t.output);
  $("c-cache").textContent = compact(t.cacheRead);

  if (recent) {
    const active = recent.filter((s) => Date.now() - s.updatedMs < 5 * 60 * 1000);
    $("active").innerHTML = active.length === 0
      ? '<p class="muted">No active sessions.</p>'
      : active.map((s) => `<div class="srow"><div class="t"><span class="agodot">●</span>${esc(s.displayTitle || s.directory)}</div><div class="m">${esc(s.project)} · ${esc(shortModel(s.model))}</div><div class="f"><span class="cost">$${s.cost.toFixed(2)}</span><span class="muted">${compact(s.input)} in / ${compact(s.output)} out / ${compact(s.cacheRead)} cache</span></div></div>`).join("");
  } else {
    $("active").innerHTML = '<p class="muted">Active sessions only shown for today.</p>';
  }

  const rows = rep.sessions.slice().sort((a, b) => b.updatedMs - a.updatedMs);
  const emptyMsg = rep.from
    ? "No sessions in this range."
    : "No sessions on this day.";
  $("sessions").innerHTML = rows.length === 0
    ? `<p class="muted">${emptyMsg}</p>`
    : rows.map((s) => `
      <div class="srow">
        <div class="t">${esc(s.displayTitle || s.directory)}</div>
        <div class="m">${esc(s.project)} · ${esc(shortModel(s.model))} · ${esc(s.agent)}</div>
        <div class="f"><span class="cost">$${s.cost.toFixed(2)}</span><span class="muted">${compact(s.input + s.output + s.reasoning)} tok · ${ago(s.updatedMs)}</span></div>
      </div>`).join("");

  $("updated").textContent = "Updated " + new Date().toLocaleTimeString();
  $("live-dot").classList.remove("stale");
  lastOk = Date.now();
}

function todayStr(d) {
  d = d || new Date();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${m}-${dd}`;
}

function shiftDay(day, delta) {
  const d = new Date(day + "T12:00:00");
  d.setDate(d.getDate() + delta);
  return todayStr(d);
}

let viewMode = "day";
let viewDay = todayStr();

function monthStr(d) {
  d = d || new Date();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  return `${d.getFullYear()}-${m}`;
}

let viewMonth = monthStr();

function shiftMonth(month, delta) {
  const d = new Date(month + "-01T12:00:00");
  d.setMonth(d.getMonth() + delta);
  return monthStr(d);
}

function syncDateNav() {
  const today = todayStr();
  const thisMonth = monthStr();
  $("mode-day").classList.toggle("active", viewMode === "day");
  $("mode-month").classList.toggle("active", viewMode === "month");
  $("day-nav").hidden = viewMode !== "day";
  $("month-nav").hidden = viewMode !== "month";
  if (viewMode === "day") {
    $("day-picker").value = viewDay;
    $("day-picker").max = today;
    $("day-next").disabled = viewDay >= today;
    $("day-today").hidden = viewDay === today;
  } else {
    $("month-picker").value = viewMonth;
    $("month-picker").max = thisMonth;
    $("month-next").disabled = viewMonth >= thisMonth;
    $("month-today").hidden = viewMonth === thisMonth;
  }
}

async function poll() {
  try {
    let rep, recent;
    if (viewMode === "month") {
      const r = await fetch("/api/range?month=" + viewMonth, { cache: "no-store" });
      if (!r.ok) throw new Error(r.status);
      rep = await r.json(); recent = null;
    } else if (viewDay === todayStr()) {
      const r = await fetch("/api/snapshot", { cache: "no-store" });
      if (!r.ok) throw new Error(r.status);
      const snap = await r.json();
      rep = snap.today; recent = snap.recent;
    } else {
      const r = await fetch("/api/day?day=" + viewDay, { cache: "no-store" });
      if (!r.ok) throw new Error(r.status);
      rep = await r.json(); recent = null;
    }
    renderDay(rep, recent);
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

$("day-prev").onclick = () => { viewDay = shiftDay(viewDay, -1); syncDateNav(); poll(); };
$("day-next").onclick = () => { viewDay = shiftDay(viewDay, 1); syncDateNav(); poll(); };
$("day-today").onclick = () => { viewDay = todayStr(); syncDateNav(); poll(); };
$("day-picker").onchange = (e) => { if (e.target.value) { viewDay = e.target.value; syncDateNav(); poll(); } };

$("mode-day").onclick = () => { viewMode = "day"; syncDateNav(); poll(); };
$("mode-month").onclick = () => { viewMode = "month"; syncDateNav(); poll(); };
$("month-prev").onclick = () => { viewMonth = shiftMonth(viewMonth, -1); syncDateNav(); poll(); };
$("month-next").onclick = () => { viewMonth = shiftMonth(viewMonth, 1); syncDateNav(); poll(); };
$("month-today").onclick = () => { viewMonth = monthStr(); syncDateNav(); poll(); };
$("month-picker").onchange = (e) => { if (e.target.value) { viewMonth = e.target.value; syncDateNav(); poll(); } };

syncDateNav();
poll();
setInterval(poll, 2000);
