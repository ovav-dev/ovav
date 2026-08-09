// OVAV cPanel v5.1 — SSE Events endpoint.
//
// GET /api/v1/events — Server-Sent Events stream with heartbeat.

package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// maxSSEConnections limits concurrent SSE streams to prevent resource exhaustion.
const maxSSEConnections = 100

var sseConnectionCount int32

// handleEvents streams SSE events to the client.
func handleEvents(w http.ResponseWriter, r *http.Request) {
	// Enforce connection limit
	if atomic.AddInt32(&sseConnectionCount, 1) > maxSSEConnections {
		atomic.AddInt32(&sseConnectionCount, -1)
		http.Error(w, "too many SSE connections", http.StatusServiceUnavailable)
		return
	}
	defer atomic.AddInt32(&sseConnectionCount, -1)
	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"time\":\"%s\"}\n\n", time.Now().UTC().Format(time.RFC3339))
	flusher.Flush()

	// Heartbeat ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"time\":\"%s\"}\n\n", t.UTC().Format(time.RFC3339))
			flusher.Flush()
		}
	}
}
