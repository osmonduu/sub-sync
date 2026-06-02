package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// ParseTimestamp convers an ASSA time string "H:M:S.C" (centiseconds) into a Go time.Duration.
func ParseTimestamp(timestamp string) (time.Duration, error) {
	var hours, minutes, seconds int
	var centiseconds int

	// Match the timestamp format of .ass dialogue events and store into variables
	_, err := fmt.Sscanf(timestamp, "%d:%d:%d.%d", &hours, &minutes, &seconds, &centiseconds)
	if err != nil {
		return 0, err
	}

	milliseconds := centiseconds * 10

	// Calculate the total duration
	totalDuration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds)*time.Millisecond

	return totalDuration, nil
}

// CleanText removes bracketed environmental subtitles like [music playing] or (sighs).
func CleanText(text string) string {
	// Match anything inside brackets or parentheses with regex
	re := regexp.MustCompile(`\[.*?\]|\(.*?\)`)
	cleaned := re.ReplaceAllString(text, "")
	return strings.TrimSpace(cleaned)
}

// ParseAssFile opens and scans each line of the .ass file to extract only the raw dialogue lines.
// It returns a slice of DialogueLine structs which hold the each line's subtitle and start and end timestamps.
func ParseAssFile(filePath string) ([]DialogueLine, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []DialogueLine
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Only process lines that represent subtitle dialogue
		if !strings.HasPrefix(line, "Dialogue:") {
			continue
		}

		// Split the line into its 10 respective sections
		parts := strings.SplitN(line, ",", 10)
		if len(parts) < 10 {
			continue // Malformed line
		}
		startTimeStr := parts[1]
		endTimeStr := parts[2]
		rawText := parts[9]

		// Filter out lines with environmental descriptions
		cleanedText := CleanText(rawText)
		isDialogueLine := true
		if cleanedText == "" {
			// Mark line as environmental descriptions so the math engine ignores it
			isDialogueLine = false
		}

		// Convert the start and end timestamps to time.Duration
		start, err := ParseTimestamp(startTimeStr)
		if err != nil {
			continue // since the start timestamp is malformed, skip line entirely
		}

		end, err := ParseTimestamp(endTimeStr)
		if err != nil {
			continue // since the end timestamp is malformed, skip line entirely
		}

		lines = append(lines, DialogueLine{
			Start:      start,
			End:        end,
			Text:       cleanedText,
			IsDialogue: isDialogueLine,
		})
	}
	// Return any errors (if any) that occured during the scanning
	return lines, scanner.Err()
}

// GenerateSubTimeline converts the parsed subtitle lines into a []bool timeline,
// where a 'true' value indicates the presence of dialogue.
// resolution defines the duration of each slot (e.g. 100 ms)
func GenerateSubTimeline(lines []DialogueLine, resolution time.Duration) []bool {
	if len(lines) == 0 {
		return []bool{}
	}

	// Find the last subtitle timestamp of the track to know how large our timeline should be
	var maxDuration time.Duration
	for _, line := range lines {
		if line.End > maxDuration {
			maxDuration = line.End
		}
	}

	// Calculate number of slots required
	totalSlots := int(maxDuration/resolution) + 1
	timeline := make([]bool, totalSlots)

	// Populate the timeline with the DialogueLines
	for _, line := range lines {
		// Skip environmental subtitles when building the timeline
		if !line.IsDialogue {
			continue
		}

		// Calculate the index which each timestamp should be in
		startIdx := int(line.Start / resolution)
		endIdx := int(line.End / resolution)

		// Mark the time from the start to the end of the dialogue as true
		for i := startIdx; i <= endIdx; i++ {
			if i >= 0 && i < len(timeline) {
				timeline[i] = true
			}
		}
	}

	return timeline
}

func main() {
	fmt.Println("Starting .ass Subtitle Parser...")
	fmt.Println("--------------------------------------")

	dialogueLines, err := ParseAssFile("test.ass")
	if err != nil {
		fmt.Printf("Error reading files: %v\n", err)
		return
	}

	// Print our parsed structs to verify accuracy
	for idx, d := range dialogueLines {
		fmt.Printf("Line %d\n", idx+1)
		fmt.Printf("	Cleaned Text: %s\n", d.Text)
		fmt.Printf("	Start Time	: %v\n", d.Start)
		fmt.Printf("	End Time	: %v\n", d.End)
		fmt.Println()
	}
}
