package subsync

import (
	// "fmt"
	"math"
	"time"
)

type segmentResult struct {
	center     time.Duration
	offset     time.Duration
	confidence float64
}

const (
	minConfidence   = 0.3 // segments below this threshold are exclueded from interpolation
)

// FindBestOffset compares the subtitle timeline against the reference audio timeline.
// It slides the subtitle timeline across a range of offsets to find the best match.
// It returns the highest matching offset, with the offset returned as in terms of slots.
// maxOffsetSlots defines how far left or right we are willing to check (e.g. 4 slots = +-400 ms)
func FindBestOffset(
	audioTimeline []bool,
	subTimeline []bool,
	maxOffsetSlots int,
) (bestOffsetSlots int, bestMatchConfidence float64) {
	bestMatchConfidence = -1.0 // Iniitalize with an impossible negative score so any real match beats it

	// Slide the subtitle timeline from -maxOffsetSlots to +maxOffsetSlots
	for offsetSlots := -maxOffsetSlots; offsetSlots <= maxOffsetSlots; offsetSlots++ {
		score := calculateOverlapScore(audioTimeline, subTimeline, offsetSlots)
		// fmt.Printf("Testing offset: %+d ms | Match score: %.2f\n", offsetSlots*100, score)
		// If the offset yields a better match than previous attempts, save it
		if score > bestMatchConfidence {
			bestMatchConfidence = score
			bestOffsetSlots = offsetSlots
		}
	}
	return bestOffsetSlots, bestMatchConfidence
}

// calculateOverlapScore returns a 0.0-1.0 match score between the audio and subtitle timelines
// at a given offset. Both 'audio' and 'subs' timelines are boolean arrays where each index represents a 100ms block,
// true indicating active dialogue or subtitles. The score is the fraction of active subtitle slots whose
// shifted position also has active audio.
func calculateOverlapScore(audio, subs []bool, offsetSlots int) float64 {
	totalSubSlots := 0 // tracks the total number of indices that have subtitles
	matches := 0       // tracks the number of 'true' subtitle index and 'true' audio index matches

	for subIdx, subIsActive := range subs {
		if subIsActive {
			totalSubSlots++

			// Map the subtitle index to the corresponding audio index based on the current slide offset
			audioIdx := subIdx + offsetSlots

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

//----------------------------------------------------------------------------------------------------------------------

// AlignPiecewise computes a unique offset for each dialogue line by segmenting
// the file into segmentDuration windows, finding the best offset for the set of subtitles within
// that segment, then interpolating the offset of the  subtitle based on its position between the
// two nearest segment centers. If no segment meets the minConfidence score, the function falls
// back to a single file wide offset using FindBestOffset, applied to every line.
// AlignPiecewise also returns the minimum and maximum offset applied across all lines.
func AlignPiecewise(
	dialogueLines []DialogueLine,
	audioTimeline []bool,
	resolution time.Duration,
	maxSearchDistanceSlots int,
	segmentDuration time.Duration,
) (alignedLines []AlignedLine, minOffset time.Duration, maxOffset time.Duration) {
	subTimeline := GenerateSubTimeline(dialogueLines, resolution)
	segments := computeSegmentOffsets(audioTimeline, subTimeline, resolution, maxSearchDistanceSlots, segmentDuration)

	alignedLines = make([]AlignedLine, 0, len(dialogueLines))
	minOffset = time.Duration(math.MaxInt64)
	maxOffset = time.Duration(math.MinInt64)

	if len(segments) == 0 {
		// Fallback plan: call FindBestOffset to find the best offset for the entire sub file
		// and apply the offset to every dialogue line
		offsetSlots, _ := FindBestOffset(audioTimeline, subTimeline, maxSearchDistanceSlots)
		offset := time.Duration(offsetSlots) * resolution
		minOffset = offset
		maxOffset = offset

		for _, dialogueLine := range dialogueLines {
			aligned := AlignedLine{
				Dialogue: dialogueLine,
				Offset:   offset,
			}
			alignedLines = append(alignedLines, aligned)
		}
	} else {
		// For each dialogue line, compute its midpoint and find its interpolated offset
		for _, dialogueLine := range dialogueLines {
			midpoint := dialogueLine.Start + (dialogueLine.End-dialogueLine.Start)/2
			interpolatedOffset := interpolateOffset(segments, midpoint)

			aligned := AlignedLine{
				Dialogue: dialogueLine,
				Offset:   interpolatedOffset,
			}
			alignedLines = append(alignedLines, aligned)

			if interpolatedOffset < minOffset {
				minOffset = interpolatedOffset
			}
			if interpolatedOffset > maxOffset {
				maxOffset = interpolatedOffset
			}
		}
	}

	return alignedLines, minOffset, maxOffset
}

// computeSegmentOffsets splits the audio into segmentDuration sections and returns an array of segmentResults
// with each segment's best offset. Segments with a low  match confidence score are excluded.
func computeSegmentOffsets(
	audioTimeline []bool,
	subTimeline []bool,
	resolution time.Duration,
	maxSearchDistanceSlots int,
	segmentDuration time.Duration,
) []segmentResult {
	slotsPerSegment := int(segmentDuration / resolution)
	var segments []segmentResult

	for start := 0; start < len(subTimeline); start += slotsPerSegment {
		end := start + slotsPerSegment
		if end > len(subTimeline) {
			end = len(subTimeline) // last segment may be shorter
		}

		segmentSubs := subTimeline[start:end]

		// Run an offset sweep using segmentSubs against the full audioTimeline to find the best offset
		bestOffsetSlots := 0
		bestMatchConfidence := -1.0

		for offsetSlots := -maxSearchDistanceSlots; offsetSlots <= maxSearchDistanceSlots; offsetSlots++ {
			matchScore := calculateSegmentOverlapScore(audioTimeline, segmentSubs, start, offsetSlots)
			if matchScore > bestMatchConfidence {
				bestMatchConfidence = matchScore
				bestOffsetSlots = offsetSlots
			}
		}

		// Compute this segment's center timestamp
		centerSlot := start + (end-start)/2
		centerTimestamp := time.Duration(centerSlot) * resolution

		// DEBUG
		// fmt.Printf("Segment [%v]: best offset %v, confidence %.3f\n", centerTimestamp, time.Duration(bestOffsetSlots)*resolution, bestMatchConfidence)

		// Only append to 'segments' if the score meets or exceeds the confidence threshold
		if bestMatchConfidence < minConfidence {
			continue
		}
		segment := segmentResult{
			center:     centerTimestamp,
			offset:     time.Duration(bestOffsetSlots) * resolution,
			confidence: bestMatchConfidence,
		}
		segments = append(segments, segment)
	}

	return segments
}

// calculateSegmentOverlapScore is calculatedOverlapScore adapted for a sliced subtitle window (segmentSubs).
// Since segmentSubs is a slice, its indices are local to the segment. The 'start' parameter helps converts
// them back to global positions to be compared to the full audio timeline.
func calculateSegmentOverlapScore(audio, segmentSubs []bool, start int, offsetSlots int) float64 {
	totalSubSlots := 0 // tracks the total number of indices that have subtitles
	matches := 0       // tracks the number of 'true' subtitle index and 'true' audio index matches

	for localIdx, subIsActive := range segmentSubs {
		if subIsActive {
			totalSubSlots++

			// Convert the local segment index back to its true global position
			// and then apply the offset
			audioIdx := start + localIdx + offsetSlots
			if audioIdx >= 0 && audioIdx < len(audio) {
				if audio[audioIdx] {
					matches++
				}
			}
		}
	}

	if totalSubSlots == 0 {
		return 0.0
	}
	return float64(matches) / float64(totalSubSlots)
}

// interpolateOffset returns the interpolated offset for a subtitle line determined by the two nearest
// surrounding segment offsets. The subtitle's midpoint is used to find the subtitle's fractional
// position between the two surrounding segments. If subtitleMidpoint falls before the first or after
// last segment's center, the nearest segment's offset is used.
func interpolateOffset(segments []segmentResult, subtitleMidpoint time.Duration) time.Duration {
	// Handle the case where the subtitles are before the first segment's center
	if subtitleMidpoint < segments[0].center {
		return segments[0].offset
	}
	// Handle the case where the subtitles are after the last segment's center
	lastSegment := segments[len(segments)-1]
	if subtitleMidpoint > lastSegment.center {
		return lastSegment.offset
	}

	// Walk through segments to find the closest segments with valid offsets that
	// surround the subtitle
	for i := 1; i < len(segments); i++ {
		if segments[i].center >= subtitleMidpoint {
			left := segments[i-1]
			right := segments[i]
			// Compute the fractional position of subtitleMidpoint between the two surrounding segments
			t := float64(subtitleMidpoint-left.center) / float64(right.center-left.center)
			// Blend the two offsets using the fractional position and return the result
			return left.offset + time.Duration(t*float64(right.offset-left.offset))
		}
	}

	return 0
}
