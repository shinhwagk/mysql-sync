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
	mysqlApplier := NewMysqlApplier(dest.dc.LogLevel, hjdb, mysqlClient, metricCh)

	if gtidset, err := GetGtidSet(hjdb); err != nil {
		return err
	} else {
		dest.Logger.Info("Destination start from gtidset '" + gtidset + "'")

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
			tcpClient.Start(ctx, moCh, gtidset)
			tcpClient.Logger.Info("stopped.")
			cancel()
		}()
		wg.Wait()

		mdCancel()
		wg0.Wait()

		dest.Logger.Info("stopped")
	}
	return nil
}

func GetGtidSet(hjdb *HJDB) (map[string]uint, error) {
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
