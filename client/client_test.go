package main

import (
	"net"
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
