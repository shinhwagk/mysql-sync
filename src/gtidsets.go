package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
)

func NewGtidSets(logLevel int, addr string, replName string, destName string) *GtidSets {
	logger := NewLogger(logLevel, "gtidsets")

	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		logger.Error("%s.", err)
	}

	gss := &GtidSets{
		Logger:                logger,
		ConsulKV:              client.KV(),
		ConsulKVPathGtidsets:  fmt.Sprintf("mysqlsync/%s/%s/gtidsets", replName, destName),
		ConsulKVPathBinlogpos: fmt.Sprintf("mysqlsync/%s/%s/binlogpos", replName, destName),

		GtidSetsMap: make(map[string]uint),
	}

	return gss
}

type GtidSets struct {
	Logger                *Logger
	ConsulKV              *api.KV
	ConsulKVPathGtidsets  string
	ConsulKVPathBinlogpos string
	GtidSetsMap           map[string]uint
	BinLogFile            string
	BinLogPos             uint32
}

type DestStartGtidSetsRangeStr struct {
	DestName    string
	GtidSetsStr string
}

func (gss *GtidSets) InitStartupGtidSetsMap(initGtidSetsRangeStr string) error {
	hjdbGtidSetsMap, err := gss.QueryGtidSetsMapFromConsul()

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

func (gss *GtidSets) QueryGtidSetsMapFromConsul() (map[string]uint, error) {
	p, _, err := gss.ConsulKV.Get(gss.ConsulKVPathGtidsets, nil)
	if err != nil {
		gss.Logger.Error("Read consul kv: %s.", err)
		return nil, err
	}

	if p == nil {
		return make(map[string]uint), nil
	} else {
		if gsm, err := GetGtidSetsMapFromGtidSetsRangeStr(string(p.Value)); err != nil {
			return nil, err
		} else {
			return gsm, nil
		}
	}
}

// gtid sets map gssm
func (gss *GtidSets) PersistGtidSetsMaptToConsul() error {
	if _, err := gss.ConsulKV.Put(&api.KVPair{Key: gss.ConsulKVPathGtidsets, Value: []byte(GetGtidSetsRangeStrFromGtidSetsMap(gss.GtidSetsMap))}, nil); err != nil {
		gss.Logger.Error("Write consul kv gtidsets: %s.", err)
		return err
	}
	return nil
}

func (gss *GtidSets) PersistBinLogPosToConsul() error {
	if _, err := gss.ConsulKV.Put(&api.KVPair{Key: gss.ConsulKVPathBinlogpos, Value: []byte(fmt.Sprintf("%s:%d", gss.BinLogFile, gss.BinLogPos))}, nil); err != nil {
		gss.Logger.Error("Write consul kv binlogpos: %s.", err)
		return err
	}
	return nil
}

func (gss *GtidSets) GetTrxIdOfServerUUID(serverUUID string) (uint, bool) {
	lastTrxID, ok := gss.GtidSetsMap[serverUUID]
	return lastTrxID, ok
}

func (gss *GtidSets) SetTrxIdOfServerUUID(serverUUID string, trxID uint) {
	gss.GtidSetsMap[serverUUID] = trxID
}

func (gss *GtidSets) SetBinlogPos(binlogfile string, binlogpos uint32) {
	gss.BinLogFile = binlogfile
	gss.BinLogPos = binlogpos
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

func MergeGtidSetss(gsss []map[string]uint) map[string]uint {
	gssout := make(map[string]uint)
	for _, gss := range gsss {
		if len(gss) == 0 {
			return make(map[string]uint)
		}
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
