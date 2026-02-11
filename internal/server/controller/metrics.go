package controller

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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
                    <span class="float-value">{{.Name}}</span>
                {{else if eq (printf "%T" .Value) "int64"}}
                    <span class="int-value">{{.Name}}</span>
                {{else}}
                    <span class="unknown">{{.Name}}</span>
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

// Обновляет/добавляет метрику в хранилище
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) Update(resp http.ResponseWriter, req *http.Request) {
	mType := chi.URLParam(req, "type")
	mName := chi.URLParam(req, "name")
	mValue := chi.URLParam(req, "value")

	switch mType {
	case model.Gauge:
		value, err := strconv.ParseFloat(mValue, 64)
		if err != nil {
			http.Error(resp, "Metric`s value must be float64", http.StatusBadRequest)
			return
		}

		c.storage.UpdateGauge(mName, value)
	case model.Counter:
		value, err := strconv.ParseInt(mValue, 10, 64)
		if err != nil {
			http.Error(resp, "Metric`s value must be int64", http.StatusBadRequest)
			return
		}

		c.storage.UpdateCounter(mName, value)
	default:
		http.Error(resp,
			fmt.Sprintf("Unknown metric`s type \"%s\" [counter, gauge]", mType),
			http.StatusBadRequest)
		return
	}

	log.Printf("\nDebug data:\n")
	log.Printf("\tURL Path: %s\n", req.URL.Path)
	log.Printf("\tStorage contains:\n\t%v\n\n", c.storage)
}

// Возвращает в ответе значение запрошенной метрики
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) GetValue(resp http.ResponseWriter, req *http.Request) {
	mType := chi.URLParam(req, "type")
	mName := chi.URLParam(req, "name")

	log.Printf("\nDebug data:\n")
	log.Printf("\tURL Path: %s\n", req.URL.Path)
	log.Printf("\tStorage contains:\n\t%v\n\n", c.storage)

	switch mType {
	case model.Gauge:
		if value, ok := c.storage.GetGauge(mName); ok {
			resp.Write([]byte(fmt.Sprint(value)))
		} else {
			resp.WriteHeader(http.StatusNotFound)
		}
	case model.Counter:
		if value, ok := c.storage.GetCounter(mName); ok {
			resp.Write([]byte(fmt.Sprint(value)))
		} else {
			resp.WriteHeader(http.StatusNotFound)
		}
	default:
		http.Error(resp,
			fmt.Sprintf("Unknown metric`s type \"%s\" [counter, gauge]", mType),
			http.StatusBadRequest)
		return
	}
}

// Ошибка обработки запроса на получение данных всех метрик
const getAllMetricsError = "controller: an error occurred while retrieving all metrics"

// Структура метрики для HTML таблицы
type metric struct {
	Name  string      // Название метрики
	Value interface{} // Значение метрики
}

// Возвращает в ответе html таблицу со всеми метриками и их значениями
//
//	@param resp объект для записи ответа
//	@param req  объект запроса
func (c *MetricsController) GetAllMetrics(resp http.ResponseWriter, req *http.Request) {
	log.Printf("\nDebug data:\n")
	log.Printf("\tURL Path: %s\n", req.URL.Path)
	log.Printf("\tStorage contains:\n\t%v\n\n", c.storage)

	metrics := []metric{}
	for mName, mValue := range c.storage.GetGauges() {
		metrics = append(metrics, metric{Name: mName, Value: mValue})
	}

	for mName, mValue := range c.storage.GetCounters() {
		metrics = append(metrics, metric{Name: mName, Value: mValue})
	}

	resultTableBuf := new(bytes.Buffer)
	err := c.tableHTMLTemplate.Execute(resultTableBuf, metrics)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		log.Printf("%s: %s", getAllMetricsError, http.StatusText(http.StatusInternalServerError))
	}

	resp.Header().Set("Content-Type", "text/html; charset=UTF-8")
	resp.Write(resultTableBuf.Bytes())
}
