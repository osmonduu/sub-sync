package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/osmonduu/sub-sync/internal/subsync"
)

func main() {
	// Define and set the variables that are passed in by the user
	subPath := flag.String("sub", "", "Path to input. ass subtitle file")
	videoPath := flag.String("video", "", "Path to input video file")
	maxSearchDistanceMs := flag.Int(
		"search", 
		3000, 
		"The range, in milliseconds, that the engine searches to find the best offset (default: 3000 ms). " + 
		"Larger values will result in more inaccurate results.",
	)
	segmentDuration := flag.Int(
		"segment",
		5,
		"Length of a segment in minutes (default: 5 min). The audio is broken up into segments to find each segment's best offset. " +
		"Smaller values will result in more inaccurate offsets.",
	)
	flag.Parse()

	if *subPath == "" || *videoPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: sub-sync -sub <file.ass> -video <file.mp4>")
		os.Exit(1)
	}

	fmt.Println("Starting alignment engine...")

	resolution := 100 * time.Millisecond
	maxSearchDistanceSlots := *maxSearchDistanceMs / 100 // search area in terms of slots (3000 / 100ms = 30 slots)
	segmentDurationMinutes := time.Duration(*segmentDuration) * time.Minute

	// Extract audio and run the VAD (voice activity detection)
	fmt.Println("[1/4] Decoding video audio and running VAD...")
	audioSamples, err := subsync.ExtractAudio(*videoPath)
	if err != nil {
		fmt.Printf("Error extracting audio: %v\n", err)
		return
	}
	audioTimeline := subsync.GenerateAudioTimeline(audioSamples, 16000, resolution)

	// Parse subtitles and create the boolean subtitle timeline
	fmt.Println("[2/4] Parsing subtitle file...")
	dialogueLines, rawLines, err := subsync.ParseAssFile(*subPath)
	if err != nil {
		fmt.Printf("Error parsing ASS file: %v\n", err)
		return
	}

	// Find the best offset using sliding alignment and interpolation
	fmt.Println("[3/4] Calculating subtitle offset based on video audio...")
	alignedLines, minOffset, maxOffset := subsync.AlignPiecewise(
		dialogueLines, 
		audioTimeline, 
		resolution, 
		maxSearchDistanceSlots,
		segmentDurationMinutes,
	)

	fmt.Println("\n====================================")
	fmt.Printf("ALIGNMENT MATCH COMPLETED:\n")
	if minOffset == maxOffset {
		fmt.Printf("Calculated offset: %d ms", minOffset.Milliseconds())
	} else {
		fmt.Printf("Calculated offset range: %d ms to %d ms", minOffset.Milliseconds(), maxOffset.Milliseconds())
	}
	fmt.Println("\n====================================")

	// Apply offset and save to new file
	outputPath := strings.TrimSuffix(*subPath, ".ass") + "_synced.ass"
	fmt.Printf("[4/4] Exporting modified subtitles to: %s\n", outputPath)

	err = subsync.SaveSyncedAssFile(outputPath, rawLines, alignedLines)
	if err != nil {
		fmt.Printf("Output file error: %v\n", err)
		return
	}

	fmt.Println("\nProcess complete! Subtitles successfully realigned.")
}
