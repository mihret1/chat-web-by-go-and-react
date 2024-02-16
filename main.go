// server.go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)
// The websocket.Upgrader configuration is essential when setting up WebSocket connections. 
// It allows customization of buffer sizes and provides a mechanism to control 
// the acceptance of WebSocket connections based on their origin.
 var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
// the Client type encapsulates the WebSocket connection 
// and a channel for sending messages to a specific client 
// in a concurrent and asynchronous manner. This is a common pattern 
// in Go for handling multiple connections concurrently in networked applications.
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) read() {
	defer func() {
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// Broadcast the message to all clients
		broadcast <- message
	}
}

func (c *Client) write() {
	defer func() {
		c.conn.Close()
	}()

	for {
		message, ok := <-c.send
		if !ok {
			break
		}

		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
}

var clients = make(map[*Client]bool)
var broadcast = make(chan []byte)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}
	clients[client] = true

	go client.read()
	go client.write()
}

func handleMessages() {
	for {
		message := <-broadcast
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	go handleMessages()

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
