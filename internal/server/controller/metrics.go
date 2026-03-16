package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/skayfish/metrics/internal/logger"
	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/storage"
)

// Контроллер обработки запросов, связанных с метриками
type MetricsController struct {
	storage           storage.Storage    // Хранилище метрик
	tableHTMLTemplate *template.Template // Шаблон html таблицы метрик
}

// HTML шаблон таблицы метрик
const templateHTML = `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Таблица метрик</title>
    <style>
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 8px;
            text-align: left;
        }
        .float-value { color: blue; }
        .int-value { color: green; }
    </style>
</head>
<body>

<h2>Таблица метрик</h2>

<table>
    <thead>
        <tr>
            <th>Название</th>
            <th>Значение</th>
        </tr>
    </thead>
    <tbody>
        {{range .}}
        <tr>
            <td>
                {{if eq (printf "%T" .Value) "float64"}}
                    <span class="float-value">{{.ID}}</span>
                {{else if eq (printf "%T" .Value) "int64"}}
                    <span class="int-value">{{.ID}}</span>
                {{else}}
                    <span class="unknown">{{.ID}}</span>
                {{end}}</td>
            <td>
                {{if eq (printf "%T" .Value) "float64"}}
                    <span class="float-value">{{printf "%.2f" .Value}}</span>
                {{else if eq (printf "%T" .Value) "int64"}}
                    <span class="int-value">{{printf "%d" .Value}}</span>
                {{else}}
                    <span class="unknown">Неизвестный тип</span>
                {{end}}
            </td>
        </tr>
        {{end}}
    </tbody>
</table>

</body>
</html>
`

// Создаёт новый контроллер обработки запросов, связанных с метриками
//
//	@param storage хранилище данных
//	@returns *MetricsController контроллер метрик, в случае успеха
//	@returns error ошибка создания, в ином случае
func NewMetricsController(storage storage.Storage) (*MetricsController, error) {
	tmpl, err := template.New("metrics-table").Parse(templateHTML)
	if err != nil {
		return nil, fmt.Errorf("controller.NewMetricsController: creation failed: %v", err)
	}

	return &MetricsController{storage: storage, tableHTMLTemplate: tmpl}, nil
}

// Обновляет/добавляет метрику в хранилище. Берёт данные из URL
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) UpdateFromURL(resp http.ResponseWriter, req *http.Request) {
	const prefix = "controller.MetricsController.UpdateFromURL"

	mType := chi.URLParam(req, "type")
	mName := chi.URLParam(req, "name")
	mValue := chi.URLParam(req, "value")

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(fmt.Sprintf("%s: before", prefix), "metrics", metrics)
	}

	var metric model.Metrics
	switch mType {
	case model.Gauge:
		value, err := strconv.ParseFloat(mValue, 64)
		if err != nil {
			http.Error(resp, "Metric`s value must be float64", http.StatusBadRequest)
			return
		}

		metric = model.Metrics{
			ID:    mName,
			MType: model.Gauge,
			Value: &value,
		}
	case model.Counter:
		delta, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			http.Error(resp, "Metric`s value must be int64", http.StatusBadRequest)
			return
		}

		metric = model.Metrics{
			ID:    mName,
			MType: model.Counter,
			Delta: &delta,
		}
	default:
		http.Error(resp,
			fmt.Sprintf("Unknown metric`s type \"%s\" [counter, gauge]", mType),
			http.StatusBadRequest)
		return
	}

	updatedMetric, err := c.storage.UpdateContext(req.Context(), metric)
	if err != nil {
		if errors.Is(err, storage.ErrFoundNotCounterMetricType) || errors.Is(err, storage.ErrFoundNotGaugeMetricType) {
			http.Error(resp, errors.Unwrap(err).Error(), http.StatusBadRequest)
		} else {
			logger.LogS.Errorf("%s: %v", prefix, err)
			resp.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	updatedMetricJSON, err := json.Marshal(updatedMetric)
	if err != nil {
		logger.LogS.Errorf("%s: marshaling metric failed: %v", prefix, err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write(updatedMetricJSON)

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(fmt.Sprintf("%s: after", prefix), "metrics", metrics)
	}
}

// Обновляет/добавляет метрику в хранилище. Берёт данные из тела в формате JSON
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) UpdateFromJSON(resp http.ResponseWriter, req *http.Request) {
	const prefix = "controller.MetricsController.UpdateFromJSON"

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(fmt.Sprintf("%s: before", prefix), "metrics", metrics)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(resp, "Expected application/json content type", http.StatusBadRequest)
		return
	}

	metric := model.Metrics{}
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		http.Error(resp, fmt.Sprintf("Failed unmarshall json: %v", err), http.StatusBadRequest)
		return
	}

	if err := metric.Valid(); err != nil {
		http.Error(resp, fmt.Sprintf("%v: id=%q", err, metric.ID), http.StatusBadRequest)
		return
	}

	updatedMetric, err := c.storage.UpdateContext(req.Context(), metric)
	if err != nil {
		if errors.Is(err, storage.ErrFoundNotCounterMetricType) || errors.Is(err, storage.ErrFoundNotGaugeMetricType) {
			http.Error(resp, errors.Unwrap(err).Error(), http.StatusBadRequest)
		} else {
			logger.LogS.Errorf("%s: %v", prefix, err)
			resp.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(fmt.Sprintf("%s: after", prefix),
			"metrics", metrics,
			"updated metric", updatedMetric,
		)
	}

	updatedMetricJSON, err := json.Marshal(updatedMetric)
	if err != nil {
		logger.LogS.Errorf("%s: marshaling metric failed: %v", prefix, err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write(updatedMetricJSON)
}

// Обновляет/добавляет метрики в хранилище. Берёт данные из тела в формате JSON
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) Updates(resp http.ResponseWriter, req *http.Request) {
	const prefix = "controller.MetricsController.Updates"

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(fmt.Sprintf("%s: before", prefix), "metrics", metrics)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(resp, "Expected application/json content type", http.StatusBadRequest)
		return
	}

	metrics := []model.Metrics{}
	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		http.Error(resp, fmt.Sprintf("Failed unmarshall json: %v", err), http.StatusBadRequest)
		return
	}

	validationErrors := []error{}
	for _, metric := range metrics {
		if err := metric.Valid(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("%v: id=%q", err, metric.ID))
		}
	}

	if len(validationErrors) > 0 {
		errMessage := ""
		for _, err := range validationErrors {
			errMessage += fmt.Sprintln(err)
		}

		http.Error(resp, errMessage, http.StatusBadRequest)
		return
	}

	updatedMetrics, err := c.storage.UpdatesContext(req.Context(), metrics)
	if err != nil {
		if errors.Is(err, storage.ErrFoundNotGaugeMetricType) || errors.Is(err, storage.ErrFoundNotCounterMetricType) {
			http.Error(resp, errors.Unwrap(err).Error(), http.StatusBadRequest)
			return
		}

		logger.LogS.Errorf("%s: %v", prefix, err)
		http.Error(resp, "failed update metrics", http.StatusInternalServerError)
		return
	}

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(fmt.Sprintf("%s: after", prefix),
			"metrics", metrics,
		)
	}

	updatedMetricsJSON, err := json.Marshal(updatedMetrics)
	if err != nil {
		logger.LogS.Errorf("%s: marshaling metrics failed: %v", prefix, err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write(updatedMetricsJSON)
}

// Возвращает значение запрошенной метрики.
// Данные для поиска в хранилище берутся из запроса в URL
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) GetValueFromURL(resp http.ResponseWriter, req *http.Request) {
	const prefix = "controller.MetricsController.GetValueFromURL"

	mType := chi.URLParam(req, "type")
	mName := chi.URLParam(req, "name")

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(prefix, "metrics", metrics)
	}

	var metric *model.Metrics
	var err error
	switch mType {
	case model.Gauge:
		metric, err = c.storage.GetContext(req.Context(), mName)
		if err == nil {
			if metric.Value != nil {
				fmt.Fprint(resp, *metric.Value)
			} else {
				http.Error(resp, `found not "gauge" metric type`, http.StatusBadRequest)
			}

			return
		}
	case model.Counter:
		metric, err = c.storage.GetContext(req.Context(), mName)
		if err == nil {
			if metric.Delta != nil {
				fmt.Fprint(resp, *metric.Delta)
			} else {
				http.Error(resp, `found not "counter" metric type`, http.StatusBadRequest)
			}

			return
		}
	default:
		http.Error(resp,
			fmt.Sprintf("Unknown metric`s type \"%s\" [counter, gauge]", mType),
			http.StatusBadRequest)
		return
	}

	switch {
	case errors.Is(err, storage.ErrMetricNotFound):
		http.Error(resp, fmt.Sprintf("Metric with id %q, type %q not found", mName, mType),
			http.StatusNotFound)
		return
	case errors.Is(err, storage.ErrFoundNotGaugeMetricType) || errors.Is(err, storage.ErrFoundNotCounterMetricType):
		http.Error(resp, errors.Unwrap(err).Error(), http.StatusBadRequest)
		return
	default:
		logger.LogS.Errorw(fmt.Sprintf("%s: unknown error while accessing the storage: %v", prefix, err),
			"metric type", mType,
			"metric name", mName,
		)
		http.Error(resp, "Unknown error while accessing the storage", http.StatusInternalServerError)
		return
	}
}

// Возвращает данные запрошенной метрики в формате JSON.
// Данные для поиска в хранилище берутся из запроса в формате JSON
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) GetMetricFromJSON(resp http.ResponseWriter, req *http.Request) {
	const prefix = "controller.MetricsController.GetMetricFromJSON"

	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(resp, "Expected application/json content type", http.StatusBadRequest)
		return
	}

	metric := model.Metrics{}
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(prefix, "metrics", metrics)
	}

	var storageMetric *model.Metrics
	var err error
	switch metric.MType {
	case model.Gauge:
		storageMetric, err = c.storage.GetContext(req.Context(), metric.ID)
		if err == nil && storageMetric.Value == nil {
			http.Error(resp, `found not "gauge" metric type`, http.StatusBadRequest)
			return
		}
	case model.Counter:
		storageMetric, err = c.storage.GetContext(req.Context(), metric.ID)
		if err == nil && storageMetric.Delta == nil {
			http.Error(resp, `found not "counter" metric type`, http.StatusBadRequest)
			return
		}
	default:
		http.Error(resp,
			fmt.Sprintf("Unknown metric`s type \"%s\" [counter, gauge]", metric.MType),
			http.StatusBadRequest)
		return
	}

	// Проверка ошибки
	switch {
	case err == nil:
		break
	case errors.Is(err, storage.ErrMetricNotFound):
		http.Error(resp, fmt.Sprintf("Metric with id %q, type %q not found", metric.ID, metric.MType),
			http.StatusNotFound)
		return
	case errors.Is(err, storage.ErrFoundNotGaugeMetricType) || errors.Is(err, storage.ErrFoundNotCounterMetricType):
		http.Error(resp, errors.Unwrap(err).Error(), http.StatusBadRequest)
		return
	default:
		logger.LogS.Errorw(fmt.Sprintf("%s: unknown error while accessing the storage", prefix),
			"err", err,
			"metric type", metric.MType,
			"metric name", metric.ID,
		)
		http.Error(resp, "Unknown error while accessing the storage: ", http.StatusInternalServerError)
		return
	}

	// Формирование ответа
	metricJSON, err := json.MarshalIndent(*storageMetric, "", "    ")
	if err != nil {
		logger.LogS.Errorf("%s: failed marshal metric: %v", prefix, err)
		http.Error(resp, "Failed marshal metric", http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write(metricJSON)
}

// Структура метрики для HTML таблицы
type metricHTML struct {
	ID    string      // Идентификатор метрики
	Value interface{} // Значение метрики
}

// Возвращает в ответе html таблицу со всеми метриками и их значениями
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) GetAllMetrics(resp http.ResponseWriter, req *http.Request) {
	const prefix = "controller.MetricsController.GetAllMetrics"

	if logger.IsDebug() {
		metrics, err := c.storage.GetAllContext(req.Context())
		if err != nil {
			logger.LogS.Errorf("%s: %v", prefix, err)
			http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
			return
		}

		logger.LogS.Debugw(prefix, "metrics", metrics)
	}

	metricsHTML := []metricHTML{}
	metrics, err := c.storage.GetAllContext(req.Context())
	if err != nil {
		logger.LogS.Errorf("%s: %v", prefix, err)
		http.Error(resp, "failed get all metrics", http.StatusInternalServerError)
		return
	}

	for _, metric := range metrics {
		switch metric.MType {
		case model.Counter:
			metricsHTML = append(metricsHTML, metricHTML{ID: metric.ID, Value: *metric.Delta})
		case model.Gauge:
			metricsHTML = append(metricsHTML, metricHTML{ID: metric.ID, Value: *metric.Value})
		default:
			logger.LogS.Warnw(fmt.Sprintf("%s: unknown metric type", prefix), "type", metric.MType)
		}
	}

	resultTableBuf := new(bytes.Buffer)
	err = c.tableHTMLTemplate.Execute(resultTableBuf, metricsHTML)
	if err != nil {
		logger.LogS.Errorf("%s: failed create html table for metrics: %v", prefix, err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "text/html; charset=UTF-8")
	resp.Write(resultTableBuf.Bytes())
}
