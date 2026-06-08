package services

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(path string) {
	var err error
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}

	createTables := `
	CREATE TABLE IF NOT EXISTS credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		login TEXT,
		password TEXT,
		ip TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS clients (
		mac TEXT PRIMARY KEY,
		ip TEXT,
		hostname TEXT,
		os TEXT,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(createTables)
	if err != nil {
		log.Fatalf("Error creating tables: %q: %s\n", err, createTables)
	}
}

func SaveCredential(login, password, ip string) error {
	_, err := DB.Exec("INSERT INTO credentials (login, password, ip) VALUES (?, ?, ?)", login, password, ip)
	return err
}

func UpsertClient(mac, ip, hostname, os string) error {
	_, err := DB.Exec(`INSERT INTO clients (mac, ip, hostname, os, last_seen)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(mac) DO UPDATE SET
		ip=excluded.ip, hostname=excluded.hostname, os=excluded.os, last_seen=CURRENT_TIMESTAMP`,
		mac, ip, hostname, os)
	return err
}
