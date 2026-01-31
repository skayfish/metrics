package handler

import (
	"fmt"
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

		// TODO: заменить на дебажное логирование
		fmt.Printf("\nDebug data:\n")
		fmt.Printf("\tURL Path: %s\n", req.URL.Path)
		fmt.Printf("\tStorage contains:\n\t%v\n\n", storage)
	}
}

// SF TODO
func CreateValueHandler(storage *storage.MemStorage) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		mType := chi.URLParam(req, "type")
		mName := chi.URLParam(req, "name")

		// TODO: заменить на дебажное логирование
		fmt.Printf("\nDebug data:\n")
		fmt.Printf("\tURL Path: %s\n", req.URL.Path)

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
