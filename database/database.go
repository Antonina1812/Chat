package database

import (
	"database/sql"
	"log"
	"net"
)

func CreateTables(db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS roles (
		id SERIAL PRIMARY KEY,
		role VARCHAR(100) NOT NULL
		)`)
	if err != nil {
		log.Fatal(err)
		return
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    password VARCHAR(100) NOT NULL,
	role_id INTEGER NOT NULL,
	FOREIGN KEY (role_id) REFERENCES roles(id)
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

	var role_id int
	err = db.QueryRow("SELECT id FROM roles WHERE role = $1", role).Scan(&role_id)
	if err != nil {
		conn.Write([]byte("Error: role not found in database\n"))
		return
	}

	_, err = db.Exec("INSERT INTO users (name, password, role_id) VALUES ($1, $2, $3)", name, password, role_id)
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
