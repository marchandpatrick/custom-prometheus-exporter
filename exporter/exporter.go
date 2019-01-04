package exporter

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/marckhouzam/custom-prometheus-exporter/configparser"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector - An object to collect the metrics
type MetricsCollector struct {
	mutex         sync.RWMutex
	metricsConfig []configparser.MetricsConfig
	// TODO should support Counter not just Gauge
	gaugeVecs []*prometheus.GaugeVec
}

func getKeys(mymap map[string]string) []string {
	i := 0
	keys := make([]string, len(mymap))
	for k := range mymap {
		keys[i] = k
		i++
	}
	return keys
}

func handleRootEndpoint(name string, endpoint string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
							<head><title>` + name + `</title></head>
							<body>
							   <h1>` + name + `</h1>
							   <p>This exporter was created in YAML using the <a href=https://github.com/marckhouzam/custom-prometheus-exporter>Custom Prometheus Exporter</a></p>
							   <p><a href='` + endpoint + `'>Metrics</a></p>
							   </body>
							</html>
						  `))
	}
}

// CreateExporters instantiates each exporter as requested
// in the configuration
func CreateExporters(exportersConf []configparser.ExporterConfig) {
	for _, exporter := range exportersConf {
		metricsCollector := MetricsCollector{}
		metricsCollector.AddMetrics(exporter.Metrics)

		prometheus.MustRegister(&metricsCollector)

		// The Handler function provides a default handler to expose metrics
		// via an HTTP server.
		http.Handle(fmt.Sprintf("%s", exporter.Endpoint), promhttp.Handler())
		http.HandleFunc("/", handleRootEndpoint(exporter.Name, exporter.Endpoint))
		log.Println("Listening on port", exporter.Port, "and endpoint", exporter.Endpoint)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", exporter.Port), nil))
	}
}

// AddMetrics defines the metrics that will be provided
func (m *MetricsCollector) AddMetrics(metrics []configparser.MetricsConfig) {
	m.metricsConfig = metrics
	m.gaugeVecs = make([]*prometheus.GaugeVec, len(metrics))

	for i, metric := range m.metricsConfig {
		if metric.MetricType == "gauge" {
			m.gaugeVecs[i] = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: metric.Name,
					Help: metric.Help,
				},
				getKeys(metric.Executions[0].Labels),
			)
		} else {
			panic("Only the gauge metric is supported at the moment")
		}
	}
}

func (m *MetricsCollector) getMetrics() {
	for i, metric := range m.metricsConfig {
		for _, execution := range metric.Executions {
			output, err := exec.Command(execution.ExecutionType, "-c", execution.Command).Output()
			if err != nil {
				log.Println("Got error when running:", execution.Command+":", err)
				return
			}

			countStr := strings.TrimSpace(string(output))
			count, err := strconv.ParseFloat(countStr, 64)
			if err != nil {
				log.Printf(
					"Got error when parsing result of: "+execution.Command+
						". Expecting integer result but got %v and error "+err.Error(), countStr)
				return
			}

			// Now set the metrics
			m.gaugeVecs[i].With(execution.Labels).Set(count)
		}
	}
}

// Describe - Implements Collector.Describe
func (m *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range m.gaugeVecs {
		m.Describe(ch)
	}
}

// Collect - Implements Collector.Collect
func (m *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	m.mutex.Lock() // To protect metrics from concurrent collects.
	defer m.mutex.Unlock()

	m.getMetrics()

	for _, m := range m.gaugeVecs {
		m.Collect(ch)
	}
}
