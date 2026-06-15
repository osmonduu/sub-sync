package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	// Define command-line flags to use SabotageAssFile
	sabotageOffset := flag.Int("sabotage", 0,
		"Specify an integer in seconds to intentionally sabotage (add offset to every subtitle) a .ass file")
	flag.Parse()

	// Grab the ramining non-flag arguments (the input and output paths)
	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: go run . [-sabotage seconds] <input_subtitles> <output_path>")
		os.Exit(1)
	}

	// If user passed a sabotage flag, run that instead of the sync engine
	if *sabotageOffset != 0 {
		fmt.Printf("Sabotaging subtitles! Adding %ds latency ...\n", *sabotageOffset)
		SabotageAssFile(args[0], args[1], *sabotageOffset)
		return // Exit early so it doesn't try to run the sync engine
	}

	// Run normal sync logic
	fmt.Println("Starting alignment sync engine...")
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <subtitles.ass> <video_file>")
		return
	}
	subPath := os.Args[1]
	videoPath := os.Args[2]

	resolution := 100 * time.Millisecond
	maxSearchDistance := 100 // Look 10 second forward/backward (100 slots * 100 ms)

	// Parse subtitles and create the boolean subtitle timeline
	fmt.Println("[1/4] Parsing subtitle file into memory...")
	dialogueLines, rawLines, err := ParseAssFile(subPath)
	if err != nil {
		fmt.Printf("Error parsing ASS file: %v\n", err)
		return
	}
	subTimeline := GenerateSubTimeline(dialogueLines, resolution)

	// Extract audio and run the VAD (voice activity detection)
	fmt.Println("[2/4] Decoding video audio and running VAD...")
	audioSamples, err := ExtractAudio(videoPath)
	if err != nil {
		fmt.Printf("Error extracting audio: %v\n", err)
		return
	}
	audioTimeline := GetVoiceActivity(audioSamples, 16000, resolution)

	// Find the best match using sliding alignment
	fmt.Println("[3/4] Calculating subtitle offset based on video audio...")
	bestOffsetSlots, confidence := FindBestOffset(audioTimeline, subTimeline, maxSearchDistance)

	// Convert slots back into milliseconds
	finalOffsetTime := time.Duration(bestOffsetSlots) * resolution

	fmt.Println("\n====================================")
	fmt.Printf("ALIGNMENT MATCH COMPLETED:\n")
	fmt.Printf("Calculated shift: %v (Slots: %+d)\n", finalOffsetTime, bestOffsetSlots)
	fmt.Printf("Confidence score: %.2f%%\n", confidence*100)
	fmt.Println("\n====================================")

	// Apply offset and save to new file
	outputPath := strings.TrimSuffix(subPath, ".ass") + "_synced.ass"
	fmt.Printf("[4/4] Exporting modified subtitles to: %s\n", outputPath)

	err = SaveSyncedAssFile(outputPath, rawLines, dialogueLines, finalOffsetTime)
	if err != nil {
		fmt.Printf("Output file error: %v\n", err)
		return
	}

	fmt.Println("\nProcess complete! Subtitles successfully realigned.")
}
