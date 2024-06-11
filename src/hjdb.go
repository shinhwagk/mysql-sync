package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HJDBResponseData struct {
	State *string      `json:"state"`
	Data  *interface{} `json:"data"`
	DB    *string      `json:"db"`
	Tab   *string      `json:"tab`
	Err   *string      `json:"err"`
	Store *string      `json.store`
}

type JsonRequestData struct {
	GtidSet string `json:"gtidset"`
}

type HJDB struct {
	Addr string
	DB   string

	Logger *Logger
}

type CheckpointGtidSet struct {
	server_uuid string
	xtd         int64
}

func NewHJDB(logLevel int, addr string, db string) *HJDB {
	return &HJDB{
		addr,
		db,
		NewLogger(logLevel, "hjdb"),
	}
}

func (hjdb HJDB) Update(store string, tab string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		hjdb.Logger.Error(fmt.Sprintf("Update -- %s", err))
		return err
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/db/%s/tab/%s/store/%s", hjdb.Addr, hjdb.DB, tab, store), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		hjdb.Logger.Error(fmt.Sprintf("Update -- db: '%s' tab: '%s' err: %s", hjdb.DB, tab, err.Error()))
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		hjdb.Logger.Error(fmt.Sprintf("Update -- db: '%s' tab: '%s' err: %s", hjdb.DB, tab, err))
		return err
	}

	hjdb.Logger.Debug(fmt.Sprintf("Update -- db: '%s' tab: '%s' %s %s", hjdb.DB, tab, string(jsonData), string(responseBody)))

	return nil
}

func (hjdb HJDB) query(store string, tab string) (*HJDBResponseData, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/db/%s/tab/%s/store/%s", hjdb.Addr, hjdb.DB, tab, store))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var hjdbResp HJDBResponseData
	err = json.Unmarshal(body, &hjdbResp)

	if err != nil {
		return nil, err
	}

	if *hjdbResp.State == "err" {
		return nil, fmt.Errorf("hjdb error: %s", *hjdbResp.Err)
	}

	return &hjdbResp, nil
}
