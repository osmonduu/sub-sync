package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"time"
)

// ExtractAudio uses ffmpeg to stream raw audio samples from a video.
// Returns a slice of float64 values representing the audio's intensity over time.
// Each float64 value represents a measurement and 16,000 of them represent a second of audio.
func ExtractAudio(videoPath string) ([]float64, error) {
	// FFmpeg command:
	// -i: input file
	// -ac 1: convert to mono (single channel)
	// -ar 16000: sample rate of 16kHz (16,000 measurements of the sound wave amplitude per second)
	// -f s16le: raw 16-bit little-endian integers
	// pipe:1: send the result to Go's stdout pipe instead of a file to keep it in memory
	cmd := exec.Command("ffmpeg", "-i", videoPath, "-ac", "1", "-ar", "16000", "-f", "s16le", "pipe:1")

	// Connect to the command's stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Could not create stdout pipe: %v", err)
	}

	// Start the command in the background
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Could not start ffmpeg: %v", err)
	}

	var samples []float64

	// Read each audio sample (16-bits/2 bytes)
	buffer := make([]byte, 2)
	for {
		_, err := io.ReadFull(stdout, buffer)
		if err == io.EOF {
			break // finished processing
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading audio stream: %v", err)
		}

		// Read the 2 bytes as an unsigned 16-bit little endian int first
		rawSample := binary.LittleEndian.Uint16(buffer)

		// Convert back to a signed 16-bit integer then cast to float64 decimal
		sample := int16(rawSample)
		samples = append(samples, float64(sample))
	}

	// Wait for the process to clean up
	if err := cmd.Wait(); err != nil {
		fmt.Printf("FFmpeg cleanup note: %v\n", err)
	}

	return samples, nil
}

// GetVoiceActivity takes the raw audio samples and returns a []bool timeline.
// sampleRate: 16000 (samples per second)
// resolution: 100 ms
func GetVoiceActivity(audioSamples []float64, sampleRate int, resolution time.Duration) []bool {
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
