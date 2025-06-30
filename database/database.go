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
    password VARCHAR(100) NOT NULL,
	role VARCHAR(100) NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func AddUser(conn net.Conn, db *sql.DB, name, password, role string) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE name = $1", name).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	if count > 0 {
		conn.Write([]byte("User sign in successfully\n"))
		return
	}

	_, err = db.Exec("INSERT INTO users (name, password, role) VALUES ($1, $2, $3)", name, password, role)
	if err != nil {
		log.Fatal(err)
	}

	conn.Write([]byte("User sign up successfully\n"))
}

func DeleteUser(conn net.Conn, db *sql.DB, name string) {
	result, err := db.Exec(`DELETE FROM users WHERE name = $1`, name)
	if err != nil {
		log.Fatal(err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal()
	}

	if rowsAffected > 0 { // проверяем сколько строчек было затронуто
		conn.Write([]byte("User is deleted\n"))
	} else {
		conn.Write([]byte("User was not found\n"))
	}
}
