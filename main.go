// signaling/main.go - ЗАЛИВАЕМ НА RENDER
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type SignalingMessage struct {
	Type      string          `json:"type"`      // "offer", "answer", "candidate"
	From      string          `json:"from"`      // "server" или "client"
	To        string          `json:"to"`        // "server" или "client"
	Data      json.RawMessage `json:"data"`      // SDP или ICE данные
}

var (
	clients   = make(map[string]*websocket.Conn)
	clientsMu sync.RWMutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/signal", signalHandler)
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/health", healthHandler)

	// Пинг самого себя
	go selfPing()

	log.Printf("✅ Signaling сервер запущен на порту %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func signalHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	role := r.URL.Query().Get("role")
	if role != "server" && role != "client" {
		log.Println("Неизвестная роль:", role)
		return
	}

	// Регистрируем клиента
	clientsMu.Lock()
	clients[role] = conn
	clientsMu.Unlock()

	log.Printf("✅ %s подключился к signaling", role)

	// Отправляем подтверждение
	conn.WriteJSON(map[string]string{"status": "connected", "role": role})

	// Обработка сообщений
	for {
		var msg SignalingMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("%s отключился", role)
			clientsMu.Lock()
			delete(clients, role)
			clientsMu.Unlock()
			break
		}

		// Пересылаем сообщение другому клиенту
		targetRole := "server"
		if role == "server" {
			targetRole = "client"
		}

		clientsMu.RLock()
		targetConn, exists := clients[targetRole]
		clientsMu.RUnlock()

		if exists {
			msg.From = role
			msg.To = targetRole
			err = targetConn.WriteJSON(msg)
			if err != nil {
				log.Printf("Ошибка отправки сообщения %s: %v", targetRole, err)
			} else {
				log.Printf("📡 Переслано %s -> %s: %s", role, targetRole, msg.Type)
			}
		}
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	clientsMu.RLock()
	serverConnected := clients["server"] != nil
	clientConnected := clients["client"] != nil
	clientsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","server":"` + 
		map[bool]string{true: "connected", false: "disconnected"}[serverConnected] + 
		`","client":"` + 
		map[bool]string{true: "connected", false: "disconnected"}[clientConnected] + 
		`"}`))
}

func selfPing() {
	for {
		time.Sleep(10 * time.Minute)
		url := "https://" + os.Getenv("RENDER_EXTERNAL_URL") + "/ping"
		if os.Getenv("RENDER_EXTERNAL_URL") == "" {
			continue
		}
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			log.Println("✅ Self-ping выполнен")
		}
	}
}
