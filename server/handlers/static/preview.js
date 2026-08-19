(function () {
  const dialog = document.getElementById("reply-dialog");
  const openBtn = document.getElementById("open-reply");
  if (!dialog || !openBtn) return;

  function openReply() {
    if (typeof dialog.showModal === "function") {
      if (!dialog.open) dialog.showModal();
    } else {
      dialog.setAttribute("open", "");
    }
    const input = document.getElementById("body-input");
    if (input) input.focus();
  }

  function closeReply() {
    if (typeof dialog.close === "function" && dialog.open) dialog.close();
    else dialog.removeAttribute("open");
  }

  openBtn.addEventListener("click", openReply);
  dialog.querySelectorAll(".reply-close").forEach((btn) => {
    btn.addEventListener("click", closeReply);
  });
  dialog.addEventListener("click", (e) => {
    if (e.target === dialog) closeReply();
  });
  dialog.addEventListener("cancel", (e) => {
    e.preventDefault();
    closeReply();
  });
  if (dialog.dataset.open === "1") openReply();
})();

(function () {
  const input = document.getElementById("body-input");
  const frame = document.getElementById("preview-frame");
  const status = document.getElementById("preview-status");
  if (!input || !frame) return;

  let inflight = null;
  let pending = false;
  let timer = null;

  async function refresh() {
    if (inflight) {
      pending = true;
      return;
    }
    if (status) status.textContent = "Actualizando…";
    const body = new URLSearchParams({ body: input.value });
    inflight = fetch("/admin/preview", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body,
      credentials: "same-origin",
    })
      .then((r) => (r.ok ? r.text() : Promise.reject(r.status)))
      .then((html) => {
        frame.srcdoc = html;
        if (status) status.textContent = "";
      })
      .catch(() => {
        if (status) status.textContent = "Error";
      })
      .finally(() => {
        inflight = null;
        if (pending) {
          pending = false;
          refresh();
        }
      });
  }

  function debouncedRefresh() {
    clearTimeout(timer);
    timer = setTimeout(refresh, 250);
  }

  input.addEventListener("input", debouncedRefresh);
  refresh();

  // Tab switcher (mobile)
  const tabs = document.querySelectorAll(".split-tabs button");
  const panes = document.querySelectorAll(".split .pane");
  tabs.forEach((btn) => {
    btn.addEventListener("click", () => {
      tabs.forEach((b) => b.classList.toggle("active", b === btn));
      const target = btn.dataset.pane;
      panes.forEach((p) => p.classList.toggle("active", p.classList.contains("pane-" + target)));
    });
  });
})();
