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
	Logger *Logger
}

func NewMysqlClient(logLevel int, dsn string) (*MysqlClient, error) {
	Logger := NewLogger(logLevel, "mysql-client")

	Logger.Info("dsn: " + dsn)
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 1)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(2)

	myclient := &MysqlClient{
		db:     db,
		tx:     nil,
		Logger: Logger,
	}

	return myclient, nil
}

func (mc *MysqlClient) Close() error {
	err := mc.db.Close()
	if err != nil {
		mc.Logger.Error("connection close error: " + err.Error())
	}
	return err
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
		_, err := mc.tx.Exec("USE " + db)

		if err != nil {
			return err
		}
	}

	_, err := mc.tx.Exec(query)

	if err != nil {
		return err
	}

	return mc.Commit()
}

func (mc *MysqlClient) ExecuteOnDatabase(query string) error {
	mc.Commit()

	mc.Begin()

	_, err := mc.tx.Exec(query)

	if err != nil {
		mc.Logger.Error(fmt.Sprintf("query: '%s', error: %s", query, err.Error()))
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
