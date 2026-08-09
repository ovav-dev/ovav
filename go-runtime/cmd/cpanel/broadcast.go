// OVAV cPanel — SSE Broadcast Hub
//
// GOV-007: Broadcast fan-out for real-time update notifications.
// Replaces single-connection heartbeat SSE with a pub/sub hub
// that fans out events to all connected Cockpit clients.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// BroadcastEvent represents a server-to-client event pushed via SSE.
type BroadcastEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	Time  string      `json:"time"`
}

// broadcastHub manages all active SSE connections and fans out events.
type broadcastHub struct {
	mu          sync.RWMutex
	clients     map[chan BroadcastEvent]struct{}
	register    chan chan BroadcastEvent
	unregister  chan chan BroadcastEvent
	broadcast   chan BroadcastEvent
	clientCount int
}

var hub = &broadcastHub{
	clients:    make(map[chan BroadcastEvent]struct{}),
	register:   make(chan chan BroadcastEvent),
	unregister: make(chan chan BroadcastEvent),
	broadcast:  make(chan BroadcastEvent, 256),
}

// Start runs the broadcast hub event loop in a background goroutine.
func (h *broadcastHub) Start() {
	go func() {
		for {
			select {
			case ch := <-h.register:
				h.mu.Lock()
				h.clients[ch] = struct{}{}
				h.clientCount = len(h.clients)
				h.mu.Unlock()
			case ch := <-h.unregister:
				h.mu.Lock()
				delete(h.clients, ch)
				close(ch)
				h.clientCount = len(h.clients)
				h.mu.Unlock()
			case event := <-h.broadcast:
				h.mu.RLock()
				for ch := range h.clients {
					select {
					case ch <- event:
					default:
						// Client too slow — skip
					}
				}
				h.mu.RUnlock()
			}
		}
	}()
}

// PushEvent sends an event to all connected SSE clients.
func PushEvent(event string, data interface{}) {
	hub.broadcast <- BroadcastEvent{
		Event: event,
		Data:  data,
		Time:  time.Now().UTC().Format(time.RFC3339),
	}
}

// PushJSON marshals data as JSON and pushes it as an SSE event.
func PushJSON(event string, data interface{}) {
	hub.broadcast <- BroadcastEvent{
		Event: event,
		Data:  data,
		Time:  time.Now().UTC().Format(time.RFC3339),
	}
}

// ClientCount returns the number of connected SSE clients.
func ClientCount() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return hub.clientCount
}

// handleEventsSSE streams SSE events to a connected Cockpit client.
// Replaces the old heartbeat-only handleEvents with broadcast subscription.
func handleEventsSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx := r.Context()
	ch := make(chan BroadcastEvent, 64)

	hub.register <- ch
	defer func() { hub.unregister <- ch }()

	// Connected event
	connected := BroadcastEvent{
		Event: "connected",
		Data:  map[string]interface{}{"status": "connected", "clients": ClientCount(), "time": time.Now().UTC().Format(time.RFC3339)},
		Time:  time.Now().UTC().Format(time.RFC3339),
	}
	writeSSE(w, flusher, connected)

	// Heartbeat + broadcast loop
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			hb := BroadcastEvent{
				Event: "heartbeat",
				Data:  map[string]interface{}{"clients": ClientCount(), "time": time.Now().UTC().Format(time.RFC3339)},
				Time:  time.Now().UTC().Format(time.RFC3339),
			}
			writeSSE(w, flusher, hb)
		case event, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, flusher, event)
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event BroadcastEvent) {
	dataBytes, err := json.Marshal(event.Data)
	if err != nil {
		dataBytes = []byte(`{"error":"json marshal failed"}`)
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, string(dataBytes))
	flusher.Flush()
}
