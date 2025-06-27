package database

import (
	"database/sql"
	"log"
	"net"
)

func CreateTables(db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    password VARCHAR(100) NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func AddUser(conn net.Conn, db *sql.DB, name, password string) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE name = $1", name).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	if count > 0 {
		conn.Write([]byte("User with such name is already exists\n"))
		return
	}

	_, err = db.Exec("INSERT INTO users (name, password) VALUES ($1, $2)", name, password)
	if err != nil {
		log.Fatal(err)
	}

	conn.Write([]byte("User sign up successfully\n"))
}
