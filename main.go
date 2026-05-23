package main

import (
	"fmt"
)

// FindBestOffset compares the subtitle timeline against the reference audio timeline.
// It slides the subtitle timeline across ar ange of offsets to find the best match.
// maxOffsetSlots defines how far left or right we are willing to check (e.g. 4 slots = +-400 ms)
func FindBestOffset(audioTimeline, subTimeline []bool, maxOffsetSlots int) (int, float64) {
	bestOffset := 0
	bestScore := -1.0 // Iniitalize with a low score so any real match beats it

	// Slide the subtitle timeline from -maxOffsetSlots to +maxOffsetSlots
	for offset := -maxOffsetSlots; offset <= maxOffsetSlots; offset++ {
		score := calculateOverlapSocre(audioTimeline, subTimeline, offset)
		fmt.Printf("Testing offset: %+d ms | Match score: %.2f%%\n", offset, score*100)

		// If the offset yields a better match than previous attempts, save it
		if score > bestScore {
			bestScore = score
			bestOffset = offset
		}
	}
	return bestOffset, bestScore
}

// calculateOverlapScore computes how well the timelines match at a specific offset.
// The audio and sub files are split into 100 ms blocks represented by each index of their respective arrays.
// An index holds true when the audio or sub is active in the respective file.
func calculateOverlapSocre(audio, subs []bool, offset int) float64 {
	matches := 0
	totalSubSlots := 0

	for subIdx, subIsActive := range subs {
		if subIsActive {
			totalSubSlots++

			// Map the subtitle index to the corresponding audio index based on the current slide offset
			audioIdx := subIdx + offset

			// Ensure we aren't looking outside the boundaries of our audio track array
			if audioIdx >= 0 && audioIdx < len(audio) {
				// If both audio and subtitle are active at this point in time, it's a successful match
				if audio[audioIdx] {
					matches++
				}
			}
		}
	}
	// Prevent division by zero if subtitle file has zero dialogue
	if totalSubSlots == 0 {
		return 0.0
	}
	// Return the percentage of subtitle slots that successfully matched to audio (0.0 to 1.0)
	return float64(matches) / float64(totalSubSlots)
}

/* func main() {
	// Assume 1 slot = 1 second
	audioTimeline := []bool{
		false, false, false, true, true, true, // 0 to 5 seconds
		false, false, false, true, true, true, // 6 to 11 seconds
		false, false, false, // 12 to 14 seconds
	}
	unsyncedSubTimeline := []bool{
		false, true, true, true, false, false, // 0 to 5 seconds
		false, true, true, true, false, false, // 6 to 11 seconds
		false, false, false, // 12 to 14 seconds
	}

	// Search up to 4 seconds in either direction
	maxSearchDistance := 4

	fmt.Println("Starting timeline alignment simultion...")
	fmt.Println("----------------------------------------")

	//Execute the search
	bestOffset, matchConfidence := FindBestOffset(audioTimeline, unsyncedSubTimeline, maxSearchDistance)

	fmt.Println()
	fmt.Println("----------------------------------------")
	fmt.Println("ALGORITHM RESULT: ")
	fmt.Printf("Subtitles shifted by: %+d slots.\n", bestOffset)
	fmt.Printf("Match condience: %.2f%%\n", matchConfidence*100)
} */
