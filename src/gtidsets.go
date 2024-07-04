package main

import (
	"fmt"
	"strconv"
	"strings"
)

func NewGtidSets(addr string, replName string, destName string) *GtidSets {
	gss := &GtidSets{
		Logger:      NewLogger(0, "gtidsets"),
		HJDB:        NewHJDB(1, addr),
		GtidSetsMap: make(map[string]uint),

		ReplName: replName,
		DestName: destName,

		HJDBDB:  "mysqlsync_" + replName,
		HJDBSCH: "dest_" + destName,
		HJDBTAB: "gtidsets",
	}

	return gss
}

type GtidSets struct {
	Logger *Logger

	HJDB *HJDB

	GtidSetsMap map[string]uint

	ReplName string
	DestName string

	HJDBDB  string
	HJDBSCH string
	HJDBTAB string
}

func (gss *GtidSets) InitStartupGtidSetsMap(initGtidSetsRangeStr string) error {
	hjdbGtidSetsMap, err := gss.QueryGtidSetsMapFromHJDB(gss.ReplName, gss.DestName)

	if err != nil {
		return err
	}

	if len(hjdbGtidSetsMap) == 0 {
		gtidSetsMap, err := GetGtidSetsMapFromGtidSetsRangeStr(initGtidSetsRangeStr)
		if err != nil {
			return err
		}
		for k, v := range gtidSetsMap {
			gss.GtidSetsMap[k] = v
		}
	} else {
		for k, v := range hjdbGtidSetsMap {
			gss.GtidSetsMap[k] = v
		}
	}
	return nil
}

func (gss *GtidSets) QueryGtidSetsMapFromHJDB(replName string, destName string) (map[string]uint, error) {
	db := "mysqlsync_" + gss.ReplName
	sch := "dest_" + gss.DestName
	tab := "gtidsets"
	hjdbResp, err := gss.HJDB.Query(db, sch, tab)
	if err != nil {
		return nil, err
	}

	if *hjdbResp.State == "err" {
		fmt.Println(*hjdbResp.ErrCode)
		if hjdbResp.ErrCode != nil && (*hjdbResp.ErrCode == "HJDB-001" || *hjdbResp.ErrCode == "HJDB-002" || *hjdbResp.ErrCode == "HJDB-005") {
			gss.Logger.Warning("hjdb-err: %s", *hjdbResp.ErrMsg)
			return make(map[string]uint), nil
		}
		return nil, fmt.Errorf(*hjdbResp.ErrMsg)
	} else {
		gss.Logger.Info(fmt.Sprintf("Query gtidsets from hjdb: %v", *hjdbResp.Data))
		return *hjdbResp.Data, nil
		// g.Logger.Debug(fmt.Sprintf("Persist gtidsets map '%v' success.", gssm))
	}

}

// gtid sets map gssm
func (gss *GtidSets) PersistGtidSetsMaptToHJDB() error {
	db := "mysqlsync_" + gss.ReplName
	sch := "dest_" + gss.DestName
	tab := "gtidsets"

	hjdbResp, err := gss.HJDB.Update(db, sch, tab, gss.GtidSetsMap)
	if err != nil {
		return err
	}

	if *hjdbResp.State == "err" {
		gss.Logger.Error(fmt.Sprintf("hjdb resp err: %s", *hjdbResp.ErrMsg))
		return fmt.Errorf(*hjdbResp.ErrMsg + "\n")
	} else {
		gss.Logger.Debug(fmt.Sprintf("Persist gtidsets map '%v' complate.", gss.GtidSetsMap))
	}

	return nil
}

func (gss *GtidSets) GetTrxIdOfServerUUID(serverUUID string) (uint, bool) {
	lastTrxID, ok := gss.GtidSetsMap[serverUUID]
	return lastTrxID, ok
}

func (gss *GtidSets) SetTrxIdOfServerUUID(serverUUID string, trxID uint) error {
	if lastTrxID, ok := gss.GetTrxIdOfServerUUID(serverUUID); ok {
		if lastTrxID+1 == trxID {
			gss.GtidSetsMap[serverUUID] = trxID
		} else {
			return fmt.Errorf("gtid trxid order uuid:'%s' last:'%d', next '%d'", serverUUID, lastTrxID, trxID)
		}
	} else {
		gss.Logger.Warning(fmt.Sprintf("Gtid: '%s:%d' first join.", serverUUID, trxID))
		gss.GtidSetsMap[serverUUID] = trxID
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

func MergeGtidSets(gsss []map[string]uint) map[string]uint {
	gssout := make(map[string]uint)
	for _, gss := range gsss {
		for serverUUID, trxID := range gss {
			if currentTrx, ok := gssout[serverUUID]; ok {
				if trxID < currentTrx {
					gssout[serverUUID] = trxID
				}
			} else {
				gssout[serverUUID] = trxID
			}
		}
	}
	return gssout
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

// func QueryGtidSetsMapFromHJDB(hjdbAddr string, replName string, destName string) (map[string]uint, error) {
// 	url := fmt.Sprintf("http://%s/file/mysqlsync_%s/dest_%s/gtidsets", hjdbAddr, replName, destName)
// 	gss.Logger.Info("Query " + url)

// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var hjdbResp HJDBResponse
// 	if err := json.Unmarshal(body, &hjdbResp); err != nil {
// 		gss.Logger.Error(fmt.Sprintf("json unmarshal err: %s", err.Error()))
// 		return nil, err
// 	} else {
// 		if *hjdbResp.State == "err" {
// 			fmt.Println(*hjdbResp.ErrCode)
// 			if hjdbResp.ErrCode != nil && (*hjdbResp.ErrCode == "HJDB-001" || *hjdbResp.ErrCode == "HJDB-002" || *hjdbResp.ErrCode == "HJDB-005") {
// 				gss.Logger.Error(fmt.Sprintf("hjdb-err: %s", *hjdbResp.ErrMsg))
// 				return make(map[string]uint), nil
// 			}
// 			return nil, fmt.Errorf(*hjdbResp.ErrMsg)
// 		} else {
// 			gss.Logger.Info(fmt.Sprintf("Query gtidsets from hjdb: %v", *hjdbResp.Data))
// 			return *hjdbResp.Data, nil
// 			// g.Logger.Debug(fmt.Sprintf("Persist gtidsets map '%v' success.", gssm))
// 		}
// 	}
// }
