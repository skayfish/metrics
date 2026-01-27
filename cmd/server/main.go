package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// SF TODO
type MemStorage struct {
	// SF TODO
	gauge map[string]float64

	// SF TODO
	counter map[string]int64
}

// SF TODO
func (storage *MemStorage) updateGauge(name string, value float64) {
	storage.gauge[name] = value
}

// SF TODO
func (storage *MemStorage) updateCounter(name string, value int64) {
	storage.counter[name] += value
}

// SF TODO
func createHandlerUpdate(storage *MemStorage) http.HandlerFunc {
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
			if metricType != "gauge" && metricType != "counter" {
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
			case "gauge":
				metricValue, err := strconv.ParseFloat(metricValueString, 64)
				if err != nil {
					http.Error(resp, "Bad Request: metric`s value must be float64", http.StatusBadRequest)
					return
				}

				storage.updateGauge(metricName, metricValue)
			case "counter":
				metricValue, err := strconv.ParseInt(metricValueString, 10, 64)
				if err != nil {
					http.Error(resp, "Bad Request: metric`s value must be int64", http.StatusBadRequest)
					return
				}

				storage.updateCounter(metricName, metricValue)
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

		// SF TODO удалить
		fmt.Printf("URL Path: %s\n", req.URL.Path)
		fmt.Printf("Storage contains:\n%v", storage)
		fmt.Println()
	}
}

// Настраивает и запускает сервер
//
// @returns ошибку работы сервера
// SF TODO
func run(storage *MemStorage) error {
	mux := http.NewServeMux()
	mux.Handle("/update/", http.StripPrefix("/update", createHandlerUpdate(storage)))
	return http.ListenAndServe(":8080", mux)
}

// Запуск программы
func main() {
	storage := MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
	log.Fatal(run(&storage))
}
