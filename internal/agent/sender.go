package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/skayfish/metrics/internal/encryption"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/model"
	"golang.org/x/sync/errgroup"
)

// Менеджер отправки метрик серверу
type sender struct {
	// Конфигурация работы системы
	config Config

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

// Генерирует случайное вещественное число
//
//	@returns случайное вещественное число
func generateFloat64() float64 {
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
func filtrate(metrics *runtime.MemStats) map[string]float64 {
	res := make(map[string]float64)

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

	return res
}

// SF TODO
func updateMS(ctx context.Context, interval time.Duration) <-chan runtime.MemStats {
	const prefix = "sender.update"

	out := make(chan runtime.MemStats)

	getMS := func() (ms runtime.MemStats) {
		runtime.ReadMemStats(&ms)
		return ms
	}

	go func() {
		logger.LogS.Debugf("%s: started", prefix)

		defer close(out)

		// Сразу обновляем и отправляем метрики
		out <- getMS()

		// Запуск таймера
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.LogS.Debugf("%s: terminated by context", prefix)
				return
			case <-ticker.C:
				logger.LogS.Debugf("%s: got time tick", prefix)
				out <- getMS()
			}
		}
	}()

	return out
}

// SF TODO
func collect(ctx context.Context, interval time.Duration, msCh <-chan runtime.MemStats) <-chan []model.Metrics {
	const prefix = "sender.collect"

	out := make(chan []model.Metrics)

	go func() {
		logger.LogS.Debugf("%s: started", prefix)

		defer close(out)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		msFirstTime := true
		type memStatsData struct {
			counter int64
			last    []model.Metrics
		}
		var msData memStatsData

		sendMetrics := func() {
			pc := msData.counter
			msData.last = append(msData.last, model.Metrics{
				ID:    "PollCount",
				MType: model.Counter,
				Delta: &pc,
			})

			out <- msData.last
			msData = memStatsData{}
		}

		for {
			select {
			case <-ctx.Done():
				logger.LogS.Debugf("%s: terminated by context", prefix)
				return
			case ms, ok := <-msCh:
				if !ok {
					logger.LogS.Debugf("%s: memory stats channel is closed", prefix)
					return
				}

				logger.LogS.Debugf("%s: got memory stats", prefix)

				filtered := filtrate(&ms)

				// Добавление дополнительных gauge метрик
				filtered["RandomValue"] = generateFloat64()

				// Заполнение данных ms
				msData.last = make([]model.Metrics, 0)
				for id, value := range filtered {
					msData.last = append(msData.last, model.Metrics{
						ID:    id,
						MType: model.Gauge,
						Value: &value,
					})
				}
				msData.counter++

				if msFirstTime {
					sendMetrics()
					msFirstTime = false
				}
			// SF LOGIC case ps, ok := <-psCh:
			case <-ticker.C:
				logger.LogS.Debugf("%s: got time tick", prefix)
				sendMetrics()
				msFirstTime = false
			}
		}
	}()

	return out
}

// SF TODO
func (s *sender) report(ctx context.Context, in <-chan []model.Metrics) error {
	const prefix = "sender.sender.report"

	errg := new(errgroup.Group)
	for m := range in {
		metrics := m
		errg.Go(func() error {
			logger.LogS.Debugf("%s: sending: %v", prefix, metrics)
			return s.send(ctx, metrics)
		})
	}

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}

	return nil
}

// Запускает обновление метрик и отправку их серверу
//
//	@param ctx контекст для завершения работы функции
//	@returns ошибку работы менеджера отправки метрик
func (s *sender) Run(ctx context.Context) error {
	const prefix = "agent.sender.Run"

	msChannel := updateMS(ctx, s.config.PollInterval)
	collectCh := collect(ctx, s.config.ReportInterval, msChannel)
	err := s.report(ctx, collectCh)
	if err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}

	return nil
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
//	@param ctx     контекст для завершения работы
//	@param metrics метрики для отправки
//	@returns ошибку, если возникли проблемы при отправки метрик
func (s *sender) send(ctx context.Context, metrics []model.Metrics) error {
	const prefix = "agent.sender.send"

	logger.LogS.Debugw(fmt.Sprintf("%s: send metrics", prefix), "metrics", metrics)

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("%s: failed marshal metrics: %w", prefix, err)
	}

	compressedMetricsJSON, err := compress(metricsJSON)
	if err != nil {
		return fmt.Errorf("%s: %v", prefix, err)
	}

	requestHeaders := map[string]string{
		"Content-Type":     "application/json",
		"Content-Encoding": "gzip",
	}
	if s.config.KeyEncryption != nil {
		hmac, err := encryption.SignHMAC(compressedMetricsJSON, []byte(*s.config.KeyEncryption), sha256.New)
		if err != nil {
			return fmt.Errorf("%s: %v", prefix, err)
		}

		requestHeaders["HashSHA256"] = hex.EncodeToString(hmac)
	}

	url := fmt.Sprintf("%s://%s:%d/updates", s.config.getConnectionType(), s.config.Host, s.config.Port)
	request := s.client.R().
		SetBody(compressedMetricsJSON).
		SetHeaders(requestHeaders).
		SetContext(ctx)

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
