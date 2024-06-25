package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type MysqlSyncConfig struct {
	// Name        string              `yaml:"name"`
	Replication  ReplicationConfig   `yaml:"replication"`
	Destinations []DestinationConfig `yaml:"destination"`
	HJDB         HJDBConfig          `yaml:"hjdb"`
}

type ReplicationConfig struct {
	Name     string `yaml:"name"`
	TCPAddr  string `yaml:"tcpaddr"`
	ServerID int    `yaml:"serverid"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	LogLevel int    `yaml:"loglevel"`
}

type DestinationConfig struct {
	Name                 string `yaml:"name"`
	TCPAddr              string `yaml:"tcpaddr"`
	Host                 string `yaml:"host"`
	Port                 int    `yaml:"port"`
	User                 string `yaml:"user"`
	Password             string `yaml:"password"`
	Params               string `yaml:"params"`
	InitGtidSetsRangeStr string `yaml:"gtidsets"`
	LogLevel             int    `yaml:"loglevel"`
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
