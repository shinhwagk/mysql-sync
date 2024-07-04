package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type MysqlSyncConfig struct {
	// Name        string              `yaml:"name"`
	Replication ReplicationConfig  `yaml:"replication"`
	Destination DestinationsConfig `yaml:"destination"`
	HJDB        HJDBConfig         `yaml:"hjdb"`
}

type ReplicationConfig struct {
	Name     string          `yaml:"name"`
	TCPAddr  string          `yaml:"tcpaddr"`
	ServerID int             `yaml:"serverid"`
	Host     string          `yaml:"host"`
	Port     int             `yaml:"port"`
	User     string          `yaml:"user"`
	Password string          `yaml:"password"`
	LogLevel int             `yaml:"loglevel"`
	Settings *SettingsConfig `yaml:"settings"`
}

type DestinationsConfig struct {
	TCPAddr      string                       `yaml:"tcpaddr"`
	Settings     *SettingsConfig              `yaml:"settings"`
	Destinations map[string]DestinationConfig `yaml:"destinations"`
}

type DestinationConfig struct {
	InitGtidSetsRangeStr string                 `yaml:"gtidsets"`
	LogLevel             int                    `yaml:"loglevel"`
	Replicate            *ReplicateConfig       `yaml:"replicate"`
	ExportPort           int                    `yaml:"export"`
	Mysql                DestinationMysqlConfig `yaml:"mysql"`
}

type DestinationMysqlConfig struct {
	Dsn        string  `yaml:"dsn"`
	SkipErrors *string `yaml:"skip_errors"`
}

type ReplicateConfig struct {
	DoDB        *string `yaml:"do_db"`
	IgnoreDB    *string `yaml:"ignore_db"`
	DoTable     *string `yaml:"do_table"`
	IgnoreTable *string `yaml:"ignore_tab"`
	// WildDoTable     *string
	// WildIgnoreTable *string
}

type SettingsConfig struct {
	CacheSize int `yaml:"cache"`
}

type HJDBConfig struct {
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
