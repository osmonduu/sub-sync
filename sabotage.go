package main

import (
	"fmt"
	"time"
)

// SabotageAssFile applies an offset to a .ass (Advanced Sub Station Alpha) file.
func SabotageAssFile(inputPath, outputPath string, offsetInSeconds int) {
	// Use parser to load lines
	dialogueLines, rawLines, err := ParseAssFile(inputPath)
	if err != nil {
		fmt.Println("Error parsing .ass file:", err)
		return
	}

	// Conver input offset into time.Duration
	offset := time.Duration(offsetInSeconds) * time.Second

	// Apply the offset and save to a new filepath
	err = SaveSyncedAssFile(outputPath, rawLines, dialogueLines, offset)
	if err != nil {
		fmt.Println("Error applying offset to subtitle file", err)
		return
	}
	fmt.Println("Successfully generated sabotaged test file:", outputPath)
}
