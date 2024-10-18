package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

type Cfg struct {
	MetricsPrefix      string
	GraphqlURL         string
	GraphqlAPIToken    string
	CacheExpire        int64
	QueryTimeout       int64
	FailFast           bool
	ExtendCacheOnError bool
	Queries            []Query
}

type Query struct {
	Query   string
	Metrics []Metric
}

type Metric struct {
	Description string
	Placeholder string
	Labels      []string
	Value       string
	Name        string
}

var (
	Config     *Cfg
	ConfigPath string
)

func Init(configPath string) error {
	ConfigPath = configPath
	content := []byte(`{}`)
	_, err := os.Stat(ConfigPath)
	if !os.IsNotExist(err) {
		content, err = os.ReadFile(ConfigPath)
		if err != nil {
			return err
		}
	}

	if len(content) == 0 {
		content = []byte(`{}`)
	}

	err = json.Unmarshal(content, &Config)
	if err != nil {
		return err
	}
	val, isSet := os.LookupEnv("GRAPHQLAPITOKEN")
	if isSet {
		Config.GraphqlAPIToken = val
	}

	if Config.QueryTimeout == 0 {
		Config.QueryTimeout = 60
	}

	slog.Info(fmt.Sprintf("Finished reading config from %s", configPath))
	return nil
}
