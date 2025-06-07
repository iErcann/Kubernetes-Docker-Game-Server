package shared

import (
	"time"

	"github.com/gorilla/websocket"
)

// Game world state shared between client and server
type AnimationState int

const (
	Idle AnimationState = iota
	Walking
	Running
	Jumping
)

// Client message for sending position/rotation/animation updates
type ClientMessage struct {
	Position  Position       `json:"position"`
	Rotation  Rotation       `json:"rotation"`
	Animation AnimationState `json:"animation"`
}

type Position struct {
	// World coordinates
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Rotation struct {
	// Rotation in radians
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Player struct that includes both game state and connection info
type Player struct {
	// Connection info
	WSConn              *websocket.Conn `json:"-"` // Don't serialize connection
	ID                  string          `json:"id"`
	ConnectedTime       time.Time       `json:"-"` // Don't serialize connection time
	LastReceivedMessage time.Time       `json:"-"` // Don't serialize last message time

	// Game state (serialized for clients)
	Name      string         `json:"name"`
	Position  Position       `json:"position"`
	Rotation  Rotation       `json:"rotation"`
	Animation AnimationState `json:"animation"`
}

func (p *Player) UpdateLastMessage() {
	p.LastReceivedMessage = time.Now()
}

func (p *Player) SendMessage(messageType int, data []byte) error {
	return p.WSConn.WriteMessage(messageType, data)
}

type GameWorld struct {
	Tick    int       `json:"tick"`
	Time    time.Time `json:"time"`
	Players []Player  `json:"players"`
}
