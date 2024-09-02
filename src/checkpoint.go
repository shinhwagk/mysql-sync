package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
)

func NewCheckpoint(logLevel int, addr string, replName string, destName string) *Checkpoint {
	logger := NewLogger(logLevel, "checkpoint")

	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		logger.Error("%s.", err)
	}

	ckpt := &Checkpoint{
		Logger:                logger,
		ConsulKV:              client.KV(),
		ConsulKVPathGtidsets:  fmt.Sprintf("mysqlsync/%s/%s/gtidsets", replName, destName),
		ConsulKVPathBinlogpos: fmt.Sprintf("mysqlsync/%s/%s/binlogpos", replName, destName),

		GtidSetsMap: make(map[string]uint),
	}

	return ckpt
}

type Checkpoint struct {
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

func (ckpt *Checkpoint) InitStartupGtidSetsMap(initGtidSetsRangeStr string) error {
	hjdbGtidSetsMap, err := ckpt.QueryGtidSetsMapFromConsul()

	if err != nil {
		return err
	}

	if len(hjdbGtidSetsMap) == 0 {
		gtidSetsMap, err := GetGtidSetsMapFromGtidSetsRangeStr(initGtidSetsRangeStr)
		if err != nil {
			return err
		}
		for k, v := range gtidSetsMap {
			ckpt.GtidSetsMap[k] = v
		}
	} else {
		for k, v := range hjdbGtidSetsMap {
			ckpt.GtidSetsMap[k] = v
		}
	}
	return nil
}

func (ckpt *Checkpoint) QueryGtidSetsMapFromConsul() (map[string]uint, error) {
	p, _, err := ckpt.ConsulKV.Get(ckpt.ConsulKVPathGtidsets, nil)
	if err != nil {
		ckpt.Logger.Error("Read consul kv: %s.", err)
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
func (ckpt *Checkpoint) PersistGtidSetsMaptToConsul() error {
	if _, err := ckpt.ConsulKV.Put(&api.KVPair{Key: ckpt.ConsulKVPathGtidsets, Value: []byte(GetGtidSetsRangeStrFromGtidSetsMap(ckpt.GtidSetsMap))}, nil); err != nil {
		ckpt.Logger.Error("Write consul kv gtidsets: %s.", err)
		return err
	}
	return nil
}

func (ckpt *Checkpoint) PersistBinLogPosToConsul() error {
	if _, err := ckpt.ConsulKV.Put(&api.KVPair{Key: ckpt.ConsulKVPathBinlogpos, Value: []byte(fmt.Sprintf("%s:%d", ckpt.BinLogFile, ckpt.BinLogPos))}, nil); err != nil {
		ckpt.Logger.Error("Write consul kv binlogpos: %s.", err)
		return err
	}
	return nil
}

func (ckpt *Checkpoint) GetTrxIdOfServerUUID(serverUUID string) (uint, bool) {
	lastTrxID, ok := ckpt.GtidSetsMap[serverUUID]
	return lastTrxID, ok
}

func (ckpt *Checkpoint) SetTrxIdOfServerUUID(serverUUID string, trxID uint) {
	ckpt.GtidSetsMap[serverUUID] = trxID
}

func (ckpt *Checkpoint) SetBinlogPos(binlogfile string, binlogpos uint32) {
	ckpt.BinLogFile = binlogfile
	ckpt.BinLogPos = binlogpos
}

func GetGtidSetsRangeStrFromGtidSetsMap(gtidSetsMap map[string]uint) string {
	var keys []string
	var parts []string

	for key := range gtidSetsMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := gtidSetsMap[key]
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
