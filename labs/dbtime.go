package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root:root_password@tcp(db2:3306)/?parseTime=true&loc=America%2FLos_Angeles")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	utcTime := time.Date(2023, 6, 1, 15, 4, 5, 0, time.UTC)
	timestamp := utcTime.Format("2006-01-02 15:04:05")

	_, err = db.Exec("INSERT INTO test1.your_table (your_timestamp_column) VALUES (?)", timestamp)
	if err != nil {
		log.Fatal("Failed to insert timestamp:", err)
	}
}
