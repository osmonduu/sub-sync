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
<div class="video-row">
  <p>Video</p>
  {#if videoPath !== ""}
    <span>{videoPath}</span>
  {:else}
    <span>No file selected</span>
  {/if}
  <button on:click={handleSelectVideo}>Browse</button>
</div>

<div class="output-folder-row">
  <p>Output folder</p>
  {#if outputFolder !== ""}
    <span>{outputFolder}</span>
  {:else}
    <span>Default - synced_subtitles/ next to each input file</span>
  {/if}
  <button on:click={handleSelectOutputFolder}>Browse</button>
</div>

<div class="panels">
  <div class="input-panel">
    <div class="panel-header">
      <p>Input subtitles</p>
      <button on:click={handleSelectSubtitles}>Add files</button>
    </div>
    <div class="scrollable-list">
      {#each subPaths as path}
        <div>
          <i class="ti ti-file-text"></i>
          <span>{path.split("/").pop()}</span>
          <i
            class="ti ti-x"
            on:click={() => (subPaths = subPaths.filter((p) => p !== path))}
          ></i>
        </div>
      {/each}
    </div>
    <div class="drop-hint">Drop .ass files here</div>
  </div>

  <div class="arrow">
    <i class="ti ti-arrow-right"></i>
  </div>

  <div class="output-panel">
    <div class="panel-header">
      <p>Output</p>
    </div>
    <div class="scrollable-list">
      {#each subPaths as path}
        {@const result = results.find((r) => r.inputPath === path)}
        <div>
          <i class="ti ti-file-text"></i>
          <span>{path.split("/").pop()}</span>
          {#if result === undefined}
            <span>waiting...</span>
          {:else if result.error !== ""}
            <span>error syncing</span>
          {:else}
            <span>{result.offsetMs > 0 ? "+" : ""}{result.offsetMs}ms</span>
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
    <div>
      <span>{finishedAt}</span>
      <span>{result.inputPath.split("/").pop()}</span>
      <span>{result.error === "" ? result.offsetMs + "ms" : result.error}</span>
    </div>
  {/each}
</div>

<div class="actions">
  <button on:click={handleClear}>Clear</button>
  <button on:click={handleRunSync}>Sync all</button>
</div>

<!-- CSS -->
<style>
</style>
