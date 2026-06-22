package subsync

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

// FormatAssTimestamp converts a time.Duration into an .ass compliant "H:MM:SS.CS" (centiseconds) string.
func FormatAssTimestamp(d time.Duration) string {
	// If applying the offset didn't push the negative timestamp to a positive timestamp,
	// clamp the 0 because .ass string format "H:MM:SS.cs" cannot represent negative time.
	if d < 0 {
		return "0:00:00.00"
	}

	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	// Centiseconds are 1/100th of a second.
	// Milliseconds / 10 is equivalent to a centisecond.
	centiseconds := (d.Milliseconds() % 1000) / 10

	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

// CleanText removes bracketed environmental subtitles like [music playing] or (sighs)
// from a given Dialogue event in an .ass file.
func CleanText(text string) string {
	// Match anything inside brackets or parentheses with regex
	re := regexp.MustCompile(`\[.*?\]|\(.*?\)`)
	cleaned := re.ReplaceAllString(text, "")
	return strings.TrimSpace(cleaned)
}

// ParseAssFile opens and scans each line of the .ass file to extract only the raw dialogue lines.
// It returns a slice of DialogueLine structs which hold the each line's subtitle and start and end timestamps.
// It also returns a slice of every line in the subtitle file.
func ParseAssFile(filePath string) ([]DialogueLine, []string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var lines []DialogueLine
	var rawLines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		rawLines = append(rawLines, line)

		// If not a dialogue line, do not process
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
	return lines, rawLines, scanner.Err()
}

// SaveSyncedAssFile writes the modified subtitles, with offset applied, out to a new file.
func SaveSyncedAssFile(outputPath string, rawLines []string, dialogueLines []DialogueLine, offset time.Duration) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	dialogueIdx := 0

	for _, rawLine := range rawLines {
		// If it isn't a Dialogue line, write the original raw text back out.
		if !strings.HasPrefix(rawLine, "Dialogue:") || dialogueIdx >= len(dialogueLines) {
			writer.WriteString(rawLine)
			writer.WriteString("\n")
			continue
		}
		currentDialogue := dialogueLines[dialogueIdx]
		dialogueIdx++

		// Calculate the new timestamps with offset applied
		newStart := currentDialogue.Start + offset
		newEnd := currentDialogue.End + offset

		// Rebuild the Dialogue line
		// .ass dialogue lines format: Diallogue: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
		// Keep the text part whole so split into 10 substrings
		parts := strings.SplitN(rawLine, ",", 10)
		if len(parts) < 10 {
			writer.WriteString(rawLine)
			writer.WriteString("\n")
			continue
		}

		// Update the start and end timestamps
		parts[1] = FormatAssTimestamp(newStart)
		parts[2] = FormatAssTimestamp(newEnd)

		// Join the substrings and write to file
		writer.Write([]byte(strings.Join(parts, ",") + "\n"))
	}
	// Make sure to flush the rest of the buffer if there is any left
	return writer.Flush()
}
