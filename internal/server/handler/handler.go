package handler

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/storage"
)

// Создаёт обработчик обновления метрик
//
//	@param storage хранилище метрик
//	@returns обработчик обновления метрик
func CreateUpdateHandler(storage *storage.MemStorage) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
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

			storage.UpdateGauge(mName, value)
		case model.Counter:
			value, err := strconv.ParseInt(mValue, 10, 64)
			if err != nil {
				http.Error(resp, "Metric`s value must be int64", http.StatusBadRequest)
				return
			}

			storage.UpdateCounter(mName, value)
		default:
			http.Error(resp,
				fmt.Sprintf("Unknown metric`s type \"%s\" [counter, gauge]", mType),
				http.StatusBadRequest)
			return
		}

		log.Printf("\nDebug data:\n")
		log.Printf("\tURL Path: %s\n", req.URL.Path)
		log.Printf("\tStorage contains:\n\t%v\n\n", storage)
	}
}

// Создаёт обработчик получения конкретной метрики
//
//	@param storage хранилище метрик
//	@returns обработчик получения конкретной метрики
func CreateGetValueHandler(storage *storage.MemStorage) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		mType := chi.URLParam(req, "type")
		mName := chi.URLParam(req, "name")

		log.Printf("\nDebug data:\n")
		log.Printf("\tURL Path: %s\n", req.URL.Path)
		log.Printf("\tStorage contains:\n\t%v\n\n", storage)

		switch mType {
		case model.Gauge:
			if value, ok := storage.GetGauge(mName); ok {
				resp.Write([]byte(fmt.Sprint(value)))
			} else {
				resp.WriteHeader(http.StatusNotFound)
			}
		case model.Counter:
			if value, ok := storage.GetCounter(mName); ok {
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

// Ошибка обработки запроса на получение данных всех метрик
const getAllValuesError = "Error during execution of the \"get all metrics\" request handler: "

// Создаёт обработчик получения всех метрик
//
//	@param storage хранилище метрик
//	@returns обработчик получения всех метрик в случае успеха
//	@returns ошибку в ином случае
func CreateGetAllMetricsHandler(storage *storage.MemStorage) (http.HandlerFunc, error) {
	// Структура метрики для HTML таблицы
	type Metric struct {
		Name  string      // Название метрики
		Value interface{} // Значение метрики
	}

	// Парсинг шаблона html
	tmpl, err := template.New("metrics-table").Parse(templateHTML)
	if err != nil {
		return nil, errors.New(fmt.Sprint(getAllValuesError, err))
	}

	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("\nDebug data:\n")
		log.Printf("\tURL Path: %s\n", req.URL.Path)
		log.Printf("\tStorage contains:\n\t%v\n\n", storage)

		metrics := []Metric{}
		for mName, mValue := range storage.GetGauges() {
			metrics = append(metrics, Metric{Name: mName, Value: mValue})
		}

		for mName, mValue := range storage.GetCounters() {
			metrics = append(metrics, Metric{Name: mName, Value: mValue})
		}

		resultTableBuf := new(bytes.Buffer)
		err = tmpl.Execute(resultTableBuf, metrics)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			log.Println(getAllValuesError, http.StatusText(http.StatusInternalServerError))
			return
		}

		resp.Header().Set("Content-Type", "text/html; charset=UTF-8")
		resp.Write(resultTableBuf.Bytes())
	}, nil
}
