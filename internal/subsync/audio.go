package subsync

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ExtractAudio uses ffmpeg to stream raw audio samples from a video into
// a slice of float64 values representing the audio's intensity over time.
// Each float64 value represents a measurement and 16,000 of them represent a second of audio.
func ExtractAudio(videoPath string) ([]float64, error) {
	// FFmpeg command:
	// -i: input file
	// -ac 1: convert to mono (single channel)
	// -ar 16000: sample rate of 16kHz (16,000 measurements of the sound wave amplitude per second)
	// -f s16le: raw 16-bit little-endian integers
	// pipe:1: send the result to Go's stdout pipe instead of a file to keep it in memory
	cmd := exec.Command(findFFmpegPath(), "-i", videoPath, "-ac", "1", "-ar", "16000", "-f", "s16le", "pipe:1")

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
	const chunkSize = 64 * 1024 // read 64KB at a time
	buffer := make([]byte, chunkSize)
	for {
		n, err := stdout.Read(buffer)
		if n > 0 {
			// Process every 2 bytes in the chunk
			for i := 0; i+1 < n; i += 2 {
				rawSample := binary.LittleEndian.Uint16(buffer[i : i+2])
				sample := int16(rawSample)
				samples = append(samples, float64(sample))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading audio stream: %v", err)
		}
	}

	// Wait for the process to clean up
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("FFmpeg cleanup note: %v\n", err)
	}

	return samples, nil
}

// findFfmpegPath tries to find ffmpeg bundled with the binary but defaults to look at PATH
// if it fails.
func findFFmpegPath() string {
	// Get the path of the running binary so we can get the directory
	binaryPath, err := os.Executable()
	if err != nil {
		return "ffmpeg"
	}
	dir := filepath.Dir(binaryPath)
	filename := "ffmpeg"
	if runtime.GOOS == "windows" {
		filename = "ffmpeg.exe"
	}
	path := filepath.Join(dir, filename)

	// Check if file even exists and fall back to looking at PATH for ffmpeg if it doesn't
	_, err = os.Stat(path)
	if err == nil {
		return path
	}
	return "ffmpeg"
}
