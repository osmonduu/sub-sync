package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/osmonduu/sub-sync/internal/subsync"
)

type Checkpoint struct {
	Timestamp time.Duration
	Offset    time.Duration
}

func main() {
	input := flag.String("input", "", "Path to input .ass file")
	output := flag.String("output", "", "Name for ouput .ass file")
	mode := flag.String("mode", "global", "Sabotage mode: global, step, or ramp")
	offset := flag.Int("offset", 0, "Offset in milliseconds to apply")
	checkpoints := flag.String(
		"checkpoints",
		"",
		"Comma-separated timestamp(seconds):offset(milliseconds) pairs, e.g. 0:0,300:2000,600:-1000",
	)
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(
			os.Stderr,
			"Usage: sabotage -input <file.ass> -output <file.ass> -mode <global/step/ramp> -offset <milliseconds> -checkpoints <timestamp:offset>",
		)
		os.Exit(1)
	}

	switch *mode {
	case "global":
		err := sabotageGlobal(*input, *output, *offset)
		if err != nil {
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
	case "step", "ramp":
		checkpointArr, err := parseCheckpoints(*checkpoints)
		if err != nil {
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
		err = sabotageCheckpoints(*mode, *input, *output, checkpointArr)
		if err != nil {
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown mode. Please use global, step, or ramp")
		os.Exit(1)
	}
}

// parseCheckpoints checks if the "checkpoints" string is correctly formatted
// and returns a Checkpoint array if so. Otherwise, it returns nil and an error.
func parseCheckpoints(checkpoints string) ([]Checkpoint, error) {
	if checkpoints == "" {
		return nil, errors.New("missing checkpoints input")
	}
	checkpointStrings := strings.Split(checkpoints, ",")
	checkpointArr := make([]Checkpoint, 0, len(checkpointStrings))
	// Check if each checkpoint is formatted correctly with the ":" separator
	// and if so, create Checkpoint, and append to return array
	for _, cp := range checkpointStrings {
		elements := strings.Split(cp, ":")
		if len(elements) != 2 {
			return nil, errors.New("malformed checkpoints string. Correct format timestamp:offset")
		}
		timestamp, err := strconv.Atoi(elements[0])
		if err != nil {
			return nil, errors.New("timestamp malformed")
		}
		offset, err := strconv.Atoi(elements[1])
		if err != nil {
			return nil, errors.New("offset malformed")
		}
		checkpointArr = append(checkpointArr, Checkpoint{
			Timestamp: time.Duration(timestamp) * time.Second,
			Offset:    time.Duration(offset) * time.Millisecond,
		})
	}

	return checkpointArr, nil
}

// sabotageGlobal applies a fixed global offset to a .ass (Advanced Sub Station Alpha) file.
func sabotageGlobal(inputPath, outputPath string, offsetMs int) error {
	// Use parser to load lines
	dialogueLines, rawLines, err := subsync.ParseAssFile(inputPath)
	if err != nil {
		return err
	}

	// Conver input offset into time.Duration
	offset := time.Duration(offsetMs) * time.Millisecond

	alignedLines := make([]subsync.AlignedLine, 0, len(dialogueLines))
	for _, dialogueLine := range dialogueLines {
		alignedLines = append(alignedLines, subsync.AlignedLine{
			Dialogue: dialogueLine,
			Offset:   offset,
		})
	}

	// Apply the offset and save to a new filepath
	err = subsync.SaveSyncedAssFile(outputPath, rawLines, alignedLines)
	if err != nil {
		return err
	}
	fmt.Println("Successfully generated sabotaged test file: ", outputPath)
	return nil
}

// sabotageCheckpoints applies a fixed or interpolated offset to every subtitle based on the mode
// and checkpoints passed in.
func sabotageCheckpoints(mode string, inputPath, outputPath string, checkpoints []Checkpoint) error {
	if mode != "step" && mode != "ramp" {
		return errors.New("unknown mode")
	}
	dialogueLines, rawLines, err := subsync.ParseAssFile(inputPath)
	if err != nil {
		return errors.New("error parsing .ass file")
	}

	alignedLines := make([]subsync.AlignedLine, 0, len(dialogueLines))

	if mode == "step" {
		for _, dl := range dialogueLines {
			// Mirror the behavior of the sync engine by using the midpoint of the line
			midpoint := dl.Start + (dl.End-dl.Start)/2
			stepOffset := stepOffsetForLine(checkpoints, midpoint)
			alignedLines = append(alignedLines, subsync.AlignedLine{
				Dialogue: dl,
				Offset:   stepOffset,
			})
		}
	} else { // mode == "ramp"
		for _, dl := range dialogueLines {
			midpoint := dl.Start + (dl.End-dl.Start)/2
			rampOffset := rampOffsetForLine(checkpoints, midpoint)
			alignedLines = append(alignedLines, subsync.AlignedLine{
				Dialogue: dl,
				Offset:   rampOffset,
			})
		}
	}

	// Apply offsets to each line and save file
	err = subsync.SaveSyncedAssFile(outputPath, rawLines, alignedLines)
	if err != nil {
		return err
	}
	fmt.Println("Successfully generated sabotaged test file:", outputPath)
	return nil
}

// stepOffsetForLine finds the step offset for a given subtitle line based on the
// checkpoints array. In step mode, each lineTimestamp that falls between checkpoints[i].Timestamp
// and checkpoints[i+1].Timestamp should have an offset of checkpoints[i].Offset applied.
func stepOffsetForLine(checkpoints []Checkpoint, lineTimestamp time.Duration) time.Duration {
	offset := checkpoints[0].Offset
	for _, cp := range checkpoints {
		// If we just passed lineTimestamp, it falls between this cp.Timestamp
		// and the previous, so the previous offset should be applied
		if cp.Timestamp > lineTimestamp {
			break
		}
		offset = cp.Offset
	}
	return offset
}

// rampOffsetForLine computes the ramp offset for a given substile line based on the
// checkpoints array. In ramp mode, each lineTimestamp that falls between checkpoints[i].Timestamp
// and checkpoints[i+1].Timestamp should have an interpolated offset based on lineTimestamp's position
// between the two checkpoints' Offset. The interpolation calculation and logic is the same as
// subsync.interpolateOffset.
func rampOffsetForLine(checkpoints []Checkpoint, lineTimestamp time.Duration) time.Duration {
	if lineTimestamp <= checkpoints[0].Timestamp {
		return checkpoints[0].Offset
	}
	lastCheckpoint := checkpoints[len(checkpoints)-1]
	if lineTimestamp >= lastCheckpoint.Timestamp {
		return lastCheckpoint.Offset
	}

	for i := 1; i < len(checkpoints); i++ {
		if lineTimestamp < checkpoints[i].Timestamp {
			left := checkpoints[i-1]
			right := checkpoints[i]
			t := float64(lineTimestamp-left.Timestamp) / float64(right.Timestamp-left.Timestamp)
			return left.Offset + time.Duration(t*float64(right.Offset-left.Offset))
		}
	}
	return 0
}
