package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HJDBResponse struct {
	State   *string         `json:"state"` //ok, err
	Data    map[string]uint `json:"data"`
	ErrMsg  *string         `json:"errmsg"`
	ErrCode *string         `json:"errcode"`
}

type HJDB struct {
	Logger *Logger
	Addr   string
}

func NewHJDB(logLevel int, addr string) *HJDB {
	return &HJDB{
		NewLogger(logLevel, "hjdb"),
		addr,
	}
}

func (hjdb *HJDB) Update(db string, sch string, tab string, data interface{}) (*HJDBResponse, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		hjdb.Logger.Error("Marshal data err: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s/file/%s/%s/%s", hjdb.Addr, db, sch, tab)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		hjdb.Logger.Error("Post url: %s: %s", url, err.Error())
		return nil, err
	}
	return hjdb.parseHJDBResponse(resp)
}

func (hjdb *HJDB) Query(db string, sch string, tab string) (*HJDBResponse, error) {
	url := fmt.Sprintf("http://%s/file/%s/%s/%s", hjdb.Addr, db, sch, tab)
	hjdb.Logger.Info("Query " + url)
	resp, err := http.Get(url)
	if err != nil {
		hjdb.Logger.Error("url '%s': %s", url, err)
		return nil, err
	}
	return hjdb.parseHJDBResponse(resp)
}

func (hjdb *HJDB) parseHJDBResponse(resp *http.Response) (*HJDBResponse, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		hjdb.Logger.Error("Read resp body err: %s", err.Error())
		return nil, err
	}

	var hjdbResp HJDBResponse
	err = json.Unmarshal(body, &hjdbResp)

	if err != nil {
		hjdb.Logger.Error("Unmarshal resp body err: %s", err.Error())
		return nil, err
	}

	return &hjdbResp, nil
}
