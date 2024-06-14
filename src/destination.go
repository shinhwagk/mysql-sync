package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

func main() {
	logger := NewLogger(1, "main")

	config, err := LoadConfig("/etc/mysqlsync/config.yml")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to LoadConfig: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	destination := NewDestination(config.Name, config.Destination, config.HJDB)
	if err := destination.Start(ctx, cancel); err != nil {
		logger.Error("main error: " + err.Error())
	}
}

func NewDestination(name string, dc DestinationConfig, hc HJDBConfig) *Destination {
	return &Destination{
		name:   name,
		dc:     dc,
		hc:     hc,
		Logger: NewLogger(dc.LogLevel, "destination"),
	}
}

type Destination struct {
	name   string
	dc     DestinationConfig
	hc     HJDBConfig
	Logger *Logger
}

func (dest *Destination) Start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)
	moCh := make(chan MysqlOperation, 1000)
	defer close(metricCh)
	defer close(moCh)

	dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/?%s", dest.dc.User, dest.dc.Password, dest.dc.Host, dest.dc.Port, dest.dc.Params)

	hjdb := NewHJDB(dest.hc.LogLevel, dest.hc.Addr, dest.name)
	metricDirector := NewMetricDirector(dest.dc.LogLevel, "destination", metricCh)
	tcpClient := NewTCPClient(dest.dc.LogLevel, dest.dc.TCPAddr, metricCh)
	mysqlClient, err := NewMysqlClient(dest.dc.LogLevel, dns)
	if err != nil {
		dest.Logger.Error("NewMysqlClient error: " + err.Error())
		cancel()
	}

	gtidSetMap, err := GetGtidSetMapFromHJDB(hjdb)
	if err != nil {
		return err
	}

	var gtidSetRangeStr = ""

	if gtidSetMap == nil {
		gtidSetRangeStr = dest.dc.GtidSet
	} else {
		gtidSetRangeStr = GetGtidSetRangeStrFromSetMap(gtidSetMap)
	}

	mysqlApplier := NewMysqlApplier(dest.dc.LogLevel, hjdb, gtidSetMap, mysqlClient, metricCh)

	dest.Logger.Info("Destination start from gtidset '" + gtidSetRangeStr + "'")

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

func GetGtidSetMapFromHJDB(hjdb *HJDB) (map[string]uint, error) {
	gtidSetMap := make(map[string]uint)
	resp, err := hjdb.query("gtidset")

	if err != nil {
		return nil, err
	}

	if resp.Data != nil {
		if keyValueMap, ok := (*resp.Data).(map[string]interface{}); ok {
			for key, value := range keyValueMap {
				if floatValue, ok := value.(float64); ok {
					if _, ok := gtidSetMap[key]; ok {
						gtidSetMap[key] = uint(floatValue)
					}
				}
			}
		} else {
			return nil, fmt.Errorf("unexpected type for Data field: %T", *resp.Data)
		}
	}
	return nil, nil
}

func GetGtidSetRangeStrFromSetMap(gtidSetMap map[string]uint) string {
	var parts []string
	for key, value := range gtidSetMap {
		part := fmt.Sprintf("%s:1-%d", key, value)
		parts = append(parts, part)
	}
	return strings.Join(parts, ",")
}
