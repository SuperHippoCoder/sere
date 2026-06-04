// render.go - залить на Render как Web Service
package main

import (
    "encoding/gob"
    "log"
    "net/http"
    "os"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

type Payload struct {
    Data []byte
    X, Y int
}

type Command struct {
    Type   string
    X, Y   int
    Button string
    Key    string
    Shift  bool
}

var (
    clients   = make(map[*websocket.Conn]bool)
    clientsMu sync.RWMutex
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    // Простой WebSocket релей
    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Println("WebSocket upgrade error:", err)
            return
        }
        defer conn.Close()

        // Регистрируем клиента
        clientsMu.Lock()
        clients[conn] = true
        clientsMu.Unlock()

        log.Printf("✅ Клиент подключился. Всего: %d", len(clients))

        // Отправляем подтверждение
        conn.WriteMessage(websocket.TextMessage, []byte("connected"))

        // Читаем и пересылаем всем
        for {
            _, msg, err := conn.ReadMessage()
            if err != nil {
                clientsMu.Lock()
                delete(clients, conn)
                clientsMu.Unlock()
                log.Printf("❌ Клиент отключился. Всего: %d", len(clients))
                break
            }

            // Пересылаем всем остальным клиентам
            clientsMu.RLock()
            for client := range clients {
                if client != conn {
                    err := client.WriteMessage(websocket.BinaryMessage, msg)
                    if err != nil {
                        client.Close()
                        delete(clients, client)
                    }
                }
            }
            clientsMu.RUnlock()
        }
    })

    // Пинг для предотвращения сна Render
    http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })

    // health check
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        clientsMu.RLock()
        count := len(clients)
        clientsMu.RUnlock()
        w.Write([]byte(`{"status":"ok","clients":` + string(rune(count)) + `}`))
    })

    // Авто-пинг каждые 10 минут
    go func() {
        for {
            time.Sleep(10 * time.Minute)
            url := "https://" + os.Getenv("RENDER_EXTERNAL_URL") + "/ping"
            if os.Getenv("RENDER_EXTERNAL_URL") != "" {
                http.Get(url)
                log.Println("✅ Self-ping выполнен")
            }
        }
    }()

    log.Printf("✅ Сервер запущен на порту %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
