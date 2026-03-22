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
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/skayfish/metrics/internal/encryption"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/model"
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

// Фильтрует необходимые runtime метрики системы
//
//	@param metrics runtime метрики системы
//	@returns отфильтрованные метрики системы
func filtrateMS(metrics *runtime.MemStats) map[string]float64 {
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

// Фильтрует необходимые метрики системы
//
//	@param ss метрики системы
//	@returns отфильтрованные метрики системы
func filtrateSS(ss *systemStat) map[string]float64 {
	res := make(map[string]float64)

	res["TotalMemory"] = float64(ss.vms.Total)
	res["FreeMemory"] = float64(ss.vms.Free)
	res["CPUutilization1"] = float64(ss.cpu)

	return res
}

// Обновляет runtime метрики системы и отправляет их по выходному каналу
//
//	@param ctx      контекст для завершения работы
//	@param interval интервал между обновлениями
//	@returns <-chan runtime.MemStats канал, в который будут отправляться runtime метрики системы
func updateMS(ctx context.Context, interval time.Duration) <-chan runtime.MemStats {
	const prefix = "sender.updateMS"

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

// Данные системы
type systemStat struct {
	vms mem.VirtualMemoryStat // Данные виртуальной памяти
	cpu int                   // Количество логических процессоров
}

// Обновляет данные системы и отправляет их по выходному каналу
//
//	@param ctx      контекст для завершения работы
//	@param interval интервал между обновлениями
//	@returns <-chan systemStat канал, в который будут отправляться данные системы
//	@returns <-chan error канал, в который будут отправляться ошибки, если возникли проблемы
func updateSS(ctx context.Context, interval time.Duration) (<-chan systemStat, <-chan error) {
	const prefix = "sender.updateSS"

	out := make(chan systemStat)
	outErr := make(chan error)

	getSS := func() error {
		vms, err := mem.VirtualMemoryWithContext(ctx)
		if err != nil {
			outErr <- err
			logger.LogS.Debugf("%s: virtual memory stat getting failed: %v", prefix, err)
			return err
		}

		cpu, err := cpu.CountsWithContext(ctx, true)
		if err != nil {
			outErr <- err
			logger.LogS.Debugf("%s: cpu counts getting failed: %v", prefix, err)
			return err
		}

		out <- systemStat{
			vms: *vms,
			cpu: cpu,
		}

		return err
	}

	go func() {
		logger.LogS.Debugf("%s: started", prefix)

		defer close(out)

		// Сразу обновляем и отправляем данные системы
		if err := getSS(); err != nil {
			return
		}

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
				if err := getSS(); err != nil {
					return
				}
			}
		}
	}()

	return out, outErr
}

// Собирает метрики и отправляет их по выходному каналу
//
//	@param ctx      контекст для завершения работы
//	@param interval интервал между отправками метрики по выходному каналу
//	@param msCh     входной канал с runtime метриками системы
//	@param ssCh     входной канал с данными системы
//	@returns <-chan []model.Metrics канал, в который будут отправляться метрики
func collect(ctx context.Context, interval time.Duration, msCh <-chan runtime.MemStats, ssCh <-chan systemStat) <-chan []model.Metrics {
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

		collectMS := func() {
			pc := msData.counter
			msData.last = append(msData.last, model.Metrics{
				ID:    "PollCount",
				MType: model.Counter,
				Delta: &pc,
			})
		}

		ssFirstTime := true
		var ssData []model.Metrics

		rawGaugesToMetrics := func(gauges map[string]float64) []model.Metrics {
			res := make([]model.Metrics, 0, len(gauges))
			for id, value := range gauges {
				res = append(res, model.Metrics{
					ID:    id,
					MType: model.Gauge,
					Value: &value,
				})
			}

			return res
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

				filtered := filtrateMS(&ms)

				// Добавление дополнительных gauge метрик
				filtered["RandomValue"] = generateFloat64()

				// Заполнение данных ms
				msData.last = rawGaugesToMetrics(filtered)
				msData.counter++

				if msFirstTime {
					collectMS()
					out <- msData.last
					msData = memStatsData{}
					msFirstTime = false
				}
			case ss, ok := <-ssCh:
				if !ok {
					logger.LogS.Debugf("%s: system stats channel is closed", prefix)
					return
				}

				logger.LogS.Debugf("%s: got system stats", prefix)

				ssData = rawGaugesToMetrics(filtrateSS(&ss))

				if ssFirstTime {
					out <- ssData
					ssData = nil
					ssFirstTime = false
				}
			case <-ticker.C:
				logger.LogS.Debugf("%s: got time tick", prefix)

				collectMS()
				msData.last = append(msData.last, ssData...)
				out <- msData.last
				msData = memStatsData{}
				ssData = nil

				msFirstTime = false
				ssFirstTime = false
			}
		}
	}()

	return out
}

// Запускает максимально возможное количество отправителей (учитывая RateLimit).
// Каждый отправитель: получает метрики из входного канала, отправляет метрики на сервер
//
//	@param ctx контекст для завершения работы
//	@param in  входной канал с метриками
//	@returns <-chan error канал, в который будут отправляться ошибки, если возникли проблемы
func (s *sender) report(ctx context.Context, in <-chan []model.Metrics) <-chan error {
	const prefix = "sender.sender.report"

	out := make(chan error)

	go func() {
		var wg sync.WaitGroup
		wg.Add(int(s.config.RateLimit))
		for i := 0; i < int(s.config.RateLimit); i++ {
			go func() {
				defer wg.Done()
				for metrics := range in {
					if err := s.send(ctx, metrics); err != nil {
						out <- fmt.Errorf("%s: %w", prefix, err)
						return
					}
				}
			}()
		}

		wg.Wait()
	}()

	return out
}

// Запускает обновление метрик и отправку их серверу
//
//	@param ctx контекст для завершения работы функции
//	@returns ошибку работы менеджера отправки метрик
func (s *sender) Run(ctx context.Context) error {
	const prefix = "agent.sender.Run"

	curCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	msCh := updateMS(curCtx, s.config.PollInterval)
	ssCh, ssErrCh := updateSS(curCtx, s.config.PollInterval)
	collectCh := collect(curCtx, s.config.ReportInterval, msCh, ssCh)
	reportErrCh := s.report(curCtx, collectCh)

	var err error
	select {
	case <-ctx.Done():
		logger.LogS.Debugf("%s: terminated by context", prefix)
		return fmt.Errorf("%s: %w", prefix, ctx.Err())
	case err = <-ssErrCh:
	case err = <-reportErrCh:
	}

	return fmt.Errorf("%s: %w", prefix, err)
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
