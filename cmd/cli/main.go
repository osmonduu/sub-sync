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
	subPath := flag.String("sub", "", "Path to input. ass subtitle file")
	videoPath := flag.String("video", "", "Path to input video file")
	flag.Parse()

	if *subPath == "" || *videoPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: sub-sync -sub <file.ass> -video <file.mp4>")
		os.Exit(1)
	}


	fmt.Println("Starting alignment engine...")

	resolution := 100 * time.Millisecond
	maxSearchDistance := 100 // Look 10 seconds forward/backward (100 slots * 100 ms)

	// Parse subtitles and create the boolean subtitle timeline
	fmt.Println("[1/4] Parsing subtitle file into memory...")
	dialogueLines, rawLines, err := subsync.ParseAssFile(*subPath)
	if err != nil {
		fmt.Printf("Error parsing ASS file: %v\n", err)
		return
	}
	subTimeline := subsync.GenerateSubTimeline(dialogueLines, resolution)

	// Extract audio and run the VAD (voice activity detection)
	fmt.Println("[2/4] Decoding video audio and running VAD...")
	audioSamples, err := subsync.ExtractAudio(*videoPath)
	if err != nil {
		fmt.Printf("Error extracting audio: %v\n", err)
		return
	}
	audioTimeline := subsync.GenerateAudioTimeline(audioSamples, 16000, resolution)

	// Find the best match using sliding alignment
	fmt.Println("[3/4] Calculating subtitle offset based on video audio...")
	bestOffsetSlots, confidence := subsync.FindBestOffset(audioTimeline, subTimeline, maxSearchDistance)

	// Convert slots back into milliseconds
	finalOffsetTime := time.Duration(bestOffsetSlots) * resolution

	fmt.Println("\n====================================")
	fmt.Printf("ALIGNMENT MATCH COMPLETED:\n")
	fmt.Printf("Calculated shift: %v (Slots: %+d)\n", finalOffsetTime, bestOffsetSlots)
	fmt.Printf("Confidence score: %.2f%%\n", confidence*100)
	fmt.Println("\n====================================")

	// Apply offset and save to new file
	outputPath := strings.TrimSuffix(*subPath, ".ass") + "_synced.ass"
	fmt.Printf("[4/4] Exporting modified subtitles to: %s\n", outputPath)

	err = subsync.SaveSyncedAssFile(outputPath, rawLines, dialogueLines, finalOffsetTime)
	if err != nil {
		fmt.Printf("Output file error: %v\n", err)
		return
	}

	fmt.Println("\nProcess complete! Subtitles successfully realigned.")
}
