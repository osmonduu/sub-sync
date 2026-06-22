package subsync

import "fmt"

// FindBestOffset compares the subtitle timeline against the reference audio timeline.
// It slides the subtitle timeline across a range of offsets to find the best match.
// It returns the highest matching offset, with the offset returned as in terms of slots.
// maxOffsetSlots defines how far left or right we are willing to check (e.g. 4 slots = +-400 ms)
func FindBestOffset(audioTimeline, subTimeline []bool, maxOffsetSlots int) (bestOffset int, bestMatchConfidence float64) {
	bestOffset = 0
	bestMatchConfidence = -1.0 // Iniitalize with a low score so any real match beats it

	// Slide the subtitle timeline from -maxOffsetSlots to +maxOffsetSlots
	for offset := -maxOffsetSlots; offset <= maxOffsetSlots; offset++ {
		score := calculateOverlapScore(audioTimeline, subTimeline, offset)
		fmt.Printf("Testing offset: %+d ms | Match score: %.2f\n", offset, score)

		// If the offset yields a better match than previous attempts, save it
		if score > bestMatchConfidence {
			bestMatchConfidence = score
			bestOffset = offset
		}
	}
	return bestOffset, bestMatchConfidence
}

// calculateOverlapScore computes how well the timelines match at a specific offset.
// The audio and sub files are split into 100 ms blocks represented by each index of their respective arrays.
// An index holds true when the audio or sub is active in the respective file.
func calculateOverlapScore(audio, subs []bool, offset int) float64 {
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
