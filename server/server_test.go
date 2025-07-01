package main

import (
	"bufio"
	s "chat/structures"
	"net"
	"testing"
	"time"
)

func TestServerStart(t *testing.T) {
	go main() // запускаем сервер в горутине
	time.Sleep(time.Second * 2)

	conn, err := net.Dial("tcp", ":8080") // пытаемся подключиться к серверу
	if err != nil {
		t.Fatal("Failed to connect to server")
	}
	defer conn.Close()
}

func TestHandleIncomingData(t *testing.T) {
	server := ChatServer{
		clients:    make(map[net.Conn]s.Client),
		broadcast:  make(chan string),
		register:   make(chan s.Client),
		unregister: make(chan net.Conn),
	}

	clientConn, serverConn := net.Pipe() /* создает пару связанных между собой in-memory (в памяти) сетевых соединений
	Они имитируют обычные сетевые соединения, но данные, записанные в одно соединение,
	напрямую передаются на чтение в другое, без использования реальной сетевой инфраструктуры (например, TCP/IP)*/

	/* Основное отличие от реальных сетевых соединений (например, созданных с помощью net.Dial() или net.Listen())
	заключается в том, что net.Pipe() не использует сокеты операционной системы и не требует сетевого интерфейса
	Все данные передаются непосредственно в памяти.*/
	defer clientConn.Close()
	defer serverConn.Close()

	done := make(chan struct{})

	go func() {
		defer close(done) // сигнал завершения первой горутины

		writer := bufio.NewWriter(clientConn)
		writer.WriteString("testuser\n")
		writer.WriteString("testpass\n")
		writer.WriteString("guest\n")
		writer.Flush()
	}()

	go func() {
		reader := bufio.NewReader(clientConn)
		for {
			if _, err := reader.ReadString('\n'); err != nil {
				return
			}

		}
	}()

	var client s.Client
	select {
	case client = <-func() chan s.Client { // анонимная функция, которая возвращает канал, и затем пытается прочитать данные из этого канала
		ch := make(chan s.Client)
		go func() {
			ch <- server.handleIncomingData(serverConn, nil) // Эта операция *блокирует* горутину до тех пор, пока кто-то не прочитает данные из канала ch
		}()
		return ch
	}():
	case <-time.After(6 * time.Second):
		t.Fatal("Test timed out")
	}

	/*select ждет либо получения данных из канала ch (результат обработки данных сервером),
	либо истечения времени ожидания в 2 секунды.
	Если данные получены из канала ch, они присваиваются переменной client
	Если время ожидания истекает, тест завершается с ошибкой.*/

	if client.Name != "testuser" {
		t.Errorf("Expected client name 'testuser', got '%s'", client.Name)
	}
	if client.Role != "guest" {
		t.Errorf("Expected client role 'guest', got '%s'", client.Role)
	}

	<-done
}
