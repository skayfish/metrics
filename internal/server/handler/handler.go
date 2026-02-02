package handler

import (
	"fmt"
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

// Начало шаблона html таблицы метрик
const htmlTableBegin = `
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
    <tbody>`

// Шаблон html строки в таблице метрик со значением вещественного типа
const htmlTableRowFloatPattern = `
        <tr>
            <td>%s</td>
            <td>%f</td>
        </tr>`

// Шаблон html строки в таблице метрик со значением целочисленного типа
const htmlTableRowIntegerPattern = `
        <tr>
            <td>%s</td>
            <td>%d</td>
        </tr>`

// Конец шаблона html таблицы метрик
const htmlTableEnd = `
    </tbody>
</table>

</body>
</html>
`

// Создаёт обработчик получения всех метрик
//
//	@param storage хранилище метрик
//	@returns обработчик получения всех метрик
func CreateGetAllValuesHandler(storage *storage.MemStorage) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("\nDebug data:\n")
		log.Printf("\tURL Path: %s\n", req.URL.Path)
		log.Printf("\tStorage contains:\n\t%v\n\n", storage)

		table := htmlTableBegin
		for mName, mValue := range storage.GetGauges() {
			table += fmt.Sprintf(htmlTableRowFloatPattern, mName, mValue)
		}

		for mName, mValue := range storage.GetCounters() {
			table += fmt.Sprintf(htmlTableRowIntegerPattern, mName, mValue)
		}
		table += htmlTableEnd

		resp.Header().Set("Content-Type", "text/html; charset=UTF-8")
		resp.Write([]byte(table))
	}
}
