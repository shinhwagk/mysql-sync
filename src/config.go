package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type MysqlSyncConfig struct {
	Replication ReplicationConfig  `yaml:"replication"`
	Destination DestinationsConfig `yaml:"destination"`
	Consul      ConsulConfig       `yaml:"consul"`
	LogLevel    string             `yaml:"loglevel"`
}

type ReplicationConfig struct {
	Name       string           `yaml:"name"`
	TCPAddr    string           `yaml:"tcpaddr"`
	ServerID   int              `yaml:"serverid"`
	Host       string           `yaml:"host"`
	Port       int              `yaml:"port"`
	User       string           `yaml:"user"`
	Password   string           `yaml:"password"`
	Settings   SettingsConfig   `yaml:"settings"`
	Prometheus PrometheusConfig `yaml:"prom"`
	Transfer   TransferConfig   `yaml:"transfer"`
}

type DestinationsConfig struct {
	TCPAddr      string                       `yaml:"tcpaddr"`
	CacheSize    int                          `yaml:"cache"`
	Destinations map[string]DestinationConfig `yaml:"destinations"`
}

type DestinationConfig struct {
	Mysql      DestinationMysqlConfig `yaml:"mysql"`
	Sync       DestinationSyncConfig  `yaml:"sync"`
	Prometheus PrometheusConfig       `yaml:"prom"`
}

type DestinationMysqlConfig struct {
	Dsn           string            `yaml:"dsn"`
	SessionParams map[string]string `yaml:"session_params"`
}

type DestinationSyncConfig struct {
	Replicate            DestinationReplicateConfig `yaml:"replicate"`
	InitGtidSetsRangeStr string                     `yaml:"gtidsets"`
	UpdateMode           string                     `yaml:"update_mode"`
	InsertMode           string                     `yaml:"insert_mode"`
}

type PrometheusConfig struct {
	ExportPort int `yaml:"export"`
}

type TransferConfig struct {
	MaxTime int `yaml:"max_time"`
}

type DestinationReplicateConfig struct {
	DoDB        string `yaml:"do_db"`
	IgnoreDB    string `yaml:"ignore_db"`
	DoTable     string `yaml:"do_table"`
	IgnoreTable string `yaml:"ignore_tab"`
	// WildDoTable     *string
	// WildIgnoreTable *string
}

type SettingsConfig struct {
	CacheSize int `yaml:"cache"`
}

type ConsulConfig struct {
	Addr string `yaml:"addr"`
	// DB   string `yaml:"db"`
	// LogLevel int    `yaml:"loglevel"`
}

func LoadConfig(path string) (*MysqlSyncConfig, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}
	var config MysqlSyncConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
