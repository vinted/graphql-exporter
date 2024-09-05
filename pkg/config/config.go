package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/template"
	"time"
)

type Cfg struct {
	GraphqlURL      string
	GraphqlAPIToken string
	CacheExpire     int64
	Queries         []Query
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
}

var (
	Config     *Cfg
	ConfigPath string
)

var funcMap = template.FuncMap{
	"NOW": func(t string) (string, error) {
		d, err := time.ParseDuration(t)
		return time.Now().UTC().Add(d).Format(time.RFC3339), err
	},
}

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

	if err != nil {
		return fmt.Errorf("%s", err)
	}
	tpl, err := template.New("query").Funcs(funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	var templateBuffer bytes.Buffer
	err = tpl.Execute(&templateBuffer, nil)
	if err != nil {
		return fmt.Errorf("Template error %s", err)
	}

	err = json.Unmarshal(templateBuffer.Bytes(), &Config)
	if err != nil {
		return err
	}
	val, isSet := os.LookupEnv("GRAPHQLAPITOKEN")
	if isSet {
		Config.GraphqlAPIToken = val
	}

	log.Printf("Finished reading config.")
	return nil
}
