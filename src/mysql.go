package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

type MysqlClient struct {
	db            *sql.DB
	tx            *sql.Tx
	Logger        *Logger
	SkipErrors    []uint16
	SessionParams map[string]string
}

func NewMysqlClient(logLevel int, dmc DestinationMysqlConfig) (*MysqlClient, error) {
	Logger := NewLogger(logLevel, "mysql-client")

	Logger.Info("dsn: %s", dmc.Dsn)

	db, err := sql.Open("mysql", dmc.Dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 10)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(2)

	pSkipErrors := ""

	sessionParams := map[string]string{}
	for pk, pv := range dmc.SessionParams {
		if pk == "replica_skip_errors" {
			pSkipErrors = pv
		} else {
			sessionParams[pk] = pv
		}
	}

	skipErrors, err := ConvertStringToUint16Slice(pSkipErrors)
	if err != nil {
		return nil, err
	}

	myclient := &MysqlClient{
		db:            db,
		tx:            nil,
		Logger:        Logger,
		SkipErrors:    skipErrors,
		SessionParams: sessionParams,
	}

	return myclient, nil
}

func (mc *MysqlClient) SkipError(err error) error {
	if merr, ok := err.(*mysql.MySQLError); ok {
		for _, v := range mc.SkipErrors {
			if v == merr.Number {
				mc.Logger.Error("Skip error: %s.", err)
				return nil
			}
		}
	}
	return err
}

func (mc *MysqlClient) Close() error {
	if err := mc.db.Close(); err != nil {
		mc.Logger.Error("Connection close: %s.", err)
		return err
	}
	return nil
}

func (mc *MysqlClient) Begin() error {
	var err error
	if mc.tx == nil {
		if mc.tx, err = mc.db.Begin(); err != nil {
			mc.Logger.Error("execute Begin: %s.", err)
			return err
		} else {
			for pk, pv := range mc.SessionParams {
				query := fmt.Sprintf("SET SESSION %s = '%s'", pk, pv)
				mc.Logger.Debug("execute Begin: init session parameter '%s'", query)
				if _, err := mc.tx.Exec(query); err != nil {
					mc.Logger.Error("execute Begin: init session parameter '%s' %s", query, err)
					return err
				}
			}
		}
	} else {
		err := fmt.Errorf("execute Begin: tx is not nil")
		mc.Logger.Error("%s.", err)
		return err
	}

	return nil
}

func (mc *MysqlClient) ExecuteDML(query string, args []interface{}) error {
	if mc.tx != nil {
		if _, err := mc.tx.Exec(query, args...); err != nil {
			if serr := mc.SkipError(err); serr != nil {
				mc.Logger.Error("execute DML: %s, Query: %s, Params: %#v.", serr, query, args)
				return err
			} else {
				mc.Logger.Warning("skip error: %s, Query: %s, Params: %#v.", err, query, args)
			}
		}
	} else {
		err := fmt.Errorf("execute DML: tx is not nil")
		mc.Logger.Error("%s.", err)
		return err
	}

	return nil
}

func (mc *MysqlClient) ExecuteOnTable(db string, query string) error {
	if err := mc.Begin(); err != nil {
		return err
	}

	if db != "" {
		if _, err := mc.tx.Exec("USE " + db); err != nil {
			mc.Logger.Error("execute DDL: %s, Database: %s Query: %s.", err, db, query)
			return err
		}
	}

	if _, err := mc.tx.Exec(query); err != nil {
		if serr := mc.SkipError(err); serr != nil {
			mc.Logger.Error("execute DDL: %s, Database: %s Query: %s.", serr, db, query)
			return err
		} else {
			mc.Logger.Warning("skip error: %s, Database: %s Query: %s.", err, db, query)
		}
		return err
	}

	if err := mc.Commit(); err != nil {
		return err
	}

	return nil
}

func (mc *MysqlClient) ExecuteOnDatabase(query string) error {
	if err := mc.Begin(); err != nil {
		return err
	}

	if _, err := mc.tx.Exec(query); err != nil {
		if serr := mc.SkipError(err); serr != nil {
			mc.Logger.Error("execute DDL: %s, Query: %s.", serr, query)
			return err
		} else {
			mc.Logger.Warning("skip error: %s, Query: %s.", err, query)
		}
	}

	if err := mc.Commit(); err != nil {
		return err
	}

	return nil
}

func (mc *MysqlClient) Commit() error {
	if mc.tx == nil {
		err := fmt.Errorf("execute Commit: tx is 'nil'")
		mc.Logger.Error("%s.", err)
		return err
	} else {
		if err := mc.tx.Commit(); err != nil {
			mc.Logger.Error("execute Commit: %s.", err)
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
