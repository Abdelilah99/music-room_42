package hub

import "sync"

// HubManager maps an entity ID to its Hub.
// It creates a Hub on the first connection and removes it when the last
// client disconnects, so there is no memory leak from idle rooms.
type HubManager struct {
	mu   sync.Mutex
	hubs map[string]*Hub
}

func NewHubManager() *HubManager {
	return &HubManager{hubs: make(map[string]*Hub)}
}

// GetOrCreate returns the existing Hub for entityID, or creates a new one.
// If the hub for that ID was stopped between GetOrCreate returning and the
// caller registering, tryRegister will return false and the caller should
// retry (ServeWS handles this automatically).
func (m *HubManager) GetOrCreate(entityID string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if h, ok := m.hubs[entityID]; ok {
		return h
	}

	h := newHub()
	m.hubs[entityID] = h

	go h.run(func() {
		m.mu.Lock()
		// Only remove this exact instance — a concurrent GetOrCreate may
		// have already replaced it with a fresh hub for the same ID.
		if m.hubs[entityID] == h {
			delete(m.hubs, entityID)
		}
		m.mu.Unlock()
	})

	return h
}

// Len returns the number of active hubs. Useful for tests and metrics.
func (m *HubManager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.hubs)
}
