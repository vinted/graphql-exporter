package prometheus

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vinted/graphql-exporter/pkg/config"
	"github.com/vinted/graphql-exporter/pkg/graphql"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
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

var (
	metrics_cache []Metric
	cache_time    = int64(0)
	mutex         = sync.RWMutex{}
)

const metric_prepend = "graphql_exporter_"

func buildValueData(val_hash map[string]interface{}, m string) (string, string, error) {
	var (
		metric        Metric
		error_in_hash error
	)
	for _, v := range strings.Split(m, ",") {
		if _, ok := val_hash[v]; !ok {
			error_in_hash = fmt.Errorf("Missing keys in value hash: key: %s", v)
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

func getMetrics() ([]Metric, error) {
	var gql *Graphql
	var metrics []Metric
	for _, q := range config.Config.Queries {
		result, err := graphql.GraphqlQuery(q.Query)
		if err != nil {
			return nil, fmt.Errorf("Query error: %s\n", err)
		}
		err = json.Unmarshal(result, &gql)
		if err != nil {
			return nil, fmt.Errorf("Unmarshal error: %s\n", err)
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
				// loop through labels from config. Build label-value keypairs.
				metric.Labels, error_in_hash = buildLabelData(val, m)
				if error_in_hash != nil {
					log.Printf("Got error: %s", error_in_hash)
					continue
				}
				metric.Name = metric_prepend + strings.Replace(m.Value, ",", "_", -1)
				metrics = append(metrics, metric)
			}
		}
	}
	return metrics, nil
}

type graphqlCollector struct {
}

func newGraphqlCollector() *graphqlCollector {
	return &graphqlCollector{}
}

func (collector *graphqlCollector) Describe(ch chan<- *prometheus.Desc) {

}

func buildPromDesc(name string, description string, labels map[string]string) *prometheus.Desc {
	return prometheus.NewDesc(
		name,
		description,
		nil,
		labels,
	)
}

func (collector *graphqlCollector) Collect(ch chan<- prometheus.Metric) {
	var err error
	mutex.Lock()
	if time.Now().Unix()-cache_time > config.Config.CacheExpire {
		metrics_cache, err = getMetrics()
		cache_time = time.Now().Unix()
	}
	mutex.Unlock()
	if err != nil {
		log.Printf("%s", err)
	}
	for _, metric := range metrics_cache {
		var desc *prometheus.Desc
		if value, err := strconv.ParseFloat(metric.Value, 64); err == nil {
			desc = buildPromDesc(metric.Name, metric.Description, metric.Labels)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value)
		} else {
			metric.Labels["value"] = metric.Value
			desc = buildPromDesc(metric.Name, metric.Description, metric.Labels)
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
	http.Handle("/", router)
	router.Path("/metrics").Handler(promhttp.Handler())
	err := http.ListenAndServe(httpListenAddress, router)
	log.Fatal(err)
}
