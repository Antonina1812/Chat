package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

type Client struct {
	Conn net.Conn
	Name string
}

type ChatServer struct {
	clients    map[net.Conn]Client
	broadcast  chan string
	register   chan Client
	unregister chan net.Conn
	mutex      sync.Mutex
}

func main() {
	server := ChatServer{
		clients:    make(map[net.Conn]Client),
		broadcast:  make(chan string),
		register:   make(chan Client),
		unregister: make(chan net.Conn),
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error of server starting:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on :8080")
	go server.handleMessages()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}

		_, _ = conn.Write([]byte("Enter your name: "))
		name, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			conn.Close()
			continue
		}

		name = strings.TrimSpace(name)
		client := Client{Conn: conn, Name: name}
		server.register <- client
		go server.handleClient(client)
	}
}

func (cs *ChatServer) handleClient(client Client) {
	defer client.Conn.Close()

	cs.broadcast <- fmt.Sprintf("%s joined", client.Name)

	reader := bufio.NewReader(client.Conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			cs.unregister <- client.Conn
			cs.broadcast <- fmt.Sprintf("%s left the chat", client.Name)
			return
		}

		message = strings.TrimSpace(message)

		// Обработка команд
		switch message {
		case "/exit":
			cs.unregister <- client.Conn
			cs.broadcast <- fmt.Sprintf("%s left the chat", client.Name)
			return
		case "/list":
			cs.sendClientList(client.Conn)
			continue
		default:
			cs.broadcast <- fmt.Sprintf("%s: %s", client.Name, message)
		}
	}
}

func (cs *ChatServer) sendClientList(conn net.Conn) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	var builder strings.Builder
	builder.WriteString("=== Participants ===\n")
	for _, client := range cs.clients {
		builder.WriteString(fmt.Sprintf("• %s\n", client.Name))
	}
	builder.WriteString(fmt.Sprintf("Total: %d\n", len(cs.clients)))

	_, err := conn.Write([]byte(builder.String()))
	if err != nil {
		conn.Close()
		delete(cs.clients, conn)
	}
}

func (cs *ChatServer) handleMessages() {
	for {
		select {
		case client := <-cs.register:
			cs.mutex.Lock()
			cs.clients[client.Conn] = client
			cs.mutex.Unlock()

		case conn := <-cs.unregister:
			cs.mutex.Lock()
			delete(cs.clients, conn)
			cs.mutex.Unlock()

		case message := <-cs.broadcast:
			cs.mutex.Lock()
			for conn := range cs.clients {
				_, err := conn.Write([]byte(message + "\n"))
				if err != nil {
					conn.Close()
					delete(cs.clients, conn)
				}
			}
			cs.mutex.Unlock()
		}
	}
}

// go run main.go
