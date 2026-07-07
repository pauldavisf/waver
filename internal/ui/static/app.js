const form = document.querySelector("#combine-form");
const projectPathInput = document.querySelector("#project-path");
const outputRootInput = document.querySelector("#output-root");
const selectProjectButton = document.querySelector("#select-project");
const selectOutputButton = document.querySelector("#select-output");
const button = document.querySelector("#run-button");
const recentPaths = document.querySelector("#recent-paths");
const resultPanel = document.querySelector("#result-panel");
const statusTitle = document.querySelector("#status-title");
const statusMessage = document.querySelector("#status-message");
const outputPanel = document.querySelector("#output-panel");
const outputPath = document.querySelector("#output-path");

const metrics = {
  tracks: document.querySelector("#metric-tracks"),
  sources: document.querySelector("#metric-sources"),
  silence: document.querySelector("#metric-silence"),
  segments: document.querySelector("#metric-segments"),
};

const storagePathKey = "waver.recentPaths";
const storageOutputKey = "waver.lastOutputRoot";

renderRecentPaths();
restoreOutputRoot();

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  const path = projectPathInput.value.trim();
  if (!path) {
    setStatus("error", "Остановлено", "Укажите папку проекта.");
    projectPathInput.focus();
    return;
  }

  const outputRoot = outputRootInput.value.trim();
  if (!outputRoot) {
    setStatus("error", "Остановлено", "Укажите папку для результата.");
    outputRootInput.focus();
    return;
  }

  setRunning();

  try {
    const response = await fetch("/api/combine", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ path, outputRoot }),
    });

    const payload = await response.json();
    if (!response.ok || !payload.ok) {
      throw new Error(payload.error || "Склейка не выполнена.");
    }

    saveRecentPath(path);
    localStorage.setItem(storageOutputKey, outputRoot);
    renderResult(payload.result);
  } catch (error) {
    setStatus("error", "Остановлено", error.message);
  } finally {
    setIdle();
  }
});

selectProjectButton.addEventListener("click", async () => {
  await pickDirectory(projectPathInput, "Выберите папку проекта");
});

selectOutputButton.addEventListener("click", async () => {
  await pickDirectory(outputRootInput, "Выберите папку результата");
});

function setControlsDisabled(disabled) {
  projectPathInput.disabled = disabled;
  outputRootInput.disabled = disabled;
  selectProjectButton.disabled = disabled;
  selectOutputButton.disabled = disabled;
  button.disabled = disabled;
}

function setRunning() {
  setControlsDisabled(true);
  button.textContent = "Склеиваю";
  outputPanel.hidden = true;
  setMetrics();
  setStatus("running", "Склеиваю", "Читаю паттерны и пишу мультитреки.");
}

function setIdle() {
  setControlsDisabled(false);
  button.textContent = "Склеить";
}

function renderResult(result) {
  setMetrics(result);
  outputPath.textContent = result.outputPath;
  outputPanel.hidden = false;

  const message = `${result.trackCount} дорожек, ${result.sourceCount} сегментов аудио, ${result.silenceCount} сегментов тишины.`;
  setStatus("success", "Готово", message);
}

function setMetrics(result = {}) {
  metrics.tracks.textContent = result.trackCount ?? 0;
  metrics.sources.textContent = result.sourceCount ?? 0;
  metrics.silence.textContent = result.silenceCount ?? 0;
  metrics.segments.textContent = result.segmentCount ?? 0;
}

function setStatus(state, title, message) {
  resultPanel.classList.remove("is-running", "is-success", "is-error");
  resultPanel.classList.add(`is-${state}`);
  statusTitle.textContent = title;
  statusMessage.textContent = message;
}

function getRecentPaths() {
  try {
    const raw = localStorage.getItem(storagePathKey);
    const parsed = JSON.parse(raw || "[]");
    return Array.isArray(parsed) ? parsed.filter(Boolean) : [];
  } catch {
    return [];
  }
}

function saveRecentPath(path) {
  const paths = getRecentPaths().filter((item) => item !== path);
  paths.unshift(path);
  localStorage.setItem(storagePathKey, JSON.stringify(paths.slice(0, 4)));
}

function restoreOutputRoot() {
  const storedOutputRoot = localStorage.getItem(storageOutputKey);
  outputRootInput.value = storedOutputRoot ? storedOutputRoot : "out";
}

function renderRecentPaths() {
  const paths = getRecentPaths();
  recentPaths.replaceChildren();
  recentPaths.hidden = paths.length === 0;

  for (const path of paths) {
    const item = document.createElement("button");
    item.type = "button";
    item.textContent = path;
    item.title = path;
    item.addEventListener("click", () => {
      projectPathInput.value = path;
      projectPathInput.focus();
    });
    recentPaths.append(item);
  }
}

async function pickDirectory(input, title) {
  const buttonForInput = input.id === "project-path" ? selectProjectButton : selectOutputButton;
  const initialButtonText = buttonForInput.textContent;
  buttonForInput.textContent = "Выбор...";
  buttonForInput.disabled = true;

  try {
    const response = await fetch("/api/select-directory", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ title }),
    });

    const payload = await response.json();
    if (!response.ok || !payload.ok) {
      if (payload.error !== "selection canceled") {
        setStatus("error", "Остановлено", payload.error || "Не удалось выбрать папку.");
      }
      return;
    }

    input.value = payload.path || "";
  } catch {
    setStatus("error", "Остановлено", "Не удалось выбрать папку.");
  } finally {
    buttonForInput.textContent = initialButtonText;
    buttonForInput.disabled = false;
  }
}
