package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osmonduu/sub-sync/internal/subsync"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds the application context given to us by Wails on startup.
// Store it so we can call Wails runtime functions like opening file dialogs.
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SyncResult is the outcome for a subtitle file.
// Used to serialize results to JSON and send back to
// Svelte frontend.
type SyncResult struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	OffsetMs   int64  `json:"offsetMs"`
	Error      string `json:"error"` // empty string means success
}

// SelectVideo opens a native file picker filtered to video formats
// and returns the selected path as a string.
func (a *App) SelectVideo() string {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select video file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Video files", Pattern: "*.mp4;*.mkv;*.avi;*.mov"},
		},
	})
	if err != nil {
		return ""
	}
	return path
}

// SelectSubtitles opens a native file picker that allows selecting
// multiple .ass files and returns a slice of the selected paths.
func (a *App) SelectSubtitles() []string {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select subtitle files",
		Filters: []runtime.FileFilter{
			{DisplayName: "ASS subtitles", Pattern: "*.ass"},
		},
	})
	if err != nil {
		return []string{}
	}
	return paths
}

// SelectOutputFolder opens a native folder picker and returns the selected path.
// If the user cancels, returns an empty string and the frontend will use the default.
func (a *App) SelectOutputFolder() string {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select output folder",
	})
	if err != nil {
		return ""
	}
	return path
}

// RunSync is the main entry point called by the frontend when the user clicks "Sync all". It:
//  1. Extracts audio from the video source once
//  2. Loops over each subtitle file, finds the best offset, saves the result
//  3. Returns a JSON array of SyncResult so the frontend can update the UI per file.
//
// outputFolder is optional. If empty, each synced file goes into a "synced subtitles"
// subfolder next to its input file
func (a *App) RunSync(videoPath string, subPaths []string, outputFolder string) string {
	resolution := 100 * time.Millisecond
	maxSearchDistance := 100 // 100 slots * 100ms = ±10 seconds search window

	results := make([]SyncResult, 0, len(subPaths))

	// Step 1: Extract audio from video file
	audioSamples, err := subsync.ExtractAudio(videoPath)
	if err != nil {
		// If audio extraction fails, the whole batch fails
		return marshalError(fmt.Sprintf("failed to extract audio from video source: %v", err))
	}
	audioTimeline := subsync.GenerateAudioTimeline(audioSamples, 16000, resolution)

	// Step 2: Process each subtitle file against the extracted audio
	for _, subPath := range subPaths {
		result := processSingle(subPath, audioTimeline, resolution, maxSearchDistance, outputFolder)
		results = append(results, result)
	}

	// Step 3: Return JSON array of results
	out, err := json.Marshal(results)
	if err != nil {
		return marshalError("failed to serialize results")
	}
	return string(out)
}

// processSingle handles one subtitle file parsing, aligning, and saving to new .ass file
func processSingle(subPath string, audioTimeline []bool, resolution time.Duration, maxSearchDistance int, outputFolder string) SyncResult {
	// Parse the subtitle file
	dialogueLines, rawLines, err := subsync.ParseAssFile(subPath)
	if err != nil {
		return SyncResult{
			InputPath: subPath,
			Error:     err.Error(),
		}
	}
	subTimeline := subsync.GenerateSubTimeline(dialogueLines, resolution)

	// Find best offset
	bestOffsetSlots, _ := subsync.FindBestOffset(audioTimeline, subTimeline, maxSearchDistance)
	finalOffset := time.Duration(bestOffsetSlots) * resolution

	// Determine the output path and save the synced file
	outputPath, err := resolveOutputPath(subPath, outputFolder)
	if err != nil {
		return SyncResult{InputPath: subPath, Error: err.Error()}
	}
	err = subsync.SaveSyncedAssFile(outputPath, rawLines, dialogueLines, finalOffset)
	if err != nil {
		return SyncResult{InputPath: subPath, Error: err.Error()}
	}

	return SyncResult{
		InputPath:  subPath,
		OutputPath: outputPath,
		OffsetMs:   finalOffset.Milliseconds(),
	}
}

// resolveOutputPath determines where the synced subtitle file should be saved.
// If outputFolder is set, it uses that path. Otherwise it creates a "synced_subtitles"
// folder next to the input file.
func resolveOutputPath(subPath string, outputFolder string) (string, error) {
	// Add the "_synced" suffix to the sub file name to differentiate the aligned subtitles
	base := filepath.Base(subPath)
	nameWithoutExt := strings.TrimSuffix(base, ".ass")
	outputName := nameWithoutExt + "_synced.ass"

	if outputFolder != "" {
		return filepath.Join(outputFolder, outputName), nil
	}

	// Create a "synced_subtitles" folder next to the input file if no path is provided
	inputDir := filepath.Dir(subPath)
	syncedDir := filepath.Join(inputDir, "synced_subtitles")
	err := os.MkdirAll(syncedDir, 0755)
	if err != nil {
		return "", err
	}
	return filepath.Join(syncedDir, outputName), nil
}

// marshalError is a helper to return a JSON error when something
// fails before we can start processing files.
func marshalError(msg string) string {
	// Only return a slice of one SyncResult contianing the error message
	result := []SyncResult{{Error: msg}}
	out, _ := json.Marshal(result)
	return string(out)
}
