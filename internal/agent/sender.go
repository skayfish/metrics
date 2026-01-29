package agent

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/skayfish/metrics/internal/model"
)

// SF TODO
const URLFloatValueTemplate = "%s/update/%s/%s/%f"

// SF TODO
const URLIntegerValueTemplate = "%s/update/%s/%s/%d"

// SF TODO
const ContentTypeText = "text/plain"

// SF TODO
type sender struct {
	// SF TODO
	serverAddress string

	// SF TODO
	pollInterval time.Duration

	// SF TODO
	reportInterval time.Duration

	// SF TODO
	pollCount int64

	// SF TODO
	totalTime time.Duration

	// SF TODO
	client http.Client
}

// SF TODO
func NewSender(serverAddress string, pollInterval time.Duration, reportInterval time.Duration) sender {
	return sender{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
	}
}

// SF TODO
func (*sender) getMetrics() (res runtime.MemStats) {
	runtime.ReadMemStats(&res)
	return
}

func (*sender) generateFloat64() float64 {
	min := -math.MaxFloat32
	max := math.MaxFloat32

	source := rand.NewSource(time.Now().UnixNano())
	gen := rand.New(source)

	return min + gen.Float64()*(max-min)
}

func (this *sender) filtrate(metrics runtime.MemStats) (res map[string]float64) {
	res = make(map[string]float64)

	res["Alloc"] = float64(metrics.Alloc)
	res["BuckHashSys"] = float64(metrics.BuckHashSys)
	res["Frees"] = float64(metrics.Frees)
	res["GCCPUFraction"] = metrics.GCCPUFraction
	res["GCSys"] = float64(metrics.GCSys)
	res["HeapAlloc"] = float64(metrics.HeapAlloc)
	res["HeapIdle"] = float64(metrics.HeapIdle)
	res["HeapInuse"] = float64(metrics.HeapInuse)
	res["HeapObjects"] = float64(metrics.HeapObjects)
	res["HeapReleased"] = float64(metrics.HeapReleased)
	res["HeapSys"] = float64(metrics.HeapSys)
	res["LastGC"] = float64(metrics.LastGC)
	res["Lookups"] = float64(metrics.Lookups)
	res["MCacheInuse"] = float64(metrics.MCacheInuse)
	res["MCacheSys"] = float64(metrics.MCacheSys)
	res["MSpanInuse"] = float64(metrics.MSpanInuse)
	res["MSpanSys"] = float64(metrics.MSpanSys)
	res["Mallocs"] = float64(metrics.Mallocs)
	res["NextGC"] = float64(metrics.NextGC)
	res["NumForcedGC"] = float64(metrics.NumForcedGC)
	res["NumGC"] = float64(metrics.NumGC)
	res["OtherSys"] = float64(metrics.OtherSys)
	res["PauseTotalNs"] = float64(metrics.PauseTotalNs)
	res["StackInuse"] = float64(metrics.StackInuse)
	res["StackSys"] = float64(metrics.StackSys)
	res["Sys"] = float64(metrics.Sys)
	res["TotalAlloc"] = float64(metrics.TotalAlloc)

	res["RandomValue"] = this.generateFloat64()

	return
}

// SF TODO
func (this *sender) Run() error {
	for {
		metrics := this.getMetrics()
		this.pollCount++ // Обновление счетчика получения метрик
		if this.totalTime%this.reportInterval == 0 {
			// Фильтрация метрик, полученных из системы
			filteredMetrics := this.filtrate(metrics)
			// Добавление дополнительных gauge метрик
			filteredMetrics["RandomValue"] = this.generateFloat64()
			// Отправка метрик серверу
			err := this.send(filteredMetrics)
			if err != nil {
				return err
			}
		}

		// Ожидание следующего считывания метрик
		time.Sleep(this.pollInterval)
		this.totalTime += this.pollInterval
	}
}

// SF TODO
func (this *sender) send(gaugeMetrics map[string]float64) error {
	err := this.sendCounterMetrics()
	if err != nil {
		return err
	}

	err = this.sendGaugeMetrics(gaugeMetrics)
	if err != nil {
		return err
	}

	return nil
}

// SF TODO
func (this *sender) sendGaugeMetric(name string, value float64) error {
	url := fmt.Sprintf(URLFloatValueTemplate, this.serverAddress, model.Gauge, name, value)
	_, err := this.client.Post(url, ContentTypeText, nil)

	return err
}

// SF TODO
func (this *sender) sendGaugeMetrics(metrics map[string]float64) error {
	for name, value := range metrics {
		err := this.sendGaugeMetric(name, value)
		if err != nil {
			return err
		}
	}

	// todo писать через дебажный логгер
	fmt.Printf("Debug info:\n")
	fmt.Printf("\tGauge metrics: %v\n", metrics)

	return nil
}

// SF TODO
func (this *sender) sendCounterMetric(name string, value int64) error {
	url := fmt.Sprintf(URLIntegerValueTemplate, this.serverAddress, model.Counter, name, value)
	_, err := this.client.Post(url, ContentTypeText, nil)

	return err
}

// SF TODO
func (this *sender) sendCounterMetrics() error {
	err := this.sendCounterMetric("PollCount", this.pollCount)

	// todo писать через дебажный логгер
	fmt.Printf("Debug info:\n")
	fmt.Printf("\tCounter metrics: [%s: %d]\n", "PollCount", this.pollCount)

	return err
}
