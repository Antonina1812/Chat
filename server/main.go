package main

import (
	"bufio"
	"chat/database"
	s "chat/structures"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type ChatServer struct {
	clients    map[net.Conn]s.Client
	broadcast  chan string
	register   chan s.Client
	unregister chan net.Conn
	mutex      sync.Mutex
}

func main() {
	server := ChatServer{
		clients:    make(map[net.Conn]s.Client),
		broadcast:  make(chan string),
		register:   make(chan s.Client),
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

		client := server.handleIncomingData(conn, db)
		clientsList := database.GetUsersList(db, conn, client)

		server.register <- client
		go server.handleClient(db, client, clientsList)
	}
}

func (cs *ChatServer) handleIncomingData(conn net.Conn, db *sql.DB) s.Client {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	conn.Write([]byte("Enter your name: \n"))

	name, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		conn.Close()
	}

	conn.Write([]byte("Enter your password: \n"))

	password, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		conn.Close()
	}

	conn.Write([]byte("Enter your role (admin or guest): \n"))
	role := cs.setRole(conn)

	name = strings.TrimSpace(name)
	client := s.Client{
		Conn:     conn,
		Name:     name,
		Password: password,
		Role:     role,
	}

	database.AddUser(conn, db, name, password, role)
	return client
}

func (cs *ChatServer) setRole(conn net.Conn) string {
	for {
		newRole, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			conn.Close()
		}

		newRole = strings.TrimSpace(newRole)

		if newRole == "admin" || newRole == "guest" {
			role := newRole
			return role
		} else {
			conn.Write([]byte("Such role doesn't exist\n"))
			continue
		}
	}
}

func (cs *ChatServer) handleClient(db *sql.DB, client s.Client, clientsList []s.Client) {
	defer client.Conn.Close()

	cs.broadcast <- fmt.Sprintf("%s joined", client.Name)
	reader := bufio.NewReader(client.Conn)

	chatHistoryFile := fmt.Sprintf("storage/%s.txt", client.Name)

	file, err := os.OpenFile(chatHistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error of writing to file: %v\n", err)
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
		time := time.Now().Format("2006-01-02 15:04:05")

		switch {
		case message == "/exit":
			cs.unregister <- client.Conn
			cs.broadcast <- fmt.Sprintf("%s left the chat", client.Name)

			cs.writeToFile(file, time, client.Name, message)
			return

		case message == "/list":
			cs.sendClientList(db, client.Conn, clientsList)

			cs.writeToFile(file, time, client.Name, message)
			continue

		case strings.HasPrefix(message, "/msg"):
			parts := strings.SplitN(message, " ", 3)
			if len(parts) < 3 {
				client.Conn.Write([]byte("Usage: /msg <name> <message>\n"))
				continue
			}
			recipientName := parts[1]
			privateMessage := parts[2]
			cs.privateChat(client.Name, recipientName, privateMessage)

			cs.writeToFile(file, time, client.Name, message)
			continue

		case strings.HasPrefix(message, "/kick"):
			parts := strings.Split(message, " ")
			if len(parts) < 2 {
				client.Conn.Write([]byte("Usage: /kick <name>\n"))
				continue
			}

			username := parts[1]

			if client.Role == "guest" {
				client.Conn.Write([]byte("You don't have enough rights\n"))
				continue
			} else if client.Role == "admin" {
				database.DeleteUser(client.Conn, db, username)
				for _, client := range cs.clients {
					if username == client.Name {
						client.Conn.Close()
						return
					}
				}
			}
			cs.writeToFile(file, time, client.Name, message)
			continue

		default:
			cs.broadcast <- fmt.Sprintf("%s %s: %s", time, client.Name, message)
			cs.writeToFile(file, time, client.Name, message)
			continue
		}
	}
}

func (cs *ChatServer) writeToFile(file *os.File, time, name, message string) {
	if _, err := file.WriteString(time + " " + name + ": " + message + "\n"); err != nil {
		fmt.Printf("Error of writing to file: %v\n", err)
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

func (cs *ChatServer) sendClientList(db *sql.DB, conn net.Conn, clientsList []s.Client) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	var builder strings.Builder

	builder.WriteString("=== All participants ===\n")
	for _, client := range clientsList {
		builder.WriteString(fmt.Sprintf("• %s (%s)\n", client.Name, client.Role))
	}
	builder.WriteString(fmt.Sprintf("Total: %d\n", database.UsersCount(db)))

	builder.WriteString("\n=== Active participants ===\n")
	for _, client := range cs.clients {
		builder.WriteString(fmt.Sprintf("• %s (%s)\n", client.Name, client.Role))
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
