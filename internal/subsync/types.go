package subsync

import "time"

// DialogueLine holds information of an .ass (Advanced Sub-Station Alpha) Events dialogue line.
type DialogueLine struct {
	Start    time.Duration
	End      time.Duration
	Text     string
	IsSpoken bool // flag to indicate whether this dialogue line is spoken or environmental dialogue
}

// AlignedLine inherets DialogueLine along with the subtitle line's unique calculated offset.
type AlignedLine struct {
	Dialogue DialogueLine
	Offset   time.Duration
}
