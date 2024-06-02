package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type SourceConfig struct {
	ServerID        uint32        `yaml:"serverID"`
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Charset         string        `yaml:"charset"`
	HeartbeatPeriod time.Duration `yaml:"heartbeatPeriod"`
}

type MySQLDestinationConfig struct {
	Flavor   string   `yaml:"flavor"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	User     string   `yaml:"user"`
	Password string   `yaml:"password"`
	Charset  string   `yaml:"charset"`
	Schemas  []string `yaml:"schemas"` // 注意这里的拼写可能有误，应为 "schemas"
	Tables   []string `yaml:"tables"`
}

type Config struct {
	Source       SourceConfig             `yaml:"source"`
	Destinations map[string][]interface{} `yaml:"destinations"`
}

func (c *Config) ReadConfig(path string) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Error unmarshalling YAML data: %v", err)
	}
	var cfg Config
	cfg.ReadConfig("config.yaml")
	fmt.Printf("Source Configuration: %+v\n", cfg.Source)
	for key, dests := range cfg.Destinations {
		fmt.Printf("%s:\n", key)
		for _, dest := range dests {
			switch d := dest.(type) {
			case *MySQLDestinationConfig:
				fmt.Printf("  MySQL Config: %+v\n", d)
			// case *KafkaConfig:
			// 	fmt.Printf("  Kafka Config: %+v\n", d)
			default:
				fmt.Println("  Unknown config type")
			}
		}
	}
}

// func (c *Config) ReadConfig(path string) *replication.BinlogSyncerConfig {
// 	yamlFile, err := os.ReadFile(path)
// 	if err != nil {
// 		log.Fatalf("yamlFile.Get err   #%v ", err)
// 	}
// 	err = yaml.Unmarshal(yamlFile, c)
// 	if err != nil {
// 		log.Fatalf("Unmarshal: %v", err)
// 	}

// 	return &replication.BinlogSyncerConfig{
// 		ServerID:        c.Source.ServerID,
// 		Flavor:          "mysql",
// 		Host:            c.Source.Host,
// 		Port:            c.Source.Port,
// 		User:            c.Source.User,
// 		Password:        c.Source.Password,
// 		Charset:         c.Source.Charset,
// 		HeartbeatPeriod: time.Second * c.Source.HeartbeatPeriod,
// 	}
// }
