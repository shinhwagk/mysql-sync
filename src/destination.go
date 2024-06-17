package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

func NewDestination(name string, dc DestinationConfig) *Destination {
	return &Destination{
		name:   name,
		dc:     dc,
		Logger: NewLogger(dc.LogLevel, "destination"),
	}
}

type Destination struct {
	name   string
	dc     DestinationConfig
	Logger *Logger
}

func (dest *Destination) Start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)
	moCh := make(chan MysqlOperation)
	defer close(metricCh)
	defer close(moCh)

	dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/?%s", dest.dc.User, dest.dc.Password, dest.dc.Host, dest.dc.Port, dest.dc.Params)

	hjdb := NewHJDB(dest.dc.LogLevel, dest.dc.HJDB.Addr, dest.dc.HJDB.DB)
	metricDirector := NewMetricDirector(dest.dc.LogLevel, "destination", metricCh)
	tcpClient := NewTCPClient(dest.dc.LogLevel, dest.dc.TCPAddr, metricCh)
	mysqlClient, err := NewMysqlClient(dest.dc.LogLevel, dns)
	if err != nil {
		dest.Logger.Error("NewMysqlClient error: " + err.Error())
		cancel()
	}
	defer mysqlClient.Close()

	gtidSetsMap, err := GetGtidSetsMapFromHJDB(hjdb)
	if err != nil {
		return err
	}

	var gtidSetRangeStr = ""

	if len(gtidSetsMap) == 0 {
		gtidSetRangeStr = dest.dc.GtidSets
		gsm, err := GetGtidSetsMapFromGtidSetsRangeStr(gtidSetRangeStr)
		if err != nil {
			return err
		}
		for key, value := range gsm {
			gtidSetsMap[key] = value
		}
	} else {
		gtidSetRangeStr = GetGtidSetsRangeStrFromSetMap(gtidSetsMap)
	}

	mysqlApplier := NewMysqlApplier(dest.dc.LogLevel, hjdb, gtidSetsMap, mysqlClient, metricCh)

	dest.Logger.Info("Destination start from gtidsets '" + gtidSetRangeStr + "'")

	mdCtx, mdCancel := context.WithCancel(context.Background())
	defer mdCancel()

	var wg0 sync.WaitGroup
	wg0.Add(1)
	go func() {
		defer wg0.Done()
		metricDirector.Logger.Info("started.")
		metricDirector.Start(mdCtx, "0.0.0.0:9092")
		metricDirector.Logger.Info("stopped.")
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		mysqlApplier.Logger.Info("started.")
		mysqlApplier.Start(ctx, moCh)
		mysqlApplier.Logger.Info("stopped.")
		cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tcpClient.Logger.Info("started.")
		tcpClient.Start(ctx, moCh, gtidSetRangeStr)
		tcpClient.Logger.Info("stopped.")
		cancel()
	}()
	wg.Wait()

	mdCancel()
	wg0.Wait()

	dest.Logger.Info("stopped")

	return nil
}

func GetGtidSetsMapFromHJDB(hjdb *HJDB) (map[string]uint, error) {
	gtidSetMap := make(map[string]uint)
	resp, err := hjdb.query("gtidsets")

	if err != nil {
		return nil, err
	}

	if resp.ErrMsg != nil {
		if resp.ErrCode != nil && *resp.ErrCode == "hjdb-001" {
			fmt.Println("hjdb error: " + *resp.ErrMsg)
			return gtidSetMap, nil
		}
		return nil, fmt.Errorf(*resp.ErrMsg)
	}

	if resp.Data != nil {
		if keyValueMap, ok := (*resp.Data).(map[string]interface{}); ok {
			for key, value := range keyValueMap {
				if floatValue, ok := value.(float64); ok {
					gtidSetMap[key] = uint(floatValue)
					fmt.Println(key, floatValue)
				}
			}
		} else {
			return nil, fmt.Errorf("unexpected type for Data field: %T", *resp.Data)
		}
		return gtidSetMap, nil
	}
	return gtidSetMap, nil
}

func GetGtidSetsRangeStrFromSetMap(gtidSetMap map[string]uint) string {
	var parts []string
	for key, value := range gtidSetMap {
		part := fmt.Sprintf("%s:1-%d", key, value)
		parts = append(parts, part)
	}
	return strings.Join(parts, ",")
}

func GetGtidSetsMapFromGtidSetsRangeStr(gtidSetsStr string) (map[string]uint, error) {
	result := make(map[string]uint)
	parts := strings.Split(gtidSetsStr, ",")
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
