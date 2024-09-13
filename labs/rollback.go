package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root:root_password@tcp(192.168.161.93:33028)/")

	if err != nil {
		fmt.Println(err.Error())
	}

	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(2)

	// if err := tx.Commit(); err != nil {
	// 	fmt.Println("null commit")
	// 	fmt.Println(err)
	// }
	tx, _ := db.Begin()

	if _, err := tx.Exec("USE abc123"); err != nil {
		fmt.Println(3333, err)
		if err := tx.Rollback(); err != nil {
			fmt.Println(1111, err)
		} else {
			fmt.Println("ok roolback")
		}

		if err := tx.Rollback(); err != nil {
			fmt.Println(1111, err)
		} else {
			fmt.Println("ok roolback")
		}
		// if _, err := tx.Exec("USE abc1213"); err != nil {
		// 	fmt.Println(1112221, err)
		// }
	}

}
