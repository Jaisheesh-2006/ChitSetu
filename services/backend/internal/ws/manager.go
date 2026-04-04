package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeTimeout = 10 * time.Second
)

type clientInfo struct {
	conn          *websocket.Conn
	userID        string
	inAuctionRoom bool
}

type registerRequest struct {
	client *clientInfo
	ready  chan struct{}
}

type Hub struct {
	fundID          string
	clients         map[*websocket.Conn]*clientInfo
	broadcast       chan []byte
	register        chan registerRequest
	unregister      chan *websocket.Conn
	auctionJoin     chan *websocket.Conn
	auctionLeave    chan *websocket.Conn
	auctionCountReq chan chan int
	closeOnce       sync.Once
}

type Manager struct {
	mu   sync.RWMutex
	hubs map[string]*Hub
	// OnAuctionParticipantChange is called after a user joins/leaves an auction room.
	// It receives (fundID, uniqueUserCount).
	OnAuctionParticipantChange func(fundID string, count int)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewManager() *Manager {
	return &Manager{hubs: make(map[string]*Hub)}
}

func (m *Manager) ServeFundConnection(w http.ResponseWriter, r *http.Request, fundID, userID string) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	hub := m.getOrCreateHub(fundID)
	ready := make(chan struct{})
	hub.register <- registerRequest{
		client: &clientInfo{conn: conn, userID: userID},
		ready:  ready,
	}
	<-ready
	go hub.readPump(conn)
	return nil
}

func (m *Manager) Broadcast(fundID string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	hub := m.getOrCreateHub(fundID)
	hub.broadcast <- encoded
	return nil
}

// AuctionParticipantCount returns the number of unique users that explicitly
// joined the auction room for a fund.
func (m *Manager) AuctionParticipantCount(fundID string) int {
	m.mu.RLock()
	hub, ok := m.hubs[fundID]
	m.mu.RUnlock()
	if !ok {
		return 0
	}

	response := make(chan int, 1)
	hub.auctionCountReq <- response
	return <-response
}

// ConnectedUserCount returns the number of unique user IDs connected to a fund room.
func (m *Manager) ConnectedUserCount(fundID string) int {
	m.mu.RLock()
	hub, ok := m.hubs[fundID]
	m.mu.RUnlock()
	if !ok {
		return 0
	}
	seen := make(map[string]struct{})
	for _, ci := range hub.clients {
		if ci.userID != "" {
			seen[ci.userID] = struct{}{}
		}
	}
	return len(seen)
}

func (m *Manager) getOrCreateHub(fundID string) *Hub {
	m.mu.RLock()
	hub, ok := m.hubs[fundID]
	m.mu.RUnlock()
	if ok {
		return hub
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, exists := m.hubs[fundID]; exists {
		return existing
	}

	hub = &Hub{
		fundID:          fundID,
		clients:         make(map[*websocket.Conn]*clientInfo),
		broadcast:       make(chan []byte, 128),
		register:        make(chan registerRequest),
		unregister:      make(chan *websocket.Conn),
		auctionJoin:     make(chan *websocket.Conn),
		auctionLeave:    make(chan *websocket.Conn),
		auctionCountReq: make(chan chan int),
	}
	m.hubs[fundID] = hub
	go hub.run(m)
	return hub
}

func (m *Manager) removeHubIfIdle(fundID string, hub *Hub) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.hubs[fundID]
	if !ok || existing != hub {
		return
	}
	if len(existing.clients) > 0 {
		return
	}
	delete(m.hubs, fundID)
	hub.closeOnce.Do(func() {
		close(hub.broadcast)
	})
}

func (h *Hub) run(manager *Manager) {
	for {
		select {
		case req := <-h.register:
			h.clients[req.client.conn] = req.client
			if req.ready != nil {
				close(req.ready)
			}
		case conn := <-h.auctionJoin:
			ci, ok := h.clients[conn]
			if !ok || ci.inAuctionRoom {
				continue
			}
			ci.inAuctionRoom = true
			if manager.OnAuctionParticipantChange != nil {
				count := h.uniqueAuctionUserCount()
				go manager.OnAuctionParticipantChange(h.fundID, count)
			}
		case conn := <-h.auctionLeave:
			ci, ok := h.clients[conn]
			if !ok || !ci.inAuctionRoom {
				continue
			}
			ci.inAuctionRoom = false
			if manager.OnAuctionParticipantChange != nil {
				count := h.uniqueAuctionUserCount()
				go manager.OnAuctionParticipantChange(h.fundID, count)
			}
		case conn := <-h.unregister:
			wasAuctionParticipant := false
			if ci, ok := h.clients[conn]; ok {
				wasAuctionParticipant = ci.inAuctionRoom
				delete(h.clients, conn)
				_ = conn.Close()
			}
			if wasAuctionParticipant && manager.OnAuctionParticipantChange != nil {
				count := h.uniqueAuctionUserCount()
				go manager.OnAuctionParticipantChange(h.fundID, count)
			}
			if len(h.clients) == 0 {
				manager.removeHubIfIdle(h.fundID, h)
			}
		case response := <-h.auctionCountReq:
			response <- h.uniqueAuctionUserCount()
		case message, ok := <-h.broadcast:
			if !ok {
				for conn := range h.clients {
					_ = conn.Close()
				}
				return
			}
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					_ = conn.Close()
				}
			}
		}
	}
}

func (h *Hub) uniqueUserCount() int {
	seen := make(map[string]struct{})
	for _, ci := range h.clients {
		if ci.userID != "" {
			seen[ci.userID] = struct{}{}
		}
	}
	return len(seen)
}

func (h *Hub) uniqueAuctionUserCount() int {
	seen := make(map[string]struct{})
	for _, ci := range h.clients {
		if ci.userID != "" && ci.inAuctionRoom {
			seen[ci.userID] = struct{}{}
		}
	}
	return len(seen)
}

func (h *Hub) readPump(conn *websocket.Conn) {
	defer func() {
		h.unregister <- conn
	}()

	conn.SetReadLimit(1024)

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var incoming struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(payload, &incoming); err != nil {
			continue
		}

		switch incoming.Type {
		case "auction_room_join":
			h.auctionJoin <- conn
		case "auction_room_leave":
			h.auctionLeave <- conn
		}
	}
}
