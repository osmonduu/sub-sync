<script>
  // @ts-nocheck
  import {
    RunSync,
    SelectOutputFolder,
    SelectSubtitles,
    SelectVideo,
  } from "../wailsjs/go/main/App.js";

  let videoPath = ""; // path to selected video
  let subPaths = []; // list of selected .ass file paths
  let outputFolder = ""; // optional custom output folder
  let results = []; // array of SyncResult from RunSync
  let running = false; // whether sync is currently running
  let progress = 0; // 0-100 for the progress bar
  let finishedAt = ""; // temp variable to store time when the sync engine finishes

  // handleSelectVideo
  async function handleSelectVideo() {
    const path = await SelectVideo();
    if (path !== "") {
      videoPath = path;
    }
  }

  // handleSelectSubtitles
  async function handleSelectSubtitles() {
    const paths = await SelectSubtitles();
    if (paths.length > 0) {
      subPaths = [...subPaths, ...paths];
    }
  }

  // handleSelectOutputFolder
  async function handleSelectOutputFolder() {
    const outDirPath = await SelectOutputFolder();
    if (outDirPath !== "") {
      outputFolder = outDirPath;
    }
  }

  // handleRunSync
  async function handleRunSync() {
    running = true;
    results = []; // clear previous run output
    // Check if videoPath or subPath is populated before running the sync engine
    if (videoPath === "" || subPaths.length === 0) {
      running = false;
      return;
    }
    try {
      const syncResults = await RunSync(videoPath, subPaths, outputFolder);
      results = JSON.parse(syncResults);
      finishedAt = new Date().toLocaleTimeString("en-US");
    } finally {
      running = false;
    }
  }

  // handleClear
  function handleClear() {
    videoPath = "";
    subPaths = [];
    outputFolder = "";
    results = [];
    progress = 0;
    finishedAt = "";
  }
</script>

<!-- Markup -->
<div class="app-container">
  <div class="video-row">
    <p>Video</p>
    {#if videoPath !== ""}
      <span class="path-value">{videoPath}</span>
    {:else}
      <span class="path-value path-value-muted">No file selected</span>
    {/if}
    <button class="btn-small" on:click={handleSelectVideo}>Browse</button>
  </div>

  <div class="output-folder-row">
    <p>Output folder</p>
    {#if outputFolder !== ""}
      <span class="path-value">{outputFolder}</span>
    {:else}
      <span class="path-value path-value-muted"
        >Default - synced_subtitles/ next to each input file</span
      >
    {/if}
    <button class="btn-small" on:click={handleSelectOutputFolder}>Browse</button
    >
  </div>

  <div class="panels">
    <div class="input-panel">
      <div class="panel-header">
        <p>INPUT SUBTITLES</p>
        <button class="btn-small" on:click={handleSelectSubtitles}
          >Add files</button
        >
      </div>
      <div class="scrollable-list">
        {#each subPaths as path}
          <div class="file-item">
            <i class="ti ti-file-text"></i>
            <span class="file-name">{path.split("/").pop()}</span>
            <button
              class="remove-icon"
              aria-label="Remove {path.split('/').pop()}"
              on:click={() => (subPaths = subPaths.filter((p) => p !== path))}
            >
              <i class="ti ti-x"></i>
            </button>
          </div>
        {/each}
      </div>
    </div>

    <div class="arrow">
      <i class="ti ti-arrow-right"></i>
    </div>

    <div class="output-panel">
      <div class="panel-header">
        <p>OUTPUT</p>
      </div>
      <div class="scrollable-list">
        {#each subPaths as path}
          {@const result = results.find((r) => r.inputPath === path)}
          <div class="file-item">
            <i class="ti ti-file-text"></i>
            <span class="file-name">{path.split("/").pop()}</span>
            {#if result === undefined}
              <span>waiting...</span>
            {:else if result.error !== ""}
              <span class="log-error">error syncing</span>
            {:else}
              <span class="log-success">
                {result.offsetMs > 0 ? "+" : ""}{result.offsetMs}ms
              </span>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  </div>

  <div class="progress-bar">
    {#if running}
      <p>Syncing...</p>
    {:else if results.length > 0}
      <p>{results.length} / {subPaths.length} complete</p>
    {:else}
      <p>Ready</p>
    {/if}
  </div>

  <div class="log">
    {#each results as result}
      <div class="log-line">
        <span class="log-time">{finishedAt}</span>
        <span>{result.inputPath.split("/").pop()}</span>
        <span class={result.error === "" ? "log-success" : "log-error"}>
          {result.error === ""
            ? result.offsetMs + "ms offset applied"
            : result.error}
        </span>
      </div>
    {/each}
  </div>

  <div class="actions">
    <button class="btn-cancel" on:click={handleClear}>Clear</button>
    <button class="btn-sync" on:click={handleRunSync}>Sync all</button>
  </div>
</div>

<!-- CSS -->
<style>
  .app-container {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
  }

  p {
    margin: 0;
  }

  .video-row,
  .output-folder-row {
    display: flex;
    align-items: center;
    gap: 10px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 8px 14px;
  }

  .path-value {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .path-value-muted {
    color: var(--color-text-muted);
  }

  .btn-small {
    display: flex;
    align-items: center;
    gap: 4px;
    /* background: var(--color-surface-2); */
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 4px 8px;
  }

  .panels {
    display: grid;
    grid-template-columns: 1fr 32px 1fr;
    gap: 10px;
  }

  .input-panel,
  .output-panel {
    display: flex;
    flex-direction: column;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    overflow: hidden;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: var(--color-surface-2);
    border-bottom: 1px solid var(--color-border);
    padding: 6px 8px;
    min-height: 24px;
  }

  .scrollable-list {
    flex: 1;
    overflow-y: auto;
    max-height: 250px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 8px 10px;
  }

  .file-item {
    display: flex;
    align-items: center;
    gap: 5px;
    background: var(--color-surface-2);
    border-radius: 3px;
    padding: 3px 8px;
  }

  .file-name {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-item i {
    font-size: 14px;
    color: var(--color-text-muted);
  }

  .remove-icon {
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    color: var(--color-text-muted);
    font-size: 14px;
    display: flex;
    align-items: center;
  }

  .remove-icon:hover i {
    color: var(--color-danger);
  }

  .arrow {
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    color: var(--color-text-muted);
  }

  .progress-bar {
    display: flex;
    align-items: center;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 8px 14px;
  }

  .log {
    display: flex;
    justify-content: flex-start;
    flex-direction: column;
    gap: 6px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 10px 12px;
  }

  .log-line {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .log-time {
    color: var(--color-text-muted);
    font-size: 12px;
  }

  .log-success {
    color: var(--color-accent);
  }

  .log-error {
    color: var(--color-danger);
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }

  .btn-cancel {
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 6px 10px;
  }

  .btn-sync {
    background: var(--color-accent);
    color: var(--color-text);
    border: none;
    border-radius: 6px;
    padding: 6px 10px;
  }
</style>
