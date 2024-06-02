package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlClient struct {
	DB       *sql.DB
	Tx       *sql.Tx
	DmlCount uint64
}

func NewMysqlClient(settings map[string]string) *MysqlClient {
	logger := log.New(os.Stdout, "mysqlclient: ", log.LstdFlags)

	// Build the DSN from settings
	dsn := settings["username"] + ":" + settings["password"] + "@tcp(" + settings["host"] + ")/" + settings["database"]
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Fatal("Failed to connect to database: ", err)
	}

	db.SetConnMaxLifetime(time.Minute * 10)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(10)

	return &MysqlClient{
		DB:       db, //user:password@tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true
		DmlCount: 0,
	}
}

func (mc *MysqlClient) Begin() error {
	fmt.Println("start trunsaction")

	var err error
	mc.Tx, err = mc.DB.Begin()
	if err != nil {
		return err
	}
	return nil
}

func (mc *MysqlClient) ExecuteDML(query string, args []interface{}) error {
	if mc.Tx != nil {
		// fmt.Println("cliemt execdml ", query)
		_, err := mc.Tx.Exec(query, args...)
		if err != nil {
			mc.Tx.Rollback()
			return err
		}
	}
	return nil
}

func (mc *MysqlClient) ExecuteNonDML(db string, sqlText string) {
	// client.Commit()
	// if db != "" {
	// 	client.Con.Exec("USE " + db)
	// }
	// if _, err := client.Con.Exec(sqlText); err != nil {
	// 	// client.Logger.Println("Error executing non-DML statement:", err)
	// }
}

func (mc *MysqlClient) Commit() error {
	if mc.Tx != nil {
		err := mc.Tx.Commit()
		if err != nil {
			return err
		}
		mc.Tx = nil
	}
	return nil
}

// func main() {
// 	settings := map[string]string{
// 		"username": "user",
// 		"password": "password",
// 		"host":     "localhost:3306",
// 		"database": "mydb",
// 	}

// 	client := NewMysqlClient(settings)
// 	defer client.Con.Close()

// 	// Example DML operation
// 	client.Begin()
// 	client.ExecuteDML("INSERT INTO users (name) VALUES (?)", []interface{}{"John"})
// 	client.Commit()

// 	// Example non-DML operation
// 	client.ExecuteNonDML("", "SET @a = 1")
// }
