package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer ws.Close()

	mu.Lock()
	clients[ws] = true
	mu.Unlock()

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			mu.Lock()
			delete(clients, ws)
			mu.Unlock()
			break
		}
		mu.Lock()
		for client := range clients {
			if client != ws {
				client.WriteMessage(websocket.BinaryMessage, msg)
			}
		}
		mu.Unlock()
	}
}

func main() {
	// Эндпоинт для пинга
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// Пинг самого себя каждые 10 минут
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			resp, err := http.Get("https://sere-wb5r.onrender.com/ping")
			if err == nil {
				resp.Body.Close()
				log.Println("✅ Пинг выполнен")
			}
		}
	}()

	http.HandleFunc("/ws", handleConnections)
	log.Println("✅ Сервер запущен")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
