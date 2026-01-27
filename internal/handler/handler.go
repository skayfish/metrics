package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/skayfish/metrics/internal/model"
)

// Создаёт обработчик обновления метрик
//
//	@param storage хранилище метрик
//	@returns обработчик обновления метрик
func CreateHandlerUpdate(storage *model.MemStorage) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(resp, "Method of request must be POST", http.StatusNotFound)
			return
		}

		if contentType := req.Header.Get("Content-Type"); contentType != "text/plain" {
			http.Error(resp, "Content-Type must be text/plain", http.StatusNotFound)
			return
		}

		// Получаем путь и убираем начальный '/'
		path := strings.TrimPrefix(req.URL.Path, "/")
		segments := strings.Split(path, "/")

		// Проверяем, что путь соответствует шаблону /update/{metric type}/{metric name}/{metric value}
		segmentsLength := len(segments)
		switch {
		case segmentsLength == 1 || (segmentsLength == 2 && segments[1] == ""):
			metricType := segments[0]
			if metricType != model.Gauge && metricType != model.Counter {
				http.Error(resp,
					"Bad Request: unknown type of metric. Use one of this [gauge, counter]",
					http.StatusBadRequest)
				return
			}

			http.Error(resp, "Not Found: metric`s name", http.StatusNotFound)
			return
		case segmentsLength == 3:
			metricType := segments[0]
			metricName := segments[1]
			metricValueString := segments[2]

			switch metricType {
			case model.Gauge:
				metricValue, err := strconv.ParseFloat(metricValueString, 64)
				if err != nil {
					http.Error(resp, "Bad Request: metric`s value must be float64", http.StatusBadRequest)
					return
				}

				storage.UpdateGauge(metricName, metricValue)
			case model.Counter:
				metricValue, err := strconv.ParseInt(metricValueString, 10, 64)
				if err != nil {
					http.Error(resp, "Bad Request: metric`s value must be int64", http.StatusBadRequest)
					return
				}

				storage.UpdateCounter(metricName, metricValue)
			default:
				http.Error(resp,
					fmt.Sprintf("Bad Request: unknown metric`s type \"%s\" [counter, gauge]",
						metricType),
					http.StatusBadRequest)
				return
			}

		default:
			http.Error(resp,
				"Bad Request: expected /update/{metric type}/{metric name}/{metric value}",
				http.StatusBadRequest)
			return
		}

		// TODO: заменить на дебажное логирование
		fmt.Printf("\nDebug data:\n")
		fmt.Printf("\tURL Path: %s\n", req.URL.Path)
		fmt.Printf("\tStorage contains:\n\t%v\n\n", storage)
	}
}
