package main

import (
	"encoding/json"
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wogri/bbox/packages/buffer"
	"github.com/wogri/bbox/packages/scale"
	"github.com/wogri/bbox/packages/temperature"
	"github.com/wogri/bbox/packages/logger"
	"net/http"
  "log"
	"time"
)

var apiServerAddr = flag.String("api_server_addr", "https://bcloud.azure.wogri.com", "API Server Address")
var httpServerPort = flag.String("http_server_port", "8333", "HTTP server port")
var debug = flag.Bool("debug", false, "debug mode")
var prometheusActive = flag.Bool("prometheus", false, "Activate Prometheus exporter")

var (
	promTemperature = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bhive_temperature",
		Help: "Temperature of the bHive",
	},
		[]string{"BBoxID", "SensorID"},
	)
	promWeight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bhive_weight",
		Help: "Weight of the bHive",
	},
		[]string{"BBoxID"},
	)
)

var bBuffer buffer.Buffer

func temperatureHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t temperature.Temperature
	err := decoder.Decode(&t)
	if err != nil {
		logger.Error(req.RemoteAddr, err)
		return
	}
	t.Timestamp = int64(time.Now().Unix())
	bBuffer.AppendTemperature(t)
	if *debug {
		//out, _ := t.String()
		//logger.Info(string(out))
	}
	if *prometheusActive {
		promTemperature.WithLabelValues(t.BBoxID, t.SensorID).Set(t.Temperature)
	}
	//logger.Info(bBuffer)
	postClient := buffer.HttpPostClient{*apiServerAddr, "token"}
	err = bBuffer.Flush(postClient)
	if err != nil {
		//logger.Info(err)
		return
	}
}

func scaleHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var s scale.Scale
	err := decoder.Decode(&s)
	if err != nil {
		//logger.Info(err)
		return
	}
	s.Timestamp = int64(time.Now().Unix())
	if *debug {
		//out, _ := s.String()
		//logger.Info(string(out))
	}
	if *prometheusActive {
		promWeight.WithLabelValues(s.BBoxID).Set(s.Weight)
	}
}

func main() {
	flag.Parse()
	if *prometheusActive {
		prometheus.MustRegister(promTemperature)
		prometheus.MustRegister(promWeight)
	}
	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/temperature", temperatureHandler)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+*httpServerPort, nil))
}
