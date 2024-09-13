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
				return nil
			}
		}
	}
	return err
}

func (mc *MysqlClient) Close() error {
	if err := mc.db.Close(); err != nil {
		mc.Logger.Error("Connection close: %s", err)
		return err
	}
	return nil
}

func (mc *MysqlClient) Begin() error {
	var err error

	if mc.tx != nil {
		err := fmt.Errorf("tx already exists")
		mc.Logger.Error("Execute[begin] -- Error: %s", err)
		return err
	}

	if mc.tx, err = mc.db.Begin(); err != nil {
		mc.Logger.Error("Execute[begin] -- Error: %s", err)
		return err
	}

	for pk, pv := range mc.SessionParams {
		query := fmt.Sprintf("SET SESSION %s = '%s'", pk, pv)
		if _, err := mc.tx.Exec(query); err != nil {
			mc.Logger.Error("Execute[begin] -- set session parameter '%s', Error: %s", query, err)
			return err
		}
		mc.Logger.Debug("Execute[begin] -- set session parameter: '%s'", query)
	}

	return nil
}

func (mc *MysqlClient) ExecuteOnDML(query string, args []interface{}) error {
	if mc.tx == nil {
		err := fmt.Errorf("tx does not exist")
		mc.Logger.Error("Execute[dml] -- Error: %s", err)
		return err
	}

	if _, err := mc.tx.Exec(query, args...); err != nil {
		if serr := mc.SkipError(err); serr != nil {
			mc.Logger.Error("Execute[dml] -- Query: %s, Params: %#v, Error: %s", query, args, err)

			if err := mc.Rollback(); err != nil {
				return err
			}
			return err
		}

		mc.Logger.Warning("Execute[dml] -- Query: %s, Params: %#v, Skip Error: %s", query, args, err)
	}

	return nil
}

func (mc *MysqlClient) ExecuteOnNonDML(schemaContext string, query string) error {
	if err := mc.Begin(); err != nil {
		return err
	}

	if schemaContext != "" {
		if _, err := mc.tx.Exec("USE " + schemaContext); err != nil {
			mc.Logger.Error("Execute[nondml] -- SchemaContext: %s, Query: %s, Error: %s", schemaContext, query, err)

			if err := mc.Rollback(); err != nil {
				return err
			}
			return err
		}
	}

	if _, err := mc.tx.Exec(query); err != nil {
		if serr := mc.SkipError(err); serr != nil {
			mc.Logger.Error("Execute[nondml] -- SchemaContext: %s, Query: %s, Error: %s", schemaContext, query, err)

			if err := mc.Rollback(); err != nil {
				return err
			}

			return err
		}

		mc.Logger.Warning("Execute[nondml] -- SchemaContext: %s, Query: %s, Skip Error: %s", schemaContext, query, err)
	}

	return mc.Commit()
}

func (mc *MysqlClient) Commit() error {
	if mc.tx == nil {
		err := fmt.Errorf("tx does not exist")
		mc.Logger.Error("Execute[commit] -- Error: %s", err)
		return err
	}

	if err := mc.tx.Commit(); err != nil {
		mc.Logger.Error("Execute[commit] -- Error: %s", err)
		return err
	}
	mc.tx = nil

	mc.Logger.Debug("Execute[commit] -- complate")
	return nil
}

func (mc *MysqlClient) Rollback() error {
	if mc.tx == nil {
		mc.Logger.Warning("Execute[rollback] -- tx does not exist")
		return nil
	}

	if err := mc.tx.Rollback(); err != nil {
		mc.Logger.Error("Execute[rollback] -- Error: %s", err)
		return err
	}
	mc.tx = nil

	mc.Logger.Debug("Execute[rollback] -- complate")
	return nil
}
