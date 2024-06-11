package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

func main() {
	logger := NewLogger(1, "main")

	config, err := LoadConfig("/workspaces/mysqlbinlog-sync/src/config.yml")
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

func GetGtidSet(hjdb *HJDB) (string, error) {
	resp, err := hjdb.query("file", "gtidset")

	if err != nil {
		return "", err
	}

	if resp.Data == nil {
		return "", nil
	}

	if keyValueMap, ok := (*resp.Data).(map[string]interface{}); ok {
		var pairs []string
		for key, value := range keyValueMap {
			if floatValue, ok := value.(float64); ok {
				intValue := int(floatValue)
				pair := fmt.Sprintf("%s:1-%d", key, intValue)
				pairs = append(pairs, pair)
			}
		}
		return strings.Join(pairs, ","), nil
	} else {
		return "", fmt.Errorf("unexpected type for Data field: %T", *resp.Data)
	}
}

func NewDestination(name string, dc DestinationConfig, hc HJDBConfig) *Destination {
	return &Destination{
		name:   name,
		dc:     dc,
		hc:     hc,
		logger: NewLogger(dc.LogLevel, "destination"),
	}
}

type Destination struct {
	name   string
	dc     DestinationConfig
	hc     HJDBConfig
	logger *Logger
}

func (dest *Destination) Start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)
	moCh := make(chan MysqlOperation, 100)

	dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/?%s", dest.dc.User, dest.dc.Password, dest.dc.Host, dest.dc.Port, dest.dc.Params)

	hjdb := NewHJDB(dest.hc.LogLevel, dest.hc.Addr, dest.name)
	metricDirector := NewMetricDirector(dest.dc.LogLevel, hjdb, "metrics_destination")
	tcpClient := NewTCPClient(dest.dc.LogLevel, dest.dc.TCPAddr, metricCh)
	mysqlClient, err := NewMysqlClient(dest.dc.LogLevel, dns)
	if err != nil {
		dest.logger.Error("NewMysqlClient error: " + err.Error())
		cancel()
	}
	mysqlApplier := NewMysqlApplier(dest.dc.LogLevel, hjdb, mysqlClient, metricCh)

	if gtidset, err := GetGtidSet(hjdb); err != nil {
		return err
	} else {
		dest.logger.Info("Destination start from gtidset '" + gtidset + "'")

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			mysqlApplier.start(ctx, moCh)
			cancel()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			metricDirector.Start(ctx, metricCh)
			cancel()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			tcpClient.Start(ctx, moCh, gtidset)
			cancel()
		}()
		wg.Wait()
		dest.logger.Info("stopped")
	}
	return nil
}
