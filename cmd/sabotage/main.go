package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/osmonduu/sub-sync/internal/subsync"
)

func main() {
	input := flag.String("input", "", "Path to input .ass file")
	output := flag.String("output", "", "Name for ouput .ass file")
	offset := flag.Int("offset", 0, "Offset in seconds to apply")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: sabotage -input <file.ass> -output <file.ass> -offset <seconds>")
		os.Exit(1)
	}

	sabotageASSFile(*input, *output, *offset)
}

// sabotageAssFile applies an offset to a .ass (Advanced Sub Station Alpha) file.
func sabotageASSFile(inputPath, outputPath string, offsetInSeconds int) {
	// Use parser to load lines
	dialogueLines, rawLines, err := subsync.ParseAssFile(inputPath)
	if err != nil {
		fmt.Println("Error parsing .ass file:", err)
		return
	}

	// Conver input offset into time.Duration
	offset := time.Duration(offsetInSeconds) * time.Second

	// Apply the offset and save to a new filepath
	err = subsync.SaveSyncedAssFile(outputPath, rawLines, dialogueLines, offset)
	if err != nil {
		fmt.Println("Error applying offset to subtitle file", err)
		return
	}
	fmt.Println("Successfully generated sabotaged test file:", outputPath)
}
