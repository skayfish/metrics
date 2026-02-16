package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/model"
)

// Менеджер отправки метрик серверу
type sender struct {
	// Конфигурация работы системы
	config Config

	// Количество обновлений метрик за время работы программы
	pollCount int64

	// Клиент для отправки запросов серверу
	client *resty.Client
}

// Создаёт новый менеджер отправки метрик серверу
//
//	@param config конфигурация работы агента
//	@returns новый менеджер отправки метрик серверу
func NewSender(config Config) sender {
	client := resty.New()
	client.
		SetRetryCount(5).
		SetRetryWaitTime(config.RetryWaitTime).
		SetRetryMaxWaitTime(config.RetryMaxWaitTime)

	logger.LogS.Infow("Client launch successful")

	return sender{config: config, client: client}
}

// Получает метрики из системы
//
//	@returns метрики из системы
func (*sender) getMetrics() (res runtime.MemStats) {
	runtime.ReadMemStats(&res)
	return
}

// Генерирует случайное вещественное число
//
//	@returns случайное вещественное число
func (*sender) generateFloat64() float64 {
	min := -math.MaxFloat32
	max := math.MaxFloat32

	source := rand.NewSource(time.Now().UnixNano())
	gen := rand.New(source)

	return min + gen.Float64()*(max-min)
}

// Фильтрует необходимые метрики системы
//
//	@param metrics метрики системы
//	@returns отфильтрованные метрики системы
func (obj *sender) filtrate(metrics runtime.MemStats) (res map[string]float64) {
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

	return
}

// Запускает обновление метрик и отправку их серверу
//
//	@param ctx контекст для завершения работы функции
//	@returns ошибку работы менеджера отправки метрик
func (obj *sender) Run(ctx context.Context) error {
	var metrics runtime.MemStats

	updateMetrics := func() {
		metrics = obj.getMetrics()
		obj.pollCount++
	}
	reportMetrics := func() error {
		// Фильтрация метрик, полученных из системы
		filteredMetrics := obj.filtrate(metrics)
		// Добавление дополнительных gauge метрик
		filteredMetrics["RandomValue"] = obj.generateFloat64()
		// Отправка метрик серверу
		err := obj.send(filteredMetrics)
		if err != nil {
			return err
		}

		obj.pollCount = 0

		return nil
	}

	// Сразу обновляются и отправляются метрики
	updateMetrics()
	if err := reportMetrics(); err != nil {
		return err
	}

	// Ожидание интервалов
	pollTicker := time.NewTicker(obj.config.PollInterval)
	reportTicker := time.NewTicker(obj.config.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()
	for {
		if ctx != nil && ctx.Err() != nil {
			return fmt.Errorf("metrics sending manager operation terminated: %w", ctx.Err())
		}

		select {
		case <-pollTicker.C:
			updateMetrics()
		case <-reportTicker.C:
			if err := reportMetrics(); err != nil {
				return err
			}
		}
	}
}

// Отправляет метрики серверу
//
//	@param gaugeMetrics метрики датчиков
//	@returns ошибку отправки метрик серверу
func (obj *sender) send(gaugeMetrics map[string]float64) error {
	err := obj.sendCounterMetrics()
	if err != nil {
		return err
	}

	err = obj.sendGaugeMetrics(gaugeMetrics)
	if err != nil {
		return err
	}

	return nil
}

// Отправляет метрику датчика на сервер
//
//	@param name  имя метрики
//	@param value значение метрики
//	@returns ошибку отправки метрики датчика на сервер
func (obj *sender) sendGaugeMetric(name string, value float64) error {
	metric := model.Metrics{
		ID:    name,
		MType: model.Gauge,
		Value: &value,
	}

	metricJSON, err := json.MarshalIndent(metric, "", "    ")
	if err != nil {
		return fmt.Errorf("agent: sender.sendGaugeMetric: failed to marshal gauge metric %q with value %f: %w", name, value, err)
	}

	url := fmt.Sprintf("%s://%s:%d/update", obj.config.getConnectionType(), obj.config.Host, obj.config.Port)
	_, err = obj.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metricJSON).
		Post(url)
	if err != nil {
		return fmt.Errorf("agent: sender.sendGaugeMetric: failed to send gauge metric %q with value %f: %w", name, value, err)
	}

	return nil
}

// Отправляет метрики датчиков на сервер
//
//	@param metrics метрики датчиков
//	@returns ошибку отправки метрик датчиков на сервер
func (obj *sender) sendGaugeMetrics(metrics map[string]float64) error {
	for name, value := range metrics {
		if err := obj.sendGaugeMetric(name, value); err != nil {
			return err
		}
	}

	logger.LogS.Debugw("agent: sender.sendGaugeMetrics: data sent successfully",
		"gauge metrics", metrics,
	)

	return nil
}

// Отправляет метрику счетчика на сервер
//
//	@param name  имя метрики
//	@param value значение метрики
//	@returns ошибку отправления метрики счетчика на сервер
func (obj *sender) sendCounterMetric(name string, value int64) error {
	metric := model.Metrics{
		ID:    name,
		MType: model.Counter,
		Delta: &value,
	}

	metricJSON, err := json.MarshalIndent(metric, "", "    ")
	if err != nil {
		return fmt.Errorf("agent: sender.sendCounterMetric: failed to marshal counter metric %q with value %d: %w", name, value, err)
	}

	url := fmt.Sprintf("%s://%s:%d/update", obj.config.getConnectionType(), obj.config.Host, obj.config.Port)
	_, err = obj.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metricJSON).
		Post(url)
	if err != nil {
		return fmt.Errorf("agent: sender.sendCounterMetric: failed to send counter metric %q with value %d: %w", name, value, err)
	}

	return nil
}

// Отправляет метрики счетчиков на сервер
//
//	@returns ошибку отправления метрик счетчиков на сервер
func (obj *sender) sendCounterMetrics() error {
	if err := obj.sendCounterMetric("PollCount", obj.pollCount); err != nil {
		return err
	}

	logger.LogS.Debugw("agent: sender.sendGaugeMetrics: data sent successfully",
		"counter metrics", map[string]interface{}{"PollCount": obj.pollCount},
	)

	return nil
}
