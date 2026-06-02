// signaling/main.go - запускается на render.com
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Type string      `json:"type"`
	From string      `json:"from"`
	To   string      `json:"to"`
	Data interface{} `json:"data"`
}

var (
	clients = make(map[string]*websocket.Conn)
	mu      sync.RWMutex
)

func main() {
	http.HandleFunc("/signal", signalHandler)
	log.Println("Signaling server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func signalHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Получаем ID клиента из URL
	id := r.URL.Query().Get("id")
	if id == "" {
		return
	}

	mu.Lock()
	clients[id] = conn
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(clients, id)
		mu.Unlock()
	}()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		mu.RLock()
		targetConn, ok := clients[msg.To]
		mu.RUnlock()

		if ok {
			targetConn.WriteJSON(msg)
		}
	}
}
