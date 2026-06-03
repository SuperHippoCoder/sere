// signaling/main.go - ЭТО ЗАЛИВАЕТСЯ НА RENDER.COM
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Храним соединения
var (
	serverConn *websocket.Conn
	clientConn *websocket.Conn
)

func main() {
	http.HandleFunc("/signal", handler)
	
	log.Println("✅ Signaling сервер запущен на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	role := r.URL.Query().Get("role")

	if role == "server" {
		serverConn = conn
		log.Println("✅ Сервер (друг) подключен к signaling")

		// Пересылаем все сообщения от друга тебе
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Сервер отключился")
				serverConn = nil
				break
			}
			if clientConn != nil {
				clientConn.WriteMessage(websocket.TextMessage, msg)
			}
		}
	} else if role == "client" {
		clientConn = conn
		log.Println("✅ Клиент (ты) подключен к signaling")

		// Пересылаем все сообщения от тебя другу
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Клиент отключился")
				clientConn = nil
				break
			}
			if serverConn != nil {
				serverConn.WriteMessage(websocket.TextMessage, msg)
			}
		}
	}
}
