package main

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

type MysqlClient struct {
	db         *sql.DB
	tx         *sql.Tx
	Logger     *Logger
	SkipErrors []uint16
}

func NewMysqlClient(logLevel int, dsn string, skipErrorsStr *string) (*MysqlClient, error) {
	Logger := NewLogger(logLevel, "mysql-client")

	Logger.Info("dsn: " + dsn)
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, err
	}

	skipErrors := []uint16{}
	if skipErrorsStr != nil {
		slice, err := ConvertStringToUint16Slice(*skipErrorsStr)
		if err != nil {
			return nil, err
		}
		skipErrors = append(skipErrors, slice...)
	}

	db.SetConnMaxLifetime(time.Minute * 1)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(2)

	myclient := &MysqlClient{
		db:         db,
		tx:         nil,
		Logger:     Logger,
		SkipErrors: skipErrors,
	}

	return myclient, nil
}

func (mc *MysqlClient) SkipError(err error) error {
	if merr, ok := err.(*mysql.MySQLError); ok {
		for _, v := range mc.SkipErrors {
			if v == merr.Number {
				mc.Logger.Error("Skip %s.", err.Error())
				return nil
			}
		}
	}
	return err
}

func (mc *MysqlClient) Close() error {
	if err := mc.db.Close(); err != nil {
		mc.Logger.Error("Connection close: ", err.Error())
		return err
	}
	return nil
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
		if mc.SkipError(err) != nil {
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

	if mc.SkipError(err) != nil {
		return err
	}

	return mc.Commit()
}

func (mc *MysqlClient) ExecuteOnDatabase(query string) error {
	mc.Commit()

	mc.Begin()

	_, err := mc.tx.Exec(query)

	if mc.SkipError(err) != nil {
		mc.Logger.Error("query: '%s': %s", query, err.Error())
		mc.Rollback()
		return err
	}

	return mc.Commit()
}

func (mc *MysqlClient) Commit() error {
	if mc.tx != nil {
		err := mc.tx.Commit()
		if err != nil {
			mc.Logger.Error("Commit: ", err.Error())
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
