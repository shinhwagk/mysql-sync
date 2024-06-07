package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlClient struct {
	db     *sql.DB
	tx     *sql.Tx
	logger *Logger
}

func NewMysqlClient(logLevel int, dsn string) *MysqlClient {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		fmt.Println(err.Error())
		// logger.Fatal("Failed to connect to database: ", err)
	}

	db.SetConnMaxLifetime(time.Minute * 1)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(2)

	myclient := &MysqlClient{
		db:     db,
		tx:     nil,
		logger: NewLogger(logLevel, "mysql-client"),
	}

	// go func() {
	// 	<-ctx.Done()
	// 	myclient.Close()
	// }()

	return myclient
}

func (mc *MysqlClient) Close() error {
	mc.logger.Info("close")
	return mc.db.Close()
}

func (mc *MysqlClient) Begin() error {
	var err error
	if mc.tx == nil {
		if mc.tx, err = mc.db.Begin(); err != nil {
			return err
		}
	}
	return nil
}

func (mc *MysqlClient) ExecuteDML(query string, args []interface{}) error {
	if mc.tx != nil {
		_, err := mc.tx.Exec(query, args...)
		if err != nil {
			return mc.Rollback()
		}
	}
	return nil
}

func (mc *MysqlClient) ExecuteOnTable(db string, query string) error {
	mc.Commit()

	mc.Begin()

	if db != "" {
		mc.tx.Exec("USE " + db)
	}

	_, err := mc.tx.Exec(query)

	if err != nil {
		fmt.Println(err)
	}

	return mc.Commit()
}

func (mc *MysqlClient) ExecuteOnDatabase(query string) error {
	mc.Commit()

	mc.Begin()

	_, err := mc.tx.Exec(query)

	if err != nil {
		mc.logger.Error(fmt.Sprintf("query: '%s', error: %s", query, err.Error()))
		mc.Rollback()
		return err
	}

	return mc.Commit()
}

func (mc *MysqlClient) Commit() error {
	if mc.tx != nil {
		err := mc.tx.Commit()
		if err != nil {
			return err
		}
		mc.tx = nil
	}
	return nil
}

func (mc *MysqlClient) Rollback() error {
	if mc.tx != nil {
		err := mc.tx.Rollback()
		if err != nil {
			return err
		}
		mc.tx = nil
	}
	return nil
}
