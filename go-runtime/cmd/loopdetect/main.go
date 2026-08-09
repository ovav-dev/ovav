// Command loopdetect is a CLI harness for the OVAV Loop Detector Guard.
// Usage:
//
//	go run -C go-runtime ./cmd/loopdetect --events=<events.jsonl>
//
// Exits 0 (no loop), 2 (loop detected → stderr JSON), 1 (usage error).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/loopguard"
)

func main() {
	eventsPath := flag.String("events", "", "path to JSONL events file")
	minOcc := flag.Int("min", 3, "min occurrences to flag a loop")
	flag.Parse()

	if *eventsPath == "" {
		fmt.Fprintln(os.Stderr, "usage: loopdetect --events=<events.jsonl>")
		os.Exit(1)
	}

	events, err := loopguard.LoadEvents(*eventsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load events: %v\n", err)
		os.Exit(1)
	}

	cfg := loopguard.DefaultConfig
	if *minOcc > 0 {
		cfg.MinOccurrences = *minOcc
	}

	det := loopguard.Detect(events, cfg)
	if det.Detected {
		out, _ := json.MarshalIndent(det, "", "  ")
		fmt.Fprintf(os.Stderr, "OVAV_LOOP_DETECTED: %s\n", out)
		os.Exit(2)
	}

	fmt.Fprintln(os.Stderr, "OVAV_LOOP_CLEAR")
}
