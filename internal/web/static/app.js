const state = {
  panes: [],
  selectedPaneId: "",
  loading: false,
  pollTimer: 0,
};

const paneList = document.querySelector("#pane-list");
const refreshPanesButton = document.querySelector("#refresh-panes");
const refreshContentButton = document.querySelector("#refresh-content");
const autoRefresh = document.querySelector("#auto-refresh");
const paneTitle = document.querySelector("#pane-title");
const capture = document.querySelector("#capture");
const promptForm = document.querySelector("#prompt-form");
const promptInput = document.querySelector("#prompt");
const sendButton = document.querySelector("#send");
const statusText = document.querySelector("#status");

function paneURL(path) {
  return `/api/codex/panes/${encodeURIComponent(state.selectedPaneId)}${path}`;
}

function setStatus(message, isError = false) {
  statusText.textContent = message;
  statusText.classList.toggle("error", isError);
}

function setControlsEnabled(enabled) {
  refreshContentButton.disabled = !enabled;
  promptInput.disabled = !enabled;
  sendButton.disabled = !enabled;
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

    if (!state.panes.some((pane) => pane.paneId === state.selectedPaneId)) {
      state.selectedPaneId = state.panes[0]?.paneId || "";
    }

    renderPanes();
    updateSelection();

    if (state.selectedPaneId) {
      await loadCapture();
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
  updateSelection();
  await loadCapture();
}

async function loadCapture() {
  if (!state.selectedPaneId || state.loading) {
    return;
  }

  state.loading = true;

  try {
    const data = await requestJSON(paneURL("/content"));
    capture.textContent = data.content || "";
    capture.scrollTop = capture.scrollHeight;
    setStatus(`Updated ${new Date().toLocaleTimeString()}`);
  } catch (error) {
    setStatus(error.message, true);
  } finally {
    state.loading = false;
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
    await requestJSON(paneURL("/prompt"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ prompt }),
    });

    promptInput.value = "";
    setStatus("Sent");
    await loadCapture();
  } catch (error) {
    setStatus(error.message, true);
  } finally {
    sendButton.disabled = false;
  }
}

function resetPolling() {
  window.clearInterval(state.pollTimer);

  if (autoRefresh.checked) {
    state.pollTimer = window.setInterval(loadCapture, 2000);
  }
}

refreshPanesButton.addEventListener("click", loadPanes);
refreshContentButton.addEventListener("click", loadCapture);
autoRefresh.addEventListener("change", resetPolling);
promptForm.addEventListener("submit", sendPrompt);

promptInput.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
    promptForm.requestSubmit();
  }
});

setControlsEnabled(false);
resetPolling();
loadPanes();
