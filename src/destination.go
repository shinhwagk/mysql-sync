package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func main() {
	logger := NewLogger(1, "main")

	config, err := LoadConfig("/workspaces/mysqlbinlog-sync/src/config.yml")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to LoadConfig: %v", err))
	}

	fmt.Println(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	destination, err := NewDestination(config.Name, config.Destination, config.HJDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to NewDestination: %v", err))
	}

	destination.Start(ctx, cancel)
}

type Destination struct {
	tcpClient      *TCPClient
	mysqlApply     *MysqlApply
	metricDirector *MetricDirector
	hjdb           *HJDB

	cancel context.CancelFunc

	logger *Logger
}

func NewDestination(name string, dc DestinationConfig, hc HJDBConfig) (*Destination, error) {
	ctx, _ := context.WithCancel(context.Background())

	hjdb := NewHJDB(hc.LogLevel, hc.Addr, hc.DB)
	dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/?%s", dc.User, dc.Password, dc.Host, dc.Port, dc.Params)
	mysqlClient := NewMysqlClient(dc.LogLevel, dns)

	tcpClient := NewTCPClient(dc.LogLevel, dc.TCPAddr)
	metricDirector := NewMetricDirector(dc.LogLevel, hjdb)

	go func() {
		<-ctx.Done()
		mysqlClient.Close()
		tcpClient.Close()
	}()

	return &Destination{
		tcpClient:      tcpClient,
		mysqlApply:     NewMysqlApply(dc.LogLevel, mysqlClient, hjdb),
		metricDirector: metricDirector,
		hjdb:           hjdb,
		logger:         NewLogger(dc.LogLevel, "destination"),
	}, nil
}

func (dest *Destination) Start(ctx context.Context, cancel context.CancelFunc) {
	gtidset, err := dest.getGtidSet()
	if err != nil {
		dest.logger.Error("gtidset parse " + err.Error())
		return
	}

	moCh := make(chan MysqlOperation, 100)

	fmt.Println(gtidset)
	go func() {
		// for oper := range moCh {
		// 	fmt.Println(oper)
		// }
		dest.mysqlApply.start(moCh)
		cancel()
	}()

	dest.tcpClient.Start(ctx, moCh, gtidset)
	dest.logger.Info("stopped")
}

func (dest Destination) getGtidSet() (string, error) {
	resp, err := dest.hjdb.query("gtidset")
	if err != nil {
		return "", err
	}

	fmt.Println(resp.Data, reflect.TypeOf(resp.Data))
	if keyValueMap, ok := (resp.Data).(map[string]interface{}); ok {
		fmt.Println("Converted successfully:", keyValueMap)

		var pairs []string

		for key, value := range keyValueMap {
			pairs = append(pairs, fmt.Sprintf("%s:%s", key, value))
		}

		return strings.Join(pairs, ","), nil
	} else {
		return "", fmt.Errorf("unexpected type for Data field: %T", resp.Data)

	}
}
