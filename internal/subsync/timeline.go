package subsync

import (
	"math"
	"time"
)

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

// GenerateAudioTimeline takes the raw audio samples and returns a []bool timeline.
// sampleRate: 16000 (samples per second)
// resolution: 100 ms
func GenerateAudioTimeline(audioSamples []float64, sampleRate int, resolution time.Duration) []bool {
	// Calculate how many samples fit into one slot (100 ms)
	samplesPerSlot := int(float64(sampleRate) * resolution.Seconds())

	numSlots := len(audioSamples) / samplesPerSlot
	timeline := make([]bool, numSlots)

	// Sensitivity threshold (volume gate)
	threshold := 0.02

	for i := 0; i < numSlots; i++ {
		startIdx := i * samplesPerSlot
		endIdx := startIdx + samplesPerSlot

		// Calculate RMS for this block of measurements
		var sumOfSquares float64
		for _, freq := range audioSamples[startIdx:endIdx] {
			// Normalize the value to make it easier to work with
			normalized := freq / 32768.0
			sumOfSquares += normalized * normalized
		}
		meanSquared := sumOfSquares / float64(samplesPerSlot)
		rms := math.Sqrt(meanSquared)

		// Only set the slot to true when there is audio loud enough for subtitles to appear
		if rms > threshold {
			timeline[i] = true
		}
	}

	return timeline
}
