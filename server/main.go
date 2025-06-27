package main

import (
	"bufio"
	"chat/database"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	_ "github.com/lib/pq"
)

type Client struct {
	Conn     net.Conn
	Name     string
	Password string
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

	listener, err := net.Listen("tcp", ":8080") // старт работы сервера на порту 8080
	if err != nil {
		fmt.Println("Error of server starting:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on :8080")

	connDb := "user=antoninabychkova dbname=chat host=localhost sslmode=disable" // подключение к базе данных
	db, err := sql.Open("postgres", connDb)
	if err != nil {
		server.broadcast <- fmt.Sprintln("db connection error")
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("db was successfuly connected")
	database.CreateTables(db)

	go server.handleMessages()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}

		conn.Write([]byte("Enter your name: \n"))

		name, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			conn.Close()
			continue
		}

		conn.Write([]byte("Enter your password: \n"))

		password, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			conn.Close()
			continue
		}

		name = strings.TrimSpace(name)
		client := Client{
			Conn:     conn,
			Name:     name,
			Password: password,
		}

		database.AddUser(conn, db, name, password)

		server.register <- client
		go server.handleClient(client)
	}
}

func (cs *ChatServer) handleClient(client Client) {
	defer client.Conn.Close()

	cs.broadcast <- fmt.Sprintf("%s joined", client.Name)
	reader := bufio.NewReader(client.Conn)

	chatHistoryFile := fmt.Sprintf("%s.txt", client.Name)

	file, err := os.OpenFile(chatHistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Ошибка при открытии файла: %v\n", err)
		return
	}
	defer file.Close()

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			cs.unregister <- client.Conn
			cs.broadcast <- fmt.Sprintf("%s left the chat", client.Name)
			return
		}

		message = strings.TrimSpace(message)

		switch {
		case message == "/exit":
			cs.unregister <- client.Conn
			cs.broadcast <- fmt.Sprintf("%s left the chat", client.Name)

			if _, err := file.WriteString(message + "\n"); err != nil {
				fmt.Printf("Ошибка при записи в файл: %v\n", err)
				continue
			}

			return
		case message == "/list":
			cs.sendClientList(client.Conn)

			if _, err := file.WriteString(message + "\n"); err != nil {
				fmt.Printf("Error of writing to file: %v\n", err)
				continue
			}

			continue
		case strings.HasPrefix(message, "/msg"):
			parts := strings.SplitN(message, " ", 3)
			if len(parts) < 3 {
				client.Conn.Write([]byte("Usage: /msg <name> <message>"))
				continue
			}
			recipientName := parts[1]
			privateMessage := parts[2]
			cs.privateChat(client.Name, recipientName, privateMessage)

			if _, err := file.WriteString(message + "\n"); err != nil {
				fmt.Printf("Error of writing to file: %v\n", err)
				continue
			}
		default:
			cs.broadcast <- fmt.Sprintf("%s: %s", client.Name, message)

			if _, err := file.WriteString(message + "\n"); err != nil {
				fmt.Printf("Error of writing to file: %v\n", err)
				continue
			}
		}
	}
}

func (cs *ChatServer) privateChat(senderName, recipientName, privateMessage string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	found := false

	for _, client := range cs.clients {
		if client.Name == recipientName {
			if _, err := client.Conn.Write([]byte(fmt.Sprintf("Private message from %s: %s\n", senderName, privateMessage))); err != nil {
				client.Conn.Close()
				delete(cs.clients, client.Conn)
			}
			found = true
			break
		}
	}
	if !found {
		for _, client := range cs.clients {
			if client.Name == senderName {
				if _, err := client.Conn.Write([]byte("User not found\n")); err != nil {
					client.Conn.Close()
					delete(cs.clients, client.Conn)
				}
				break
			}
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
