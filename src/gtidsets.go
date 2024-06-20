package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type HJDBResponse struct {
	State   *string          `json:"state"`
	Data    *map[string]uint `json:"data"`
	ErrMsg  *string          `json:"errmsg"`
	ErrCode *string          `json:"errcode"`
}

func NewGtidSets(logLevel int, addr string, database string, initGtidSetsRangeStr string) (*GtidSets, error) {
	gss := &GtidSets{
		Logger:       NewLogger(logLevel, "gtidsets"),
		HJDBAddr:     addr,
		HJDBDatabase: database,
		// InitGtidSets: initGtidSetsRangeStr,
		GtidSetsMap:      make(map[string]uint),
		GtidSetsRangeStr: "",
	}

	hjdbGtidSetsMap, err := gss.QueryGtidSetsMapFromHJDB()
	if err != nil {
		return nil, err
	}

	if len(hjdbGtidSetsMap) == 0 {
		gtidSetsMap, err := GetGtidSetsMapFromGtidSetsRangeStr(initGtidSetsRangeStr)
		if err != nil {
			return nil, err
		}
		for k, v := range gtidSetsMap {
			gss.GtidSetsMap[k] = v
		}
		gss.GtidSetsRangeStr = initGtidSetsRangeStr
	} else {
		for k, v := range hjdbGtidSetsMap {
			gss.GtidSetsMap[k] = v
		}
		gss.GtidSetsRangeStr = GetGtidSetsRangeStrFromGtidSetsMap(gss.GtidSetsMap)
	}

	return gss, nil
}

type GtidSets struct {
	Logger *Logger

	HJDBAddr     string
	HJDBDatabase string

	GtidSetsMap      map[string]uint
	GtidSetsRangeStr string
}

func (g *GtidSets) QueryGtidSetsMapFromHJDB() (map[string]uint, error) {
	url := fmt.Sprintf("http://%s/file/%s/gtidsets", g.HJDBAddr, g.HJDBDatabase)
	g.Logger.Info("Query " + url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var hjdbResp HJDBResponse
	if err := json.Unmarshal(body, &hjdbResp); err != nil {
		g.Logger.Error(fmt.Sprintf("json unmarshal err: %s", err.Error()))
		return nil, err
	} else {
		if *hjdbResp.State == "err" {
			fmt.Println(*hjdbResp.ErrCode)
			if hjdbResp.ErrCode != nil && (*hjdbResp.ErrCode == "HJDB-001" || *hjdbResp.ErrCode == "HJDB-002") {
				g.Logger.Error(fmt.Sprintf("hjdb-err: %s", *hjdbResp.ErrMsg))
				return make(map[string]uint), nil
			}
			return nil, fmt.Errorf(*hjdbResp.ErrMsg)
		} else {
			g.Logger.Info(fmt.Sprintf("Query gtidsets from hjdb: %v", *hjdbResp.Data))
			return *hjdbResp.Data, nil
			// g.Logger.Debug(fmt.Sprintf("Persist gtidsets map '%v' success.", gssm))
		}
	}
}

// gtid sets map gssm
func (ggs *GtidSets) PersistGtidSetsMaptToHJDB() error {
	jsonData, err := json.Marshal(ggs.GtidSetsMap)
	if err != nil {
		ggs.Logger.Error(fmt.Sprintf("json marshal err: %s", err))
	}

	url := fmt.Sprintf("http://%s/file/%s/gtidsets", ggs.HJDBAddr, ggs.HJDBDatabase)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		ggs.Logger.Error(fmt.Sprintf("request '%s' err: %s", url, err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ggs.Logger.Error(fmt.Sprintf("read body err: %s", err.Error()))
	}

	var hjdbResp HJDBResponse
	if err := json.Unmarshal(body, &hjdbResp); err != nil {
		ggs.Logger.Error(fmt.Sprintf("json unmarshal err: %s", err.Error()))
		return err
	} else {
		if *hjdbResp.State == "err" {
			ggs.Logger.Error(fmt.Sprintf("hjdb resp err: %s", *hjdbResp.ErrMsg))
			return fmt.Errorf(*hjdbResp.ErrMsg + "\n")
		} else {
			ggs.Logger.Debug(fmt.Sprintf("Persist gtidsets map '%v' complate.", ggs.GtidSetsMap))
		}
	}
	return nil
}

func (g *GtidSets) GetTrxIdOfServerUUID(serverUUID string) (uint, bool) {
	lastTrxID, ok := g.GtidSetsMap[serverUUID]
	return lastTrxID, ok
}

func (g *GtidSets) SetTrxIdOfServerUUID(serverUUID string, trxID uint) error {
	if lastTrxID, ok := g.GetTrxIdOfServerUUID(serverUUID); ok {
		if lastTrxID+1 == trxID {
			g.GtidSetsMap[serverUUID] = trxID
		} else {
			return fmt.Errorf("gtid trxid order error: uuid:'%s' last:'%d', next '%d'", serverUUID, lastTrxID, trxID)
		}
	} else {
		g.Logger.Warning(fmt.Sprintf("Gtid: '%s:%d' first join.", serverUUID, trxID))
		g.GtidSetsMap[serverUUID] = trxID
	}
	return nil
}

func GetGtidSetsRangeStrFromGtidSetsMap(gtidSetsMap map[string]uint) string {
	var parts []string
	for key, value := range gtidSetsMap {
		part := fmt.Sprintf("%s:1-%d", key, value)
		parts = append(parts, part)
	}
	return strings.Join(parts, ",")
}

func GetGtidSetsMapFromGtidSetsRangeStr(gtidSetsRangeStr string) (map[string]uint, error) {
	result := make(map[string]uint)
	if gtidSetsRangeStr == "" {
		return result, nil
	}
	parts := strings.Split(gtidSetsRangeStr, ",")

	for _, part := range parts {
		pair := strings.Split(part, ":")
		if len(pair) != 2 {
			return nil, fmt.Errorf("invalid format, expected uuid:number-number but got %s", part)
		}
		uuid := pair[0]
		rangeParts := strings.Split(pair[1], "-")
		if len(rangeParts) != 2 {
			return nil, fmt.Errorf("invalid range format, expected number-number but got %s", pair[1])
		}
		end, err := strconv.ParseUint(rangeParts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number in range: %s", rangeParts[1])
		}
		result[uuid] = uint(end)
	}
	return result, nil
}

// func main() {
// 	gss, err := NewGtidSets(1, "hjdb:8000", "test1", "aaa:1-2")

// 	if err != nil {
// 		fmt.Println(err.Error())
// 	}

// 	fmt.Println(gss.GtidSetsRangeStr)
// 	fmt.Println(gss.GtidSetsMap)

// 	gss.SetTrxIdOfServerUUID("aaa", 3)
// 	gss.PersistGtidSetsMaptToHJDB()
// 	fmt.Println(gss.GetTrxIdOfServerUUID("aaa"))
// }
