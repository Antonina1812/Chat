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

type Server struct {
	clients    map[net.Conn]Client
	broadcast  chan string
	register   chan Client
	unregister chan net.Conn
	mutex      sync.Mutex
}

func (s *Server) handleMessage() {
	for {
		select {
		case client := <-s.register:
			s.mutex.Lock()
			s.clients[client.Conn] = client
			s.mutex.Unlock()

		case conn := <-s.unregister:
			s.mutex.Lock()
			delete(s.clients, conn)
			s.mutex.Unlock()

		case message := <-s.broadcast:
			s.mutex.Lock()
			for conn := range s.clients {
				_, err := conn.Write([]byte(message + "\n"))
				if err != nil {
					conn.Close()
					delete(s.clients, conn)
				}
			}
			s.mutex.Unlock()
		}
	}
}

func (s *Server) handleClient(client Client) {
	defer client.Conn.Close()
	s.broadcast <- fmt.Sprintf("%s joined to chat", client.Name)

	reader := bufio.NewReader(client.Conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			s.unregister <- client.Conn
			s.broadcast <- fmt.Sprintf("%s left the chat", client.Name)
			return
		}
		if message == "exit" {
			s.unregister <- client.Conn
			s.broadcast <- fmt.Sprintf("%s left the chat", client.Name)
			return
		}
		s.broadcast <- fmt.Sprintf("%s: %s", client.Name, message)
	}
}

func main() {
	server := Server{
		clients:    make(map[net.Conn]Client),
		broadcast:  make(chan string),
		register:   make(chan Client),
		unregister: make(chan net.Conn),
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("server started with error", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on :8080")

	go server.handleMessage()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error of connection: ", err)
			continue
		}
		_, _ = conn.Write([]byte("Enter your name: "))
		name, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			conn.Close()
			continue
		}
		name = strings.TrimSpace(name)
		client := Client{
			Conn: conn,
			Name: name,
		}
		server.register <- client
		go server.handleClient(client)
	}
}
