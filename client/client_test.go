package main

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClientConnection(t *testing.T) {
	ln, err := net.Listen("tcp", ":8081") // прослушиваем порт 8081
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	connAccepted := make(chan struct{})

	go func() { // имитация работы сервера
		conn, _ := ln.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		close(connAccepted) // сигнализируем, что соединение принято
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", ":8081")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	select {
	case <-connAccepted:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for connection acceptance")
	}
}

func TestClientMessageHandling(t *testing.T) {
	ln, err := net.Listen("tcp", ":8082") // сервер
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverMsgs := make(chan string)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Error(err)
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			serverMsgs <- strings.TrimSpace(msg)
		}
	}()

	conn, err := net.Dial("tcp", ":8082") // клиент
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	testMsg := "test msg"
	_, err = conn.Write([]byte(testMsg + "\n"))
	if err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-serverMsgs:
		if msg != testMsg {
			t.Errorf("Expected message '%s', got '%s'", testMsg, msg)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Timeout waiting for server to receive message")
	}
}
