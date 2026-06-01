package main

import (
    "log"
    "net/http"
    "sync"
    "time" // Добавляем пакет для таймера

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex

func handleConnections(w http.ResponseWriter, r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return // Лучше не делать log.Fatal внутри обработчика, иначе сервер упадет при ошибке клиента
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
    // --- ЭТА ЧАСТЬ НЕ ДАЕТ СЕРВЕРУ ЗАСНУТЬ ---
    go func() {
        for {
            // Замени на свой домен Render
            http.Get("https://sere-wb5r.onrender.com") 
            time.Sleep(10 * time.Minute) 
        }
    }()
    // ----------------------------------------

    http.HandleFunc("/ws", handleConnections)
    log.Println("Server started on :8080") // Для логов
    log.Fatal(http.ListenAndServe(":8080", nil))
}
