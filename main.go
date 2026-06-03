// signaling/main.go - ИСПРАВЛЕННЫЙ ДЛЯ RENDER
package main

import (
	"log"
	"net/http"
	"os"
	"time"

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
	// Render передает порт через переменную окружения PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/signal", handler)

	// Пинг самого себя
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			url := "https://sere-wb5r.onrender.com/ping"
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				log.Println("✅ Пинг выполнен")
			} else {
				log.Println("❌ Ошибка пинга:", err)
			}
		}
	}()

	// Эндпоинт для пинга
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	log.Printf("✅ Signaling сервер запущен на порту %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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
