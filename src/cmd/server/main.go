package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"game-server/internal/shared"

	"github.com/gorilla/websocket"
)

func loadConfig() shared.Config {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	tickRate := 20 // default
	if val := os.Getenv("TICK_RATE"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			tickRate = i
		}
	}

	config := shared.Config{
		Port:     port,
		Host:     host,
		TickRate: tickRate,
	}

	log.Printf("Server config - Host: %s, Port: %s, TickRate: %d", host, port, tickRate)

	return config
}

// Global player manager
type PlayerManager struct {
	// Map of player ID to Player
	players map[string]*shared.Player
	mutex   sync.RWMutex
}

func (pm *PlayerManager) AddPlayer(player *shared.Player) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.players[player.ID] = player
	log.Printf("Player %s connected. Total players: %d", player.ID, len(pm.players))
}

func (pm *PlayerManager) RemovePlayer(playerID string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	delete(pm.players, playerID)
	log.Printf("Player %s disconnected. Total players: %d", playerID, len(pm.players))
}

func (pm *PlayerManager) BroadcastToAll(data []byte) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	for _, player := range pm.players {
		if err := player.SendMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send to player %s: %v", player.ID, err)
		}
	}
}

func (pm *PlayerManager) GetPlayerCount() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return len(pm.players)
}

func (pm *PlayerManager) GetAllPlayers() []shared.Player {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	players := make([]shared.Player, 0, len(pm.players))
	for _, player := range pm.players {
		players = append(players, *player)
	}
	return players
}

func (pm *PlayerManager) UpdatePlayerState(playerID string, clientMsg shared.ClientMessage) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if player, exists := pm.players[playerID]; exists {
		player.Position = clientMsg.Position
		player.Rotation = clientMsg.Rotation
		player.Animation = clientMsg.Animation
		player.UpdateLastMessage()
	}
}

var playerManager = &PlayerManager{
	players: make(map[string]*shared.Player),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func gameLoop(tickRate int) {
	ticker := time.NewTicker(time.Duration(1000/tickRate) * time.Millisecond)
	defer ticker.Stop()

	tick := 0

	for range ticker.C {
		tick++

		// Create game world state
		world := shared.GameWorld{
			Tick:    tick,
			Time:    time.Now(),
			Players: playerManager.GetAllPlayers(),
		}

		// Serialize to JSON
		data, err := json.Marshal(world)
		if err != nil {
			log.Printf("Failed to marshal game state: %v", err)
			continue
		}

		// Broadcast to all players
		playerManager.BroadcastToAll(data)

		// Log every 100 ticks (5 seconds)
		if tick%100 == 0 {
			log.Printf("Game tick %d - Broadcasting to %d players", tick, len(world.Players))
			printStats()
		}

	}
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	// Extract game session ID from URL
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Create player
	player := &shared.Player{
		WSConn:        conn,
		ID:            sessionID,
		ConnectedTime: time.Now(),
		Name:          "Player-" + sessionID,
		Position:      shared.Position{X: 0, Y: 0, Z: 0},
		Rotation:      shared.Rotation{X: 0, Y: 0, Z: 0},
		Animation:     shared.Idle,
	}

	// Add to player manager
	playerManager.AddPlayer(player)
	defer playerManager.RemovePlayer(player.ID)

	// Handle incoming messages from this player
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Player %s disconnected: %v", player.ID, err)
			break
		}

		// Try to parse as ClientMessage (position/rotation/animation update)
		var clientMsg shared.ClientMessage
		if err := json.Unmarshal(msg, &clientMsg); err == nil {
			// Update player's game state
			playerManager.UpdatePlayerState(player.ID, clientMsg)
			log.Printf("Updated player %s - Pos: (%.2f, %.2f), Rot: %.2f, Anim: %d",
				player.ID, clientMsg.Position.X, clientMsg.Position.Y,
				clientMsg.Rotation, clientMsg.Animation)
		} else {
			// Treat as regular text message
			log.Printf("Received from %s: %s", player.ID, string(msg))

			// Echo back to sender (optional)
			response := fmt.Sprintf("Echo from %s: %s", player.ID, string(msg))
			if err := player.SendMessage(websocket.TextMessage, []byte(response)); err != nil {
				log.Printf("Write error: %v", err)
				break
			}
		}
	}
}

// Print memory, cpu usage and other stats
func printStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	log.Printf("Memory Usage: Alloc = %v MiB, TotalAlloc = %v MiB, Sys = %v MiB, NumGC = %v",
		memStats.Alloc/1024/1024, memStats.TotalAlloc/1024/1024, memStats.Sys/1024/1024, memStats.NumGC)

	cpuUsage := runtime.NumCPU()
	log.Printf("CPU Cores: %d", cpuUsage)
}

func main() {
	// Load configuration
	config := loadConfig()

	// Start the game loop in a separate goroutine
	go gameLoop(config.TickRate)

	address := fmt.Sprintf("%s:%s", config.Host, config.Port)
	http.HandleFunc("/ws", handleConnection)
	log.Printf("Server starting on %s...", address)
	log.Printf("Game loop running at %d TPS...", config.TickRate)
	log.Fatal(http.ListenAndServe(address, nil))
}
