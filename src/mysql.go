package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

const (
	StateNULL   = "null"
	StateDML    = "dml"
	StateDDL    = "ddl"
	StateBEGIN  = "begin"
	StateCOMMIT = "commit"
)

type MysqlClient struct {
	db         *sql.DB
	tx         *sql.Tx
	Logger     *Logger
	SkipErrors []uint16
	State      string
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
		State:      StateNULL,
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
	if mc.State == StateCOMMIT || mc.State == StateNULL {
		var err error
		if mc.tx == nil {
			if mc.tx, err = mc.db.Begin(); err != nil {
				mc.Logger.Error("execute Begin: %s", err.Error())
				return err
			}
			mc.State = StateBEGIN
		} else {
			return fmt.Errorf("execute Begin: tx is not nil")
		}
	} else {
		return fmt.Errorf("execute Begin: last state is '%s'", mc.State)
	}
	return nil
}

func (mc *MysqlClient) ExecuteDML(query string, args []interface{}) error {
	if mc.State == StateBEGIN || mc.State == StateDML {
		if mc.tx != nil {
			_, err := mc.tx.Exec(query, args...)
			if mc.SkipError(err) != nil {
				mc.Logger.Error("execute DML: %s, Query: %s, Params: %v.", err.Error(), query, args)
				return err
			}
			mc.State = StateDML
		} else {
			return fmt.Errorf("execute DML: tx is not nil")
		}
	} else {
		return fmt.Errorf("execute DML: last state is '%s'", mc.State)
	}
	return nil
}

func (mc *MysqlClient) ExecuteOnTable(db string, query string) error {
	mc.Begin()

	if db != "" {
		if _, err := mc.tx.Exec("USE " + db); err != nil {
			return err
		}
	}

	if _, err := mc.tx.Exec(query); mc.SkipError(err) != nil {
		mc.Logger.Error("execute DDL: %s, Database: %s Query: %s.", err.Error(), db, query)
		return err
	}

	mc.State = StateDDL

	return mc.Commit()
}

func (mc *MysqlClient) ExecuteOnDatabase(query string) error {
	mc.Begin()

	if _, err := mc.tx.Exec(query); mc.SkipError(err) != nil {
		mc.Logger.Error("execute DDL: %s, Query: %s.", err.Error(), query)
	}

	mc.State = StateDDL

	return mc.Commit()
}

func (mc *MysqlClient) Commit() error {
	if mc.State == StateDML || mc.State == StateDDL || mc.State == StateBEGIN {
		if mc.tx == nil {
			return fmt.Errorf("execute Commit: tx is 'nil'")
		} else {
			if err := mc.tx.Commit(); err != nil {
				mc.Logger.Error("execute Commit: %s", err.Error())
				return err
			}
			mc.State = StateCOMMIT
			mc.tx = nil
		}
	} else {
		return fmt.Errorf("execute Commit: last state is '%s'", mc.State)
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
