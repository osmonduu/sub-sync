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
	MinOffset  int64  `json:"minOffset"`
	MaxOffset  int64  `json:"maxOffset"`
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
func (a *App) RunSync(
	videoPath string,
	subPaths []string,
	outputFolder string,
	maxSearchDistanceMs int,
	segmentDurationMinutes int,
) string {
	resolution := 100 * time.Millisecond

	results := make([]SyncResult, 0, len(subPaths))

	runtime.EventsEmit(a.ctx, "sync:extracting-audio", true)
	// Extract audio from video file
	audioSamples, err := subsync.ExtractAudio(videoPath)
	if err != nil {
		// If audio extraction fails, the whole batch fails
		runtime.EventsEmit(a.ctx, "sync:extracting-audio", false)
		return marshalError(fmt.Sprintf("Failed to extract audio from video source: %v", err))
	}
	audioTimeline := subsync.GenerateAudioTimeline(audioSamples, 16000, resolution)
	runtime.EventsEmit(a.ctx, "sync:extracting-audio", false)

	// Process each subtitle file against the extracted audio
	maxSearchDistanceSlots := maxSearchDistanceMs / 100 	// convert from milliseconds to number of 100 ms slots
	segmentDuration := time.Duration(segmentDurationMinutes) * time.Minute
	for _, subPath := range subPaths {
		result := processSingle(subPath, outputFolder, audioTimeline, resolution, maxSearchDistanceSlots, segmentDuration)
		results = append(results, result)

		// For every synced subtitle file, send an event to frontend via Wails
		resultJSON, err := json.Marshal(result)
		if err != nil {
			fmt.Println("failed to serialize result: ", err.Error())
			continue
		}
		runtime.EventsEmit(a.ctx, "sync:file-complete", string(resultJSON))
	}

	// Return JSON array of results
	out, err := json.Marshal(results)
	if err != nil {
		return marshalError("failed to serialize results")
	}
	return string(out)
}

// processSingle handles one subtitle file's parsing, aligning, and saving to new .ass file
func processSingle(
	subPath string, 
	outputFolder string,
	audioTimeline []bool, 
	resolution time.Duration, 
	maxSearchDistanceSlots int, 
	segmentDuration time.Duration,
	) SyncResult {
	// Parse the subtitle file
	dialogueLines, rawLines, err := subsync.ParseAssFile(subPath)
	if err != nil {
		return SyncResult{
			InputPath: subPath,
			Error:     err.Error(),
		}
	}

	// Find the best offset for each dialogue line
	alignedLines, minOffset, maxOffset := subsync.AlignPiecewise(
		dialogueLines, 
		audioTimeline, 
		resolution, 
		maxSearchDistanceSlots,
		segmentDuration,
	)

	// Determine the output path and save the synced file
	outputPath, err := resolveOutputPath(subPath, outputFolder)
	if err != nil {
		return SyncResult{InputPath: subPath, Error: err.Error()}
	}
	err = subsync.SaveSyncedAssFile(outputPath, rawLines, alignedLines)
	if err != nil {
		return SyncResult{InputPath: subPath, Error: err.Error()}
	}

	return SyncResult{
		InputPath:  subPath,
		OutputPath: outputPath,
		MinOffset:  minOffset.Milliseconds(),
		MaxOffset:  maxOffset.Milliseconds(),
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
