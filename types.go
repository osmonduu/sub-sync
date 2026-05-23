package main

import "time"

type DialogueLine struct {
	Start      time.Duration
	End        time.Duration
	Text       string
	IsDialogue bool
}
