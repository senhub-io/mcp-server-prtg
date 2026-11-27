"use strict";

const $ = (s) => document.querySelector(s);
const $$ = (s) => Array.from(document.querySelectorAll(s));

const state = {
  settings: null,
  status: null,
  route: "overview",
  refreshTimer: null,
  restartTimer: null,
};

function el(tag, cls, text) {
  const n = document.createElement(tag);
  if (cls) n.className = cls;
  if (text !== undefined && text !== null) n.textContent = String(text);
  return n;
}

function clear(node) {
  while (node.firstChild) node.removeChild(node.firstChild);
}

function fmtTime(iso) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return String(iso);
  return d.toLocaleString();
}

function pill(kind, text) {
  return el("span", "pill " + kind, text);
}

async function api(path, opts) {
  const options = opts || {};
  const headers = Object.assign({ "Accept": "application/json" }, options.headers || {});
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    headers["X-Requested-With"] = "mcp-admin";
  }
  const res = await fetch(path, {
    method: options.method || "GET",
    headers: headers,
    credentials: "same-origin",
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });
  if (res.status === 401) {
    showLogin();
    throw new Error("unauthorized");
  }
  return res;
}

async function apiJson(path, opts) {
  const res = await api(path, opts);
  if (res.status === 204) return null;
  const text = await res.text();
  let data = null;
  if (text) {
    try { data = JSON.parse(text); } catch (e) { data = { error: text }; }
  }
  if (!res.ok) {
    const err = new Error((data && data.error) || ("HTTP " + res.status));
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

/* ---------- views ---------- */

function showLogin() {
  $("#restart-banner").classList.add("hide");
  $("#app-view").classList.add("hide");
  $("#login-view").classList.remove("hide");
  stopAutoRefresh();
  const key = $("#login-key");
  key.value = "";
  $("#login-error").classList.add("hide");
  setTimeout(() => key.focus(), 0);
}

function showApp() {
  $("#login-view").classList.add("hide");
  $("#app-view").classList.remove("hide");
}

function setRoute(route) {
  state.route = route;
  $$("#main-nav a").forEach((a) => {
    a.classList.toggle("on", a.getAttribute("data-route") === route);
  });
  $$(".view").forEach((v) => v.classList.add("hide"));
  const view = $("#view-" + route);
  if (view) view.classList.remove("hide");
  if (route === "overview") loadOverview();
  if (route === "settings") loadSettings();
  if (route === "clients") loadClientConfig();
  if (route === "overview") startAutoRefresh(); else stopAutoRefresh();
}

function startAutoRefresh() {
  stopAutoRefresh();
  state.refreshTimer = setInterval(() => {
    if (state.route === "overview" && document.visibilityState === "visible") loadOverview(true);
  }, 30000);
}

function stopAutoRefresh() {
  if (state.refreshTimer) { clearInterval(state.refreshTimer); state.refreshTimer = null; }
}

/* ---------- login ---------- */

async function doLogin(ev) {
  ev.preventDefault();
  const btn = $("#login-btn");
  const err = $("#login-error");
  err.classList.add("hide");
  btn.disabled = true;
  try {
    const res = await fetch("/admin/api/login", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Requested-With": "mcp-admin" },
      credentials: "same-origin",
      body: JSON.stringify({ api_key: $("#login-key").value }),
    });
    if (res.status === 429) {
      err.textContent = "Too many attempts, wait a minute.";
      err.classList.remove("hide");
      return;
    }
    if (res.status !== 200) {
      err.textContent = "Wrong administration key.";
      err.classList.remove("hide");
      return;
    }
    showApp();
    setRoute(currentHash() || "overview");
  } catch (e) {
    err.textContent = "Sign-in failed. Try again.";
    err.classList.remove("hide");
  } finally {
    btn.disabled = false;
  }
}

async function doLogout() {
  try { await api("/admin/api/logout", { method: "POST", body: {} }); } catch (e) { /* ignore */ }
  showLogin();
}

function currentHash() {
  const h = (location.hash || "").replace(/^#/, "");
  return ["overview", "settings", "clients"].includes(h) ? h : "overview";
}

/* ---------- overview ---------- */

function kvRow(term, valueNode) {
  const dt = el("dt", null, term);
  const dd = el("dd");
  if (typeof valueNode === "string") dd.textContent = valueNode; else dd.appendChild(valueNode);
  return [dt, dd];
}

async function loadOverview(silent) {
  let status;
  try {
    status = await apiJson("/admin/api/status");
  } catch (e) {
    if (!silent) {
      const body = $("#overview-body");
      clear(body);
      const b = el("div", "banner err", "Could not load status.");
      body.appendChild(b);
    }
    return;
  }
  state.status = status;
  renderOverview(status);
}

function renderOverview(s) {
  const body = $("#overview-body");
  clear(body);

  // Service card
  const service = el("div", "card");
  const sh = el("div", "card-h");
  sh.appendChild(el("h2", null, "Service"));
  service.appendChild(sh);
  const kv = el("dl", "kv left");
  const metrics = s.metrics_sources || {};
  kv.append(...kvRow("Version", s.version || "—"));
  kv.append(...kvRow("Uptime", s.uptime || "—"));
  kv.append(...kvRow("Tools", s.tools_count !== undefined ? String(s.tools_count) : "—"));
  const ms = el("span", null);
  ms.appendChild(pill(metrics.database ? "ok" : "off", "database"));
  ms.appendChild(document.createTextNode(" "));
  ms.appendChild(pill(metrics.prtg ? "ok" : "off", "prtg"));
  kv.append(...kvRow("Metric sources", ms));
  service.appendChild(kv);
  body.appendChild(service);

  // Database card
  const db = el("div", "card");
  const dh = el("div", "card-h");
  dh.appendChild(el("h2", null, "Database"));
  const dbStatus = s.database && s.database.status ? s.database.status : "not_configured";
  const dbPill = dbStatus === "connected" ? pill("ok", "connected")
    : dbStatus === "error" ? pill("err", "error")
    : pill("off", "not configured");
  dh.appendChild(dbPill);
  db.appendChild(dh);
  const dbkv = el("dl", "kv left");
  dbkv.append(...kvRow("Host", s.database && s.database.host ? s.database.host : "—"));
  dbkv.append(...kvRow("Database", s.database && s.database.name ? s.database.name : "—"));
  if (s.database && s.database.error) dbkv.append(...kvRow("Error", s.database.error));
  db.appendChild(dbkv);
  const schema = s.database && s.database.schema;
  if (schema) {
    const flags = el("div", "row wrap mt3");
    const flag = (label, on) => flags.appendChild(pill(on ? "ok" : "off", label));
    flag("soft delete: " + (schema.soft_delete ? "yes" : "no"), schema.soft_delete);
    flag("tree depth column: " + (schema.tree_depth ? "yes" : "no"), schema.tree_depth);
    flag("channel history: " + (schema.channel_values ? "yes" : "no"), schema.channel_values);
    flag("collection status: " + (schema.collection_status ? "yes" : "no"), schema.collection_status);
    flag("per-sensor freshness: " + (schema.last_fetched ? "yes" : "no"), schema.last_fetched);
    db.appendChild(flags);
  }
  body.appendChild(db);

  // PRTG card
  const prtg = el("div", "card");
  const ph = el("div", "card-h");
  ph.appendChild(el("h2", null, "PRTG API"));
  const p = s.prtg || {};
  ph.appendChild(p.configured ? pill("ok", "configured") : pill("off", "not configured"));
  prtg.appendChild(ph);
  const pkv = el("dl", "kv left");
  pkv.append(...kvRow("Base URL", p.base_url || "—"));
  if (p.error) pkv.append(...kvRow("Error", p.error));
  prtg.appendChild(pkv);
  body.appendChild(prtg);

  // Statistics card
  const stats = el("div", "card");
  const sth = el("div", "card-h");
  sth.appendChild(el("h2", null, "Channel history coverage"));
  stats.appendChild(sth);
  const st = s.statistics || {};
  const skv = el("dl", "kv left");
  skv.append(...kvRow("Sensors", st.total_sensors !== undefined ? String(st.total_sensors) : "—"));
  skv.append(...kvRow("Devices", st.total_devices !== undefined ? String(st.total_devices) : "—"));
  skv.append(...kvRow("Groups", st.total_groups !== undefined ? String(st.total_groups) : "—"));
  skv.append(...kvRow("Last collection", fmtTime(st.last_collection_utc)));
  stats.appendChild(skv);
  const cov = st.export_coverage || {};
  const cgrid = el("div", "grid g3 mt3");
  const covItem = (label, val) => {
    const c = el("div", "card tight");
    c.appendChild(el("div", "muted xs", label));
    c.appendChild(el("div", null, val !== undefined ? String(val) : "—"));
    return c;
  };
  cgrid.appendChild(covItem("Up to date", cov.sensors_up_to_date));
  cgrid.appendChild(covItem("Stale", cov.sensors_stale));
  cgrid.appendChild(covItem("Never fetched", cov.sensors_never_fetched));
  cgrid.appendChild(covItem("Without data", cov.sensors_without_data));
  cgrid.appendChild(covItem("Not tracked", cov.sensors_not_tracked));
  cgrid.appendChild(covItem("Fresh within (h)", cov.fresh_within_hours));
  stats.appendChild(cgrid);
  body.appendChild(stats);

  // Collection profiles table
  const prof = el("div", "card");
  const profh = el("div", "card-h");
  profh.appendChild(el("h2", null, "Data Exporter collection profiles"));
  prof.appendChild(profh);
  const profiles = (s.statistics && s.statistics.collection_profiles) || [];
  if (!profiles.length) {
    prof.appendChild(el("p", "muted", "No collection profiles reported."));
  } else {
    const table = el("table", "tbl compact");
    const thead = el("thead");
    const hr = el("tr");
    ["Alias", "Kind", "Last collection", "Next collection", "Flags"].forEach((t) => hr.appendChild(el("th", null, t)));
    thead.appendChild(hr);
    table.appendChild(thead);
    const tb = el("tbody");
    profiles.forEach((p) => {
      const tr = el("tr");
      const alias = el("td", "name", p.alias || "—");
      tr.appendChild(alias);
      tr.appendChild(el("td", null, p.kind || "—"));
      tr.appendChild(el("td", null, fmtTime(p.last_collection_end_utc)));
      tr.appendChild(el("td", null, fmtTime(p.next_collection_utc)));
      const flags = el("td");
      const frow = el("span", "row wrap");
      if (p.paused) frow.appendChild(pill("warn", "flagged paused by Data Exporter"));
      if (p.has_errors) frow.appendChild(pill("err", "errors"));
      if (p.has_warnings) frow.appendChild(pill("warn", "warnings"));
      if (!p.paused && !p.has_errors && !p.has_warnings) frow.appendChild(pill("ok", "ok"));
      flags.appendChild(frow);
      tr.appendChild(flags);
      tb.appendChild(tr);
    });
    table.appendChild(tb);
    prof.appendChild(table);
  }
  body.appendChild(prof);
}

/* ---------- settings ---------- */

function setSecretState(node, isSet) {
  node.textContent = isSet ? "(stored — leave empty to keep)" : "(not set)";
}

async function loadSettings() {
  let s;
  try {
    s = await apiJson("/admin/api/settings");
  } catch (e) { return; }
  state.settings = s;
  $("#db-host").value = s.database.host || "";
  $("#db-port").value = s.database.port || "";
  $("#db-name").value = s.database.name || "";
  $("#db-user").value = s.database.user || "";
  $("#db-sslmode").value = s.database.sslmode || "disable";
  $("#db-password").value = "";
  $("#db-password").placeholder = s.database.password_set ? "unchanged" : "";
  setSecretState($("#db-pass-state"), s.database.password_set);
  $("#db-clear").checked = false;

  $("#prtg-enabled").checked = !!s.prtg.enabled;
  $("#prtg-url").value = s.prtg.base_url || "";
  $("#prtg-token").value = "";
  $("#prtg-token").placeholder = s.prtg.token_set ? "unchanged" : "";
  setSecretState($("#prtg-token-state"), s.prtg.token_set);
  $("#prtg-timeout").value = s.prtg.timeout_seconds || "";
  $("#prtg-verify").checked = !!s.prtg.verify_ssl;

  $("#server-port").value = s.server_port || "";
  $("#log-level").value = s.log_level || "info";

  $("#save-state").textContent = "";
  $("#save-error").classList.add("hide");
}

function dbFormValues() {
  const port = parseInt($("#db-port").value, 10);
  const v = {
    host: $("#db-host").value.trim(),
    port: port,
    name: $("#db-name").value.trim(),
    user: $("#db-user").value.trim(),
    sslmode: $("#db-sslmode").value,
  };
  const pw = $("#db-password").value;
  if ($("#db-clear").checked) v.password = "";
  else if (pw !== "") v.password = pw;
  return v;
}

function prtgFormValues() {
  const v = {
    enabled: $("#prtg-enabled").checked,
    base_url: $("#prtg-url").value.trim(),
    timeout_seconds: parseInt($("#prtg-timeout").value, 10),
    verify_ssl: $("#prtg-verify").checked,
  };
  const tk = $("#prtg-token").value;
  if (tk !== "") v.token = tk;
  return v;
}

function validateSettings(db, prtg) {
  if (!db.port || db.port < 1 || db.port > 65535) return "Database port must be 1–65535.";
  const sport = parseInt($("#server-port").value, 10);
  if (!sport || sport < 1 || sport > 65535) return "Server port must be 1–65535.";
  if (prtg.enabled && !/^https?:\/\//i.test(prtg.base_url)) return "PRTG base URL must start with http:// or https://.";
  return null;
}

function renderTestResult(container, result) {
  clear(container);
  const b = el("div", "banner " + (result.ok ? "ok" : "err"));
  b.appendChild(el("strong", null, (result.ok ? "OK" : "Failed") + " — " + result.message));
  if (result.duration_ms !== undefined) {
    b.appendChild(document.createTextNode(" (" + result.duration_ms + " ms)"));
  }
  container.appendChild(b);
  if (result.details && result.details.length) {
    const ul = el("ul");
    result.details.forEach((d) => ul.appendChild(el("li", null, d)));
    b.appendChild(ul);
  }
}

async function testDb() {
  const v = dbFormValues();
  const err = validateSettings(v, prtgFormValues());
  const box = $("#db-test-result");
  if (err) { clear(box); box.appendChild(el("div", "banner err", err)); return; }
  try {
    const r = await apiJson("/admin/api/test/database", { method: "POST", body: v });
    renderTestResult(box, r);
  } catch (e) {
    clear(box);
    box.appendChild(el("div", "banner err", e.message));
  }
}

async function testPrtg() {
  const v = prtgFormValues();
  const err = validateSettings(dbFormValues(), v);
  const box = $("#prtg-test-result");
  if (err) { clear(box); box.appendChild(el("div", "banner err", err)); return; }
  try {
    const r = await apiJson("/admin/api/test/prtg", { method: "POST", body: v });
    renderTestResult(box, r);
  } catch (e) {
    clear(box);
    box.appendChild(el("div", "banner err", e.message));
  }
}

function buildSettingsPayload() {
  const db = dbFormValues();
  const prtg = prtgFormValues();
  return {
    database: db,
    prtg: prtg,
    server_port: parseInt($("#server-port").value, 10),
    log_level: $("#log-level").value,
  };
}

async function saveSettings(restart) {
  const payload = buildSettingsPayload();
  const err = validateSettings(payload.database, payload.prtg);
  const errBox = $("#save-error");
  const stateNode = $("#save-state");
  errBox.classList.add("hide");
  if (err) {
    errBox.textContent = err;
    errBox.classList.remove("hide");
    return;
  }
  stateNode.textContent = "Saving…";
  try {
    await apiJson("/admin/api/settings", { method: "PUT", body: payload });
    stateNode.textContent = "Saved. Restart the service to apply.";
    if (restart) {
      await apiJson("/admin/api/restart", { method: "POST", body: {} });
      const banner = $("#restart-banner");
      banner.textContent = "Restarting… the service will come back shortly.";
      banner.classList.remove("hide");
      pollAfterRestart();
    }
  } catch (e) {
    stateNode.textContent = "";
    errBox.textContent = e.message || "Save failed.";
    errBox.classList.remove("hide");
  }
}

function pollAfterRestart() {
  if (state.restartTimer) clearInterval(state.restartTimer);
  const started = Date.now();
  const stop = () => {
    clearInterval(state.restartTimer);
    state.restartTimer = null;
    $("#restart-banner").classList.add("hide");
  };
  state.restartTimer = setInterval(async () => {
    let res;
    try {
      res = await fetch("/admin/api/session", { credentials: "same-origin", cache: "no-store" });
    } catch (e) {
      // The service is still down: keep waiting, up to two minutes.
      if (Date.now() - started > 120000) {
        stop();
        showLogin();
      }
      return;
    }
    stop();
    if (res.ok) {
      loadOverview(true);
    } else {
      // Sessions live in memory: the restarted service asks to sign in again.
      showLogin();
    }
  }, 2000);
}

/* ---------- clients ---------- */

async function loadClientConfig() {
  const host = $("#client-host").value.trim();
  const q = host ? "?host=" + encodeURIComponent(host) : "";
  try {
    const r = await apiJson("/admin/api/client-config" + q);
    if (r.host) $("#client-host").value = r.host;
    $("#client-config").textContent = r.text || "";
  } catch (e) {
    $("#client-config").textContent = "Could not load client config: " + e.message;
  }
}

async function copyConfig() {
  const text = $("#client-config").textContent;
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
  } catch (e) {
    const range = document.createRange();
    range.selectNodeContents($("#client-config"));
    const sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(range);
  }
}

/* ---------- init ---------- */

async function init() {
  $("#login-form").addEventListener("submit", doLogin);
  $("#signout-btn").addEventListener("click", doLogout);
  $("#refresh-status").addEventListener("click", () => loadOverview());
  $("#test-db").addEventListener("click", testDb);
  $("#test-prtg").addEventListener("click", testPrtg);
  $("#save-btn").addEventListener("click", () => saveSettings(false));
  $("#save-restart-btn").addEventListener("click", () => saveSettings(true));
  $("#client-refresh").addEventListener("click", loadClientConfig);
  $("#client-copy").addEventListener("click", copyConfig);
  window.addEventListener("hashchange", () => setRoute(currentHash()));

  try {
    const res = await fetch("/admin/api/session", { credentials: "same-origin" });
    if (res.ok) {
      showApp();
      setRoute(currentHash());
    } else {
      showLogin();
    }
  } catch (e) {
    showLogin();
  }
}

document.addEventListener("DOMContentLoaded", init);
