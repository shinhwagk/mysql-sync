package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

// Config 定义了 YAML 文件的结构
type Config struct {
	Debug          bool   `yaml:"debug"`
	FeatureEnabled bool   `yaml:"feature_enabled"`
	SkipErrors     string `yaml:"skip_errors"`
}

func main() {
	// 读取 YAML 文件
	data, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.Fatalf("读取配置文件出错: %v", err)
	}

	// 解析 YAML 文件
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("解析配置文件出错: %v", err)
	}

	// 打印配置
	fmt.Printf("Debug: %v\n", config.Debug)
	fmt.Printf("Feature Enabled: %v\n", config.FeatureEnabled)
	fmt.Printf("Feature Enabled: %v\n", len(config.SkipErrors))

}
