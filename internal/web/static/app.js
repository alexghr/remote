const state = {
  panes: [],
  selectedPaneId: "",
  loading: false,
  pollTimer: 0,
  lastCaptureContent: "",
  capturePinnedToBottom: true,
};

const bottomStickTolerance = 24;

const paneList = document.querySelector("#pane-list");
const refreshPanesButton = document.querySelector("#refresh-panes");
const refreshContentButton = document.querySelector("#refresh-content");
const autoRefresh = document.querySelector("#auto-refresh");
const paneTitle = document.querySelector("#pane-title");
const capture = document.querySelector("#capture");
const keyControlButtons = document.querySelectorAll("[data-key]");
const promptForm = document.querySelector("#prompt-form");
const promptInput = document.querySelector("#prompt");
const sendButton = document.querySelector("#send");
const statusText = document.querySelector("#status");

function paneURL(path) {
  return `/api/codex/panes/${encodeURIComponent(state.selectedPaneId)}${path}`;
}

function paneIdFromURL() {
  return new URL(window.location.href).searchParams.get("pane") || "";
}

function syncPaneIdToURL() {
  const url = new URL(window.location.href);

  if (state.selectedPaneId) {
    url.searchParams.set("pane", state.selectedPaneId);
  } else {
    url.searchParams.delete("pane");
  }

  window.history.replaceState(null, "", url);
}

function setStatus(message, isError = false) {
  statusText.textContent = message;
  statusText.classList.toggle("error", isError);
}

function setControlsEnabled(enabled) {
  refreshContentButton.disabled = !enabled;
  for (const button of keyControlButtons) {
    button.disabled = !enabled;
  }
  promptInput.disabled = !enabled;
  sendButton.disabled = !enabled;
}

function captureIsAtBottom() {
  return capture.scrollHeight - capture.scrollTop - capture.clientHeight <= bottomStickTolerance;
}

function rememberCaptureScroll() {
  state.capturePinnedToBottom = captureIsAtBottom();
}

function scrollCaptureToBottom() {
  capture.scrollTop = capture.scrollHeight;
  state.capturePinnedToBottom = true;
}

function renderCaptureContent(content) {
  const fragment = document.createDocumentFragment();
  const lines = content.split("\n");

  lines.forEach((line, index) => {
    const boxContent = line.match(/^\s*[│┃]\s?(.*?)\s*[│┃]\s*$/u);
    const labeledRule = line.match(/^(.*?\S)\s+[─━-]{20,}\s*$/u);

    if (/^\s*-{20,}\s*$/.test(line) || /^[\s╭╮╰╯┌┐└┘─━-]{20,}\s*$/u.test(line)) {
      const rule = document.createElement("span");
      rule.className = "codex-rule";
      rule.setAttribute("aria-hidden", "true");
      fragment.append(rule);
    } else if (labeledRule) {
      const row = document.createElement("span");
      row.className = "codex-labeled-rule";

      const label = document.createElement("span");
      label.className = "codex-rule-label";
      label.textContent = labeledRule[1];

      const rule = document.createElement("span");
      rule.className = "codex-rule";
      rule.setAttribute("aria-hidden", "true");

      row.append(label, rule);
      fragment.append(row);
    } else if (boxContent) {
      const chromeLine = document.createElement("span");
      chromeLine.className = "codex-chrome-line";
      chromeLine.textContent = boxContent[1].trimEnd();
      fragment.append(chromeLine);
    } else {
      fragment.append(document.createTextNode(line));
    }

    if (index < lines.length - 1) {
      fragment.append(document.createTextNode("\n"));
    }
  });

  capture.replaceChildren(fragment);
}

function selectionTouchesCapture() {
  const selection = window.getSelection();
  if (!selection || selection.isCollapsed || selection.rangeCount === 0) {
    return false;
  }

  for (let i = 0; i < selection.rangeCount; i++) {
    const range = selection.getRangeAt(i);
    if (
      capture.contains(range.startContainer) ||
      capture.contains(range.endContainer) ||
      range.intersectsNode(capture)
    ) {
      return true;
    }
  }

  return false;
}

async function requestJSON(url, options) {
  const response = await fetch(url, options);
  const body = await response.json().catch(() => ({}));

  if (!response.ok) {
    throw new Error(body.error || `HTTP ${response.status}`);
  }

  return body;
}

function renderPanes() {
  paneList.replaceChildren();

  if (state.panes.length === 0) {
    const empty = document.createElement("p");
    empty.className = "pane-meta";
    empty.textContent = "No Codex panes found";
    paneList.append(empty);
    return;
  }

  for (const pane of state.panes) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "pane-button";
    button.setAttribute("role", "option");
    button.setAttribute("aria-selected", pane.paneId === state.selectedPaneId ? "true" : "false");

    const id = document.createElement("span");
    id.className = "pane-id";
    id.textContent = pane.paneId;

    const meta = document.createElement("span");
    meta.className = "pane-meta";
    meta.textContent = `PID ${pane.pid}`;

    button.append(id, meta);
    button.addEventListener("click", () => selectPane(pane.paneId));
    paneList.append(button);
  }
}

async function loadPanes() {
  setStatus("Loading panes...");

  try {
    const data = await requestJSON("/api/codex/panes");
    state.panes = data.panes || [];

    const previousPaneId = state.selectedPaneId;
    const urlPaneId = paneIdFromURL();
    const urlPaneExists = state.panes.some((pane) => pane.paneId === urlPaneId);

    if (urlPaneExists) {
      state.selectedPaneId = urlPaneId;
    }

    if (!state.panes.some((pane) => pane.paneId === state.selectedPaneId)) {
      state.selectedPaneId = state.panes[0]?.paneId || "";
    }

    if (state.selectedPaneId !== previousPaneId) {
      state.lastCaptureContent = "";
      state.capturePinnedToBottom = true;
    }
    syncPaneIdToURL();

    renderPanes();
    updateSelection();

    if (state.selectedPaneId) {
      await loadCapture({ scrollToBottom: true });
    } else {
      capture.textContent = "No Codex panes found.";
      setStatus("");
    }
  } catch (error) {
    capture.textContent = "Failed to load panes.";
    setStatus(error.message, true);
  }
}

function updateSelection() {
  paneTitle.textContent = state.selectedPaneId || "No pane selected";
  setControlsEnabled(Boolean(state.selectedPaneId));
  renderPanes();
}

async function selectPane(paneId) {
  if (paneId === state.selectedPaneId) {
    return;
  }

  state.selectedPaneId = paneId;
  state.lastCaptureContent = "";
  state.capturePinnedToBottom = true;
  syncPaneIdToURL();
  updateSelection();
  await loadCapture({ scrollToBottom: true });
}

async function loadCapture(options = {}) {
  if (!state.selectedPaneId || state.loading) {
    return;
  }

  if (options.skipWhileSelecting && selectionTouchesCapture()) {
    return;
  }

  state.loading = true;

  try {
    const data = await requestJSON(paneURL("/content"));
    const content = data.content || "";
    const shouldStickToBottom = options.scrollToBottom || state.capturePinnedToBottom || captureIsAtBottom();
    const previousScrollTop = capture.scrollTop;

    if (state.lastCaptureContent !== content) {
      renderCaptureContent(content);
      state.lastCaptureContent = content;
    }

    if (shouldStickToBottom) {
      scrollCaptureToBottom();
    } else {
      capture.scrollTop = previousScrollTop;
      rememberCaptureScroll();
    }

    setStatus(`Updated ${new Date().toLocaleTimeString()}`);
  } catch (error) {
    setStatus(error.message, true);
  } finally {
    state.loading = false;
  }
}

async function sendRawPrompt(prompt) {
  await requestJSON(paneURL("/prompt"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ prompt }),
  });
}

async function sendKeys(keys) {
  if (!state.selectedPaneId) {
    return;
  }

  setStatus("Sending keys...");

  try {
    await requestJSON(paneURL("/keys"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ keys }),
    });
    setStatus("Keys sent");
    await loadCapture({ scrollToBottom: true });
  } catch (error) {
    setStatus(error.message, true);
  }
}

async function sendPrompt(event) {
  event.preventDefault();

  const prompt = promptInput.value.trim();
  if (!prompt || !state.selectedPaneId) {
    return;
  }

  sendButton.disabled = true;
  setStatus("Sending...");

  try {
    await sendRawPrompt(prompt);

    promptInput.value = "";
    setStatus("Sent");
    await loadCapture({ scrollToBottom: true });
  } catch (error) {
    setStatus(error.message, true);
  } finally {
    sendButton.disabled = false;
  }
}

function resetPolling() {
  window.clearInterval(state.pollTimer);

  if (autoRefresh.checked) {
    state.pollTimer = window.setInterval(() => loadCapture({ skipWhileSelecting: true }), 2000);
  }
}

refreshPanesButton.addEventListener("click", loadPanes);
refreshContentButton.addEventListener("click", loadCapture);
autoRefresh.addEventListener("change", resetPolling);
promptForm.addEventListener("submit", sendPrompt);
capture.addEventListener("scroll", rememberCaptureScroll);

for (const button of keyControlButtons) {
  button.addEventListener("click", () => sendKeys([button.dataset.key]));
}

promptInput.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
    promptForm.requestSubmit();
  }
});

setControlsEnabled(false);
resetPolling();
loadPanes();
