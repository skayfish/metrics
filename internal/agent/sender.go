package agent

import (
	"bytes"
	"compress/gzip"
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
func (sender) getMetrics() (res runtime.MemStats) {
	runtime.ReadMemStats(&res)
	return
}

// Генерирует случайное вещественное число
//
//	@returns случайное вещественное число
func (sender) generateFloat64() float64 {
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
func (sender) filtrate(metrics runtime.MemStats) (res map[string]float64) {
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
func (s *sender) Run(ctx context.Context) error {
	const prefix = "agent.sender.Run"

	var metrics runtime.MemStats
	updateMetrics := func() {
		metrics = s.getMetrics()
		s.pollCount++
	}
	collectMetrics := func() (result []model.Metrics) {
		result = make([]model.Metrics, 0)

		// Фильтрация метрик, полученных из системы
		filteredMetrics := s.filtrate(metrics)
		// Добавление дополнительных gauge метрик
		filteredMetrics["RandomValue"] = s.generateFloat64()

		for id, value := range filteredMetrics {
			result = append(result, model.Metrics{
				ID:    id,
				MType: model.Gauge,
				Value: &value,
			})
		}

		result = append(result, model.Metrics{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &s.pollCount,
		})

		return result
	}
	reportMetrics := func() error {
		err := s.send(collectMetrics())
		if err != nil {
			return err
		}

		s.pollCount = 0

		return nil
	}

	// Сразу обновляются и отправляются метрики
	updateMetrics()
	if err := reportMetrics(); err != nil {
		return fmt.Errorf("%s: %v", prefix, err)
	}

	// Ожидание интервалов
	pollTicker := time.NewTicker(s.config.PollInterval)
	defer pollTicker.Stop()
	reportTicker := time.NewTicker(s.config.ReportInterval)
	defer reportTicker.Stop()
	for {
		if ctx != nil && ctx.Err() != nil {
			return fmt.Errorf("%s: metrics sending manager operation terminated: %w", prefix, ctx.Err())
		}

		select {
		case <-pollTicker.C:
			updateMetrics()
		case <-reportTicker.C:
			if err := reportMetrics(); err != nil {
				return fmt.Errorf("%s: %v", prefix, err)
			}
		}
	}
}

// Сжимает переданные данные с помощью gzip
//
//	@param data данные, которые нужно сжать
//	@returns []byte сжатые данные, в случае успеха
//	@returns error ошибку в ином случае
func compress(data []byte) ([]byte, error) {
	result := new(bytes.Buffer)
	compressor, err := gzip.NewWriterLevel(result, gzip.BestSpeed)
	if err != nil {
		return nil, fmt.Errorf("error creating gzip compression object: %w", err)
	}

	_, err = compressor.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed compress data %q: %w", data, err)
	}

	err = compressor.Close()
	if err != nil {
		return nil, fmt.Errorf("failed close compressor: %v", err)
	}

	return result.Bytes(), nil
}

// Отправляет метрики серверу
//
//	@param metrics метрики для отправки
//	@returns ошибку, если возникли проблемы при отправки метрик
func (s *sender) send(metrics []model.Metrics) error {
	const prefix = "agent.sender.send"

	logger.LogS.Debugw(fmt.Sprintf("%s: send metrics", prefix), "metrics", metrics)

	metricsJSON, err := json.MarshalIndent(metrics, "", "    ")
	if err != nil {
		return fmt.Errorf("%s: failed marshal metrics: %w", prefix, err)
	}

	compressedMetricsJSON, err := compress(metricsJSON)
	if err != nil {
		return fmt.Errorf("%s: %v", prefix, err)
	}

	url := fmt.Sprintf("%s://%s:%d/updates", s.config.getConnectionType(), s.config.Host, s.config.Port)
	request := s.client.R().
		SetBody(compressedMetricsJSON).
		SetHeaders(map[string]string{
			"Content-Type":     "application/json",
			"Content-Encoding": "gzip",
		})

	response, err := request.Post(url)
	if err != nil {
		return fmt.Errorf("%s: failed send metrics: %w", prefix, err)
	}

	logger.LogS.Infow(fmt.Sprintf("%s: HTTP response", prefix),
		"METHOD", "POST",
		"URL", url,
		"HEADER", response.Header(),
		"STATUS_CODE", response.StatusCode(),
		"BODY", response.String(),
	)

	return nil
}
