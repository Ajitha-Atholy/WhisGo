package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	conn *websocket.Conn
}

var (
	clients   = make(map[*Client]bool)
	broadcast = make(chan []byte)
	mutex     = &sync.Mutex{}
)

func handleClients() {
	for {
		msg := <-broadcast
		mutex.Lock()
		for client := range clients {
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println(err)
				client.conn.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{conn: conn}
	mutex.Lock()
	clients[client] = true
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		delete(clients, client)
		mutex.Unlock()
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
		}
		broadcast <- message
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir(".")))
	go handleClients()
	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error on Listen and Serve: ", err)
	}
}
