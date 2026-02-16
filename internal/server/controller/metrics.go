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
	storage           *storage.MemStorage // Хранилище метрик
	tableHTMLTemplate *template.Template  // Шаблон html таблицы метрик
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

// Ошибка во время создания контроллера метрик
const newMetricsControllerError = "controller: metrics controller creation failed"

// Создаёт новый контроллер метрик
//
//	@param storage хранилище метрик
//	@returns *MetricsController контроллер метрик, в случае успеха
//	@returns error ошибка создания, в ином случае
func NewMetricsController(storage *storage.MemStorage) (*MetricsController, error) {
	// Парсинг шаблона html
	tmpl, err := template.New("metrics-table").Parse(templateHTML)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", newMetricsControllerError, err)
	}

	return &MetricsController{storage: storage, tableHTMLTemplate: tmpl}, nil
}

// Обновляет/добавляет метрику в хранилище. Берёт данные из URL
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) UpdateFromURL(resp http.ResponseWriter, req *http.Request) {
	mType := chi.URLParam(req, "type")
	mName := chi.URLParam(req, "name")
	mValue := chi.URLParam(req, "value")

	logger.LogS.Debugw("controller: MetricsController.UpdateFromURL (before)",
		"metrics", c.storage.GetMetrics(),
	)

	switch mType {
	case model.Gauge:
		value, err := strconv.ParseFloat(mValue, 64)
		if err != nil {
			http.Error(resp, "Metric`s value must be float64", http.StatusBadRequest)
			return
		}

		if err = c.storage.UpdateGauge(mName, value); err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
	case model.Counter:
		value, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			http.Error(resp, "Metric`s value must be int64", http.StatusBadRequest)
			return
		}

		if err = c.storage.UpdateCounter(mName, value); err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(resp,
			fmt.Sprintf("Unknown metric`s type \"%s\" [counter, gauge]", mType),
			http.StatusBadRequest)
		return
	}

	logger.LogS.Debugw("controller: MetricsController.UpdateFromURL (after)",
		"metrics", c.storage.GetMetrics(),
	)
}

// Обновляет/добавляет метрику в хранилище. Берёт данные из тела в формате JSON
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) UpdateFromJSON(resp http.ResponseWriter, req *http.Request) {
	metric := model.Metrics{}
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	logger.LogS.Debugw("controller: MetricsController.UpdateFromURL (before)",
		"metrics", c.storage.GetMetrics(),
	)

	if err := c.storage.Update(metric); err != nil {
		http.Error(resp, errors.Unwrap(err).Error(), http.StatusBadRequest)
		return
	}

	logger.LogS.Debugw("controller: MetricsController.UpdateFromURL (after)",
		"metrics", c.storage.GetMetrics(),
	)
}

// Возвращает в ответе значение запрошенной метрики
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) GetValue(resp http.ResponseWriter, req *http.Request) {
	mType := chi.URLParam(req, "type")
	mName := chi.URLParam(req, "name")

	logger.LogS.Debugw("controller: MetricsController.GetValue",
		"metrics", c.storage.GetMetrics(),
	)

	var value interface{}
	var err error
	switch mType {
	case model.Gauge:
		value, err = c.storage.GetGauge(mName)
	case model.Counter:
		value, err = c.storage.GetCounter(mName)
	default:
		http.Error(resp,
			fmt.Sprintf("Unknown metric`s type \"%s\" [counter, gauge]", mType),
			http.StatusBadRequest)
		return
	}

	switch {
	case err == nil:
		fmt.Fprint(resp, value)
	case errors.Is(err, storage.ErrNotFound):
		resp.WriteHeader(http.StatusNotFound)
	case errors.Is(err, storage.ErrIncorrectCounterMetricType) || errors.Is(err, storage.ErrIncorrectGaugeMetricType):
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	default:
		logger.LogS.Errorw("controller: unknown error while accessing the storage",
			"err", err,
			"metric type", mType,
			"metric name", mName,
		)
		http.Error(resp, fmt.Sprint("Unknown error while accessing the storage: ", err), http.StatusInternalServerError)
		return
	}
}

// Ошибка обработки запроса на получение данных всех метрик
const getAllMetricsError = "controller: an error occurred while retrieving all metrics"

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
	logger.LogS.Debugw("controller: MetricsController.GetAllMetrics",
		"metrics", c.storage.GetMetrics(),
	)

	metrics := []metricHTML{}
	for id, metric := range c.storage.GetMetrics() {
		switch metric.MType {
		case model.Counter:
			metrics = append(metrics, metricHTML{ID: id, Value: *metric.Delta})
		case model.Gauge:
			metrics = append(metrics, metricHTML{ID: id, Value: *metric.Value})
		default:
			logger.LogS.Warnw("Unknown metric type", "type", metric.MType)
		}
	}

	resultTableBuf := new(bytes.Buffer)
	err := c.tableHTMLTemplate.Execute(resultTableBuf, metrics)
	if err != nil {
		logger.LogS.Errorw(getAllMetricsError,
			"error", http.StatusText(http.StatusInternalServerError),
		)
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "text/html; charset=UTF-8")
	resp.Write(resultTableBuf.Bytes())
}
