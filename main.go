// signaling/main.go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	serverConn *websocket.Conn
	clientConn *websocket.Conn
)

func main() {
	// ПИНГ САМОГО СЕБЯ КАЖДЫЕ 10 МИНУТ
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

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	http.HandleFunc("/ws", handler)
	log.Println("Signaling server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	role := r.URL.Query().Get("role")

	if role == "server" {
		serverConn = conn
		log.Println("✅ Server connected")

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if clientConn != nil {
				clientConn.WriteMessage(websocket.TextMessage, msg)
			}
		}
	} else if role == "client" {
		clientConn = conn
		log.Println("✅ Client connected")

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if serverConn != nil {
				serverConn.WriteMessage(websocket.TextMessage, msg)
			}
		}
	}
}
