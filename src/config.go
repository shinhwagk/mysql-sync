package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Replication ReplicationConfig `yaml:"replication"`
	Destination DestinationConfig `yaml:"destination"`
	HJDB        HJDBConfig        `yaml:"hjdb"`
}

type ReplicationConfig struct {
	ServerID int    `yaml:"serverid"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	GTID     string `yaml:"gtid"`
}

type DestinationConfig struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Params   string `yaml:"params"`
}

type HJDBConfig struct {
	Addr string `yaml:"addr"`
	DB   string `yaml:"db"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
