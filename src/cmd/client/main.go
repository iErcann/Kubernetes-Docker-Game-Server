package main

import (
	"encoding/json"
	"fmt"
	"game-server/internal/shared"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// Generate a unique session ID
	sessionID := fmt.Sprintf("client-%d", time.Now().Unix())

	// Server URL
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	q := u.Query()
	q.Set("session", sessionID)
	u.RawQuery = q.Encode()

	log.Printf("Connecting to %s as %s", u.String(), sessionID)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// Handle interrupt signal for graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Channel to receive messages
	done := make(chan struct{})

	// Goroutine to read messages from server
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			// Try to parse as GameWorld (server broadcasts)
			var world shared.GameWorld
			if err := json.Unmarshal(message, &world); err == nil {
				log.Printf("Game State - Tick: %d, Players: %d, Time: %s",
					world.Tick, len(world.Players), world.Time.Format("15:04:05.000"))

				// Print player info
				for _, player := range world.Players {
					log.Printf("  Player %s: Pos(%.2f, %.2f), Rot: %.2f, Anim: %d",
						player.Name, player.Position.X, player.Position.Y,
						player.Rotation, player.Animation)
				}
			} else {
				// Regular message (echo responses)
				log.Printf("Message: %s", message)
			}
		}
	}()

	// Send periodic player updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Simulate player movement
	playerPos := shared.Position{X: 0, Y: 0, Z: 0}
	rotation := shared.Rotation{X: 0, Y: 0, Z: 0}
	animation := shared.Idle

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// Simulate random movement and animation changes
			playerPos = shared.Position{
				X: rand.Float64() * 100, // Random X position
				Y: rand.Float64() * 100, // Random Y position
				Z: 0,
			}
			rotation = shared.Rotation{
				X: rand.Float64() * 360,
				Y: rand.Float64() * 360,
				Z: rand.Float64() * 360,
			}
			animation = shared.AnimationState(rand.Intn(4)) // Random animation 0-3

			// Create player update
			playerUpdate := shared.Player{
				Name:      "Player-" + sessionID,
				Position:  playerPos,
				Rotation:  rotation,
				Animation: animation,
			}

			// Send player update as JSON
			data, err := json.Marshal(playerUpdate)
			if err != nil {
				log.Printf("Failed to marshal player update: %v", err)
				continue
			}

			err = conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Println("write:", err)
				return
			}

			log.Printf("Sent update - Pos: (%.2f, %.2f), Rot: %.2f, Anim: %d",
				playerPos.X, playerPos.Y, rotation, animation)

		case <-interrupt:
			log.Println("Interrupt received, closing connection...")

			// Send close message
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}

			// Wait for server to close or timeout
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
