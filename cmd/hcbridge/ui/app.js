/* Hardcover page. ES5 only (the Kindle's WebKit is old): no arrow
   functions, no fetch, no Promise. HC (port, key) comes from config.js,
   written by the daemon at each start. */
(function () {
  "use strict";
  var base = window.HC ? "http://127.0.0.1:" + HC.port + "/api/" : "";
  var key = window.HC ? HC.key : "";
  var timer = null;
  var pendingAct = null;

  function $(id) { return document.getElementById(id); }
  function show(id, on) {
    var el = $(id);
    var cls = el.className.replace(/\s*hidden/g, "");
    el.className = on ? cls : cls + " hidden";
  }
  // Change the page only when a value changes: each change redraws e-ink.
  function setText(id, text) {
    var el = $(id);
    if (el.textContent !== text) { el.textContent = text; }
  }

  function call(name, method, done) {
    if (!base) { done(null); return; }
    var xhr = new XMLHttpRequest();
    xhr.open(method, base + name + "?k=" + key, true);
    xhr.timeout = 95000;
    xhr.onreadystatechange = function () {
      if (xhr.readyState !== 4) { return; }
      var out = null;
      try { out = JSON.parse(xhr.responseText); } catch (e) { out = null; }
      done(xhr.status === 0 ? null : (out || { error: "HTTP " + xhr.status }));
    };
    xhr.send(null);
  }

  function message(text) {
    if (text) { setText("msg", text); }
    show("msg", !!text);
  }

  function render(s) {
    show("loading", false);
    if (!s) {
      show("off", true); show("signin", false); show("status", false); show("actions", false);
      schedule(30000);
      return;
    }
    show("off", false);
    setText("ver", s.version || "");
    var login = s.login || {};
    show("signin", !s.signedIn);
    show("status", s.signedIn);
    show("actions", true);
    var acts = document.querySelectorAll(".needs-signin");
    for (var i = 0; i < acts.length; i++) {
      acts[i].style.display = s.signedIn ? "" : "none";
    }
    if (!s.signedIn) {
      show("btnLogin", !login.active);
      show("code", !!(login.active && login.code));
      if (login.active && login.code) {
        setText("url", login.url);
        if ($("ucode").textContent !== login.code) {
          $("qr").src = base + "qr.png?k=" + key + "&c=" + login.code;
        }
        setText("ucode", login.code);
      }
      if (login.error) { message("Sign-in failed: " + login.error); }
    } else {
      setText("user", s.user || "...");
      setText("last", s.lastCheck || "not yet");
      setText("result", s.lastResult || "-");
      show("pendingRow", s.pending > 0);
      setText("pending", String(s.pending));
    }
    // Poll fast only while waiting for something.
    schedule(login.active || (s.signedIn && !s.user) ? 4000 : 30000);
  }

  function schedule(ms) {
    if (timer) { clearTimeout(timer); }
    timer = setTimeout(refresh, ms);
  }

  function refresh() { call("status", "GET", render); }

  function act(name) {
    message("Working...");
    show("checks", false);
    call(name, "POST", function (out) {
      if (!out) { message("No answer. Is sync running?"); refresh(); return; }
      if (out.error) { message("Error: " + out.error); refresh(); return; }
      if (name === "check") {
        message("");
        setText("checks", out.message || "");
        show("checks", true);
      } else {
        message(out.message || "");
      }
      if (name === "stop" || name === "uninstall") {
        show("actions", false); show("status", false); show("signin", false);
        if (timer) { clearTimeout(timer); }
        return;
      }
      refresh();
    });
  }

  function onAction(el) {
    var name = el.getAttribute("data-act");
    var text = el.getAttribute("data-confirm");
    if (!text) { act(name); return; }
    pendingAct = name;
    setText("confirmText", text);
    show("confirm", true);
    show("actions", false);
  }

  var buttons = document.querySelectorAll("[data-act]");
  for (var i = 0; i < buttons.length; i++) {
    (function (el) {
      el.addEventListener("click", function () { onAction(el); }, false);
    })(buttons[i]);
  }
  $("btnYes").addEventListener("click", function () {
    show("confirm", false); show("actions", true);
    if (pendingAct) { act(pendingAct); pendingAct = null; }
  }, false);
  $("btnNo").addEventListener("click", function () {
    show("confirm", false); show("actions", true); pendingAct = null;
  }, false);
  $("btnLogin").addEventListener("click", function () {
    message("Getting a sign-in code...");
    call("login", "POST", function (out) {
      message(out && out.error ? "Error: " + out.error : "");
      refresh();
    });
  }, false);
  $("btnCancel").addEventListener("click", function () {
    call("login-cancel", "POST", function () { message(""); refresh(); });
  }, false);

  refresh();
})();
