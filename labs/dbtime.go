package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func query(db *sql.DB) {
	if _, err := db.Exec("SET foreign_key_checks = 0"); err != nil {
		fmt.Println(err)

	}

	if _, err := db.Exec("SET time_zone = '+00:00'"); err != nil {
		fmt.Println(err)
	}
	rows, err := db.Query("SELECT CONNECTION_ID(), @@global.time_zone, @@session.time_zone, @@global.foreign_key_checks, @@session.foreign_key_checks;")
	if err != nil {
		fmt.Println(err)

	}
	defer rows.Close()

	var connectionid, globalTimeZone, sessionTimeZone, a, b string
	if rows.Next() {
		err := rows.Scan(&connectionid, &globalTimeZone, &sessionTimeZone, &a, &b)
		if err != nil {
			log.Fatal("Error scanning rows: ", err)
		}
		fmt.Printf("Connectionid: %s\n", connectionid)
		fmt.Printf("Global Time Zone: %s\n", globalTimeZone)
		fmt.Printf("Session Time Zone: %s\n", sessionTimeZone)
		fmt.Printf("Session Time Zone: %s\n", a)

		fmt.Printf("Session Time Zone: %s\n", b)

	}

	if err = rows.Err(); err != nil {
		log.Fatal("Error during rows iteration: ", err)
	}
}

func main() {
	db, err := sql.Open("mysql", "root:root_password@tcp(db1:3306)/")
	if err != nil {
		log.Fatal(err)
	}
	db.SetConnMaxLifetime(time.Second * 10)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(2)

	query(db)

	time.Sleep(time.Second * 15)
	query(db)

}
