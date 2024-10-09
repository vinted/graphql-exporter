package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vinted/graphql-exporter/internal/config"
	"github.com/vinted/graphql-exporter/internal/graphql"
)

type Graphql struct {
	Data interface{}
}

type Metric struct {
	Name        string
	Description string
	Labels      map[string]string
	ValueName   string
	Value       string
}

type Label struct {
	Name  string
	Value string
}

type GraphqlCollector struct {
	cachedMetrics    []Metric
	cachedAt         int64
	updaterIsRunning bool
	updaterMu        sync.Mutex
	accessMu         sync.Mutex
	graphqlURL       string
}

func newGraphqlCollector() *GraphqlCollector {
	return &GraphqlCollector{
		updaterMu:  sync.Mutex{},
		accessMu:   sync.Mutex{},
		graphqlURL: config.Config.GraphqlURL,
	}
}

func buildValueData(val_hash map[string]interface{}, m string) (string, string, error) {
	var (
		metric        Metric
		error_in_hash error
	)
	for _, v := range strings.Split(m, ",") {
		if _, ok := val_hash[v]; !ok {
			error_in_hash = fmt.Errorf("missing keys in value hash: key: %s", v)
			break
		}
		if val_hash[v] == nil {
			val_hash[v] = ""
		}
		switch reflect.TypeOf(val_hash[v]).Kind() {
		case reflect.Map:
			val_hash = val_hash[v].(map[string]interface{})
		case reflect.String:
			metric.Value = val_hash[v].(string)
			metric.ValueName = v
		case reflect.Float64:
			metric.Value = fmt.Sprintf("%v", val_hash[v].(float64))
			metric.ValueName = v
		}
	}
	return metric.ValueName, metric.Value, error_in_hash
}

func buildLabelData(val interface{}, m config.Metric) (map[string]string, error) {
	var (
		metric        Metric
		label         Label
		error_in_hash error
		label_hash    map[string]interface{}
	)
	metric.Labels = make(map[string]string)
	for _, labels := range m.Labels {
		label_hash = val.(map[string]interface{})
		for _, l := range strings.Split(labels, ",") {
			if _, ok := label_hash[l]; !ok {
				error_in_hash = fmt.Errorf("Missing keys in label hash. Key: %s", l)
				break
			}
			if label_hash[l] == nil {
				label_hash[l] = ""
			}
			switch reflect.TypeOf(label_hash[l]).Kind() {
			case reflect.Map:
				label_hash = label_hash[l].(map[string]interface{})
			case reflect.String:
				label.Value = label_hash[l].(string)
				label.Name = l
			case reflect.Float64:
				label.Value = fmt.Sprintf("%v", label_hash[l].(float64))
				label.Name = l
			}
		}
		metric.Labels[label.Name] = label.Value
	}
	return metric.Labels, error_in_hash
}

func (collector *GraphqlCollector) getMetrics() ([]Metric, error) {
	var gql *Graphql
	var metrics []Metric
	for _, q := range config.Config.Queries {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(config.Config.QueryTimeout))
		result, err := graphql.GraphqlQuery(ctx, q.Query)
		cancel()
		if err != nil {
			if config.Config.FailFast {
				return nil, err
			} else {
				slog.Error(fmt.Sprintf("query error: %s", err))
				continue
			}
		}
		err = json.Unmarshal(result, &gql)
		if err != nil {
			if config.Config.FailFast {
				return nil, err
			} else {
				slog.Error(fmt.Sprintf("unmarshal error: %s", err))
				continue
			}
		}
		data := gql.Data.(map[string]interface{})
		for _, m := range q.Metrics {
			for _, val := range data[m.Placeholder].([]interface{}) {
				var metric Metric
				metric.Labels = make(map[string]string)
				metric.Description = m.Description
				var error_in_hash error
				val_hash := val.(map[string]interface{})
				// loop through value path from config. extract result
				metric.ValueName, metric.Value, error_in_hash = buildValueData(val_hash, m.Value)
				if error_in_hash != nil {
					slog.Error(fmt.Sprintf("metric value build error: %s", error_in_hash))
					continue
				}
				// loop through labels from config. Build label-value keypairs.
				metric.Labels, error_in_hash = buildLabelData(val, m)
				if error_in_hash != nil {
					slog.Error(fmt.Sprintf("metric labels build error: %s", error_in_hash))
					continue
				}
				metric.Name = config.Config.MetricsPrefix + strings.Replace(m.Value, ",", "_", -1)
				metrics = append(metrics, metric)
			}
		}
	}
	return metrics, nil
}

func (collector *GraphqlCollector) Describe(ch chan<- *prometheus.Desc) {}

func (collector *GraphqlCollector) updateMetrics() error {
	if time.Now().Unix()-collector.cachedAt > config.Config.CacheExpire {
		metrics, err := collector.getMetrics()
		collector.accessMu.Lock()
		defer collector.accessMu.Unlock()
		if err != nil {
			slog.Error(fmt.Sprintf("error collecting metrics: %s", err))
			if config.Config.ExtendCacheOnError {
				collector.cachedAt = time.Now().Unix()
			}
			return err
		} else {
			collector.cachedMetrics = metrics
			collector.cachedAt = time.Now().Unix()
		}
	}
	return nil
}

func (collector *GraphqlCollector) atomicUpdate() {
	collector.updaterMu.Lock()
	start := !collector.updaterIsRunning
	collector.updaterIsRunning = true
	collector.updaterMu.Unlock()
	if start {
		go func() {
			collector.updateMetrics()
			collector.updaterMu.Lock()
			collector.updaterIsRunning = false
			collector.updaterMu.Unlock()
		}()
	}
}

func (collector *GraphqlCollector) Collect(ch chan<- prometheus.Metric) {
	collector.atomicUpdate()

	collector.accessMu.Lock()
	defer collector.accessMu.Unlock()
	for _, metric := range collector.cachedMetrics {
		if value, err := strconv.ParseFloat(metric.Value, 64); err == nil {
			desc := prometheus.NewDesc(metric.Name, metric.Description, nil, metric.Labels)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value)
		} else {
			metric.Labels["value"] = metric.Value
			desc := prometheus.NewDesc(metric.Name, metric.Description, nil, metric.Labels)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 1)
		}
	}
}

func staticPage(w http.ResponseWriter, req *http.Request) {
	page := `<html>
    <head><title>Graphql Exporter</title></head>
    <body>
    <h1>Graphql Exporter</h1>
    <p><a href='metrics'>Metrics</a></p>
    </body>
    </html>`
	fmt.Fprintln(w, page)
}

func Start(httpListenAddress string) {
	graphql := newGraphqlCollector()
	prometheus.MustRegister(graphql)

	router := mux.NewRouter()
	router.HandleFunc("/", staticPage)
	router.Path("/metrics").Handler(promhttp.Handler())
	slog.Info("Listening on " + httpListenAddress)
	slog.Error(fmt.Sprintf("%s", http.ListenAndServe(httpListenAddress, router)))
}
