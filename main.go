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

	log.Println("✅ Клиент подключился")

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			mu.Lock()
			delete(clients, ws)
			mu.Unlock()
			log.Println("❌ Клиент отключился")
			break
		}
		// Пересылаем сообщение всем остальным подключенным клиентам
		mu.Lock()
		for client := range clients {
			if client != ws {
				err := client.WriteMessage(websocket.BinaryMessage, msg)
				if err != nil {
					client.Close()
					delete(clients, client)
				}
			}
		}
		mu.Unlock()
	}
}

func main() {
	// Пинг самого себя каждые 10 минут чтобы сервер не засыпал
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			resp, err := http.Get("https://sere-wb5r.onrender.com/ping")
			if err == nil {
				resp.Body.Close()
				log.Println("✅ Пинг выполнен, сервер активен")
			} else {
				log.Println("❌ Ошибка пинга:", err)
			}
		}
	}()

	// Эндпоинт для пинга
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	http.HandleFunc("/ws", handleConnections)
	
	log.Println("✅ Signaling сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
