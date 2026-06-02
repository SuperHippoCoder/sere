// server/main.go - запускается на render.com
package main

import (
	"bytes"
	"encoding/gob"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/gorilla/websocket"
	"golang.org/x/image/draw"
)

type Payload struct {
	Data []byte
	X, Y int
}

type Command struct {
	Type   string
	X, Y   int
	Button string
	Scroll int
	Key    string
	Shift  bool
	Text   string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)

func main() {
	// Запускаем WebSocket сервер
	http.HandleFunc("/ws", wsHandler)
	
	// ЗАПУСКАЕМ ПИНГ САМОГО СЕБЯ КАЖДЫЕ 10 МИНУТ
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			// Просто делаем HTTP запрос к самому себе
			resp, err := http.Get("https://sere-wb5r.onrender.com/ping")
			if err == nil {
				resp.Body.Close()
				log.Println("✅ Пинг выполнен, сервер не заснет")
			} else {
				log.Println("❌ Ошибка пинга:", err)
			}
		}
	}()

	// Эндпоинт для пинга
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	log.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true
	defer delete(clients, conn)

	log.Println("✅ Клиент подключился")

	// Канал для команд от клиента
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var cmd Command
			gob.NewDecoder(bytes.NewReader(msg)).Decode(&cmd)
			handleCommand(cmd)
		}
	}()

	// Отправка видео всем клиентам
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		bit := robotgo.CaptureScreen()
		img := robotgo.ToImage(bit)

		resized := image.NewRGBA(image.Rect(0, 0, 1280, 720))
		draw.ApproxBiLinear.Scale(resized, resized.Bounds(), img, img.Bounds(), draw.Over, nil)

		buf := new(bytes.Buffer)
		jpeg.Encode(buf, resized, &jpeg.Options{Quality: 40})

		mx, my := robotgo.GetMousePos()

		gobBuf := new(bytes.Buffer)
		gob.NewEncoder(gobBuf).Encode(Payload{Data: buf.Bytes(), X: mx, Y: my})

		for client := range clients {
			client.WriteMessage(websocket.BinaryMessage, gobBuf.Bytes())
		}

		robotgo.FreeBitmap(bit)
	}
}

func handleCommand(cmd Command) {
	switch cmd.Type {
	case "MOVE":
		w, h := robotgo.GetScreenSize()
		robotgo.Move(cmd.X*w/1280, cmd.Y*h/720)
	case "CLICK":
		robotgo.Click(cmd.Button)
	case "SCROLL":
		robotgo.Scroll(cmd.Scroll, 0)
	case "KEY":
		robotgo.KeyTap(cmd.Key)
	}
}
