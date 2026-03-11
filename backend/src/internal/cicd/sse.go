package cicd

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// ── SSE Hub ─────────────────────────────────────────────────────
// Manages all active SSE connections keyed by runID

type client struct {
	ch     chan string
	runID  int64
}

type Hub struct {
	mu      sync.RWMutex
	clients map[int64][]chan string
}

var GlobalHub = &Hub{
	clients: make(map[int64][]chan string),
}

func (h *Hub) Subscribe(runID int64) chan string {
	ch := make(chan string, 32)
	h.mu.Lock()
	h.clients[runID] = append(h.clients[runID], ch)
	h.mu.Unlock()
	log.Printf("[SSE] client subscribed runID=%d total=%d", runID, len(h.clients[runID]))
	return ch
}

func (h *Hub) Unsubscribe(runID int64, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.clients[runID]
	for i, c := range list {
		if c == ch {
			h.clients[runID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(h.clients[runID]) == 0 {
		delete(h.clients, runID)
	}
	close(ch)
	log.Printf("[SSE] client unsubscribed runID=%d remaining=%d", runID, len(h.clients[runID]))
}

// Broadcast sends a pipeline event to all subscribers of a runID
func (h *Hub) Broadcast(runID int64, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[SSE] marshal error: %v", err)
		return
	}
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.clients[runID] {
		select {
		case ch <- msg:
		default:
			log.Printf("[SSE] client buffer full runID=%d, dropping event", runID)
		}
	}
}