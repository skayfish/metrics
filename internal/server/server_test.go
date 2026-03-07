package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/skayfish/metrics/internal/flags"
	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Возвращает пустую middleware-обёртку
func getEmptyMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(resp http.ResponseWriter, req *http.Request) {}
	}
}

// Проверяет настройку маршрутизатора запросов
func Test_getRouter(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := storage.NewMemStorage()
		_, err := getRouter(&s, getEmptyMiddleware())
		assert.NoError(t, err)
	})
	t.Run("success", func(t *testing.T) {
		_, err := getRouter(nil, getEmptyMiddleware())
		assert.NoError(t, err)
	})
}

// Возвращает непустое хранилище с валидными данными
func newSuccessMemStorage() storage.MemStorage {
	var delta1 int64 = 1298476200
	var value1 float64 = 131072
	var value2 float64 = 15288

	return storage.MemStorage{
		"GetSet92": model.Metrics{
			ID:    "GetSet92",
			MType: model.Counter,
			Delta: &delta1,
		},
		"StackInuse": model.Metrics{
			ID:    "StackInuse",
			MType: model.Gauge,
			Value: &value1,
		},
		"MCacheSys": model.Metrics{
			ID:    "MCacheSys",
			MType: model.Gauge,
			Value: &value2,
		},
	}
}

// Проверяет создание хранилища метрик из json файла
func Test_createStorageFromJSON(t *testing.T) {
	tests := []struct {
		test        string
		filePath    string
		want        storage.MemStorage
		wantErr     bool
		errorPrefix string
	}{
		{
			test:        "file does not exist",
			filePath:    "./testdata/unknown.json",
			want:        storage.MemStorage{},
			wantErr:     true,
			errorPrefix: fmt.Sprintf("failed read from file %q:", "./testdata/unknown.json"),
		},
		{
			test:        "failed unmarshal",
			filePath:    "./testdata/errorJSON.json",
			want:        storage.MemStorage{},
			wantErr:     true,
			errorPrefix: fmt.Sprintf("failed unmarshal metrics from file %q:", "./testdata/errorJSON.json"),
		},
		{
			test:     "success",
			filePath: "./testdata/success.json",
			want:     newSuccessMemStorage(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			metrics, err := createStorageFromJSON(tt.filePath)
			if tt.wantErr {
				require.True(t, strings.HasPrefix(err.Error(), tt.errorPrefix))
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, *metrics)
			}
		})
	}
}

// Проверяет создание сервера по переданной конфигурации
func TestNewServer(t *testing.T) {
	tests := []struct {
		test   string
		config Config
		want   storage.MemStorage
	}{
		{
			test: "restore and empty file storage path",
			config: Config{
				FileStoragePath: "",
				ToRestore:       true,
			},
			want: storage.NewMemStorage(),
		},
		{
			test: "failed restore",
			config: Config{
				FileStoragePath: "./testdata/errorJSON.json",
				ToRestore:       true,
			},
			want: storage.NewMemStorage(),
		},
		{
			test: "success restore",
			config: Config{
				FileStoragePath: "./testdata/success.json",
				ToRestore:       true,
			},
			want: newSuccessMemStorage(),
		},
		{
			test: "no restore",
			config: Config{
				FileStoragePath: "./testdata/success.json",
				ToRestore:       false,
			},
			want: storage.NewMemStorage(),
		},
		{
			test: "no restore",
			config: Config{
				FileStoragePath: "./testdata/errorJSON.json",
				ToRestore:       false,
			},
			want: storage.NewMemStorage(),
		},
		{
			test: "no restore",
			config: Config{
				FileStoragePath: "",
				ToRestore:       false,
			},
			want: storage.NewMemStorage(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			s, err := NewServer(&tt.config)
			require.NoError(t, err)
			assert.Equal(t, tt.config, *s.config)
			ms, ok := s.storage.(*storage.MemStorage)
			require.True(t, ok)
			assert.Equal(t, tt.want, *ms)
		})
	}
}

// Возвращает свободный порт на устройстве
func getFreePort() (int, error) {
	// Слушаем на порту :0 — система выделит любой свободный
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}

	// Получаем реальный выделенный порт
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Сразу закрываем слушатель — порт освободится
	listener.Close()

	return port, nil
}

// Создаёт корректный сервер с переданной конфигурацией
func createServer(t *testing.T, restore bool, fileStoragePath string, storeInterval time.Duration) *Server {
	port, err := getFreePort()
	require.NoError(t, err)

	s, err := NewServer(&Config{
		Address:         flags.NetAddress{Host: "localhost", Port: port},
		FileStoragePath: fileStoragePath,
		ToRestore:       restore,
		StoreInterval:   storeInterval,
	})
	require.NoError(t, err)

	return s
}

// Запускает сервер в отдельной горутине
func listenServer(s *Server) chan error {
	errChan := make(chan error)
	go func() {
		err := s.Listen(context.Background())
		errChan <- err
	}()
	return errChan
}

// Проверяет запуск сервера с различными настройками сохранения данных хранилища метрик
func TestServer_Listen(t *testing.T) {
	t.Run("success launch", func(t *testing.T) {
		s := createServer(t, false, "", 0)

		// Инициализация роутера
		handler := func(resp http.ResponseWriter, req *http.Request) {
			resp.WriteHeader(http.StatusOK)
			resp.Write([]byte("OK"))
		}
		router := chi.Router(chi.NewRouter())
		router.Post("/", handler)
		s.router = &router

		// Запуск сервера
		errChan := listenServer(s)
		defer close(errChan)
		time.Sleep(100 * time.Millisecond)

		// Делаем запрос к серверу
		client := resty.New()
		resp, err := client.R().Post(strings.Join([]string{"http:/", s.config.Address.String()}, "/"))
		require.NoError(t, err)

		assert.Equal(t, "OK", resp.String())
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		// Завершаем сервер
		select {
		case err := <-errChan:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			fmt.Println("Server forced shutdown")
		}
	})

	t.Run("sync save storage", func(t *testing.T) {
		notEmptyStorage := newSuccessMemStorage()

		s := createServer(t, true, "", 0)
		s.storage = &notEmptyStorage

		// Инициализация роутера
		handlerNoSave := func(resp http.ResponseWriter, req *http.Request) {}
		handlerSave := func(resp http.ResponseWriter, req *http.Request) {
			s.saveStorageChan <- struct{}{}
		}

		router := chi.Router(chi.NewRouter())
		router.Post("/nosave", handlerNoSave)
		router.Post("/save", handlerSave)
		s.router = &router

		// Запуск сервера
		errChan := listenServer(s)
		defer close(errChan)
		time.Sleep(100 * time.Millisecond)

		// Делаем запрос к серверу
		client := resty.New()
		url := "http://" + s.config.Address.String()

		resp, err := client.R().Post(url + "/nosave")
		require.NoError(t, err)

		assert.Empty(t, resp.String())
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		// Ожидаем запись в файл (но её быть не должно)
		time.Sleep(100 * time.Millisecond)

		// Проверка файла с данными хранилища
		data, err := os.ReadFile(s.config.FileStoragePath)
		require.NoError(t, err)
		assert.Empty(t, data)

		// Второй запрос с сохранением данных хранилища
		resp, err = client.R().Post(url + "/save")
		require.NoError(t, err)

		assert.Empty(t, resp.String())
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		// Ожидаем запись в файл
		time.Sleep(100 * time.Millisecond)

		// Проверка файла с данными хранилища
		data, err = os.ReadFile(s.config.FileStoragePath)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		metrics := []model.Metrics{}
		err = json.Unmarshal(data, &metrics)
		require.NoError(t, err)

		storage := make(storage.MemStorage)
		for _, metric := range metrics {
			storage[metric.ID] = metric
		}

		assert.Equal(t, notEmptyStorage, storage)

		// Завершаем сервер
		select {
		case err := <-errChan:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			fmt.Println("Server forced shutdown")
		}
	})

	t.Run("failed sync save storage", func(t *testing.T) {
		notEmptyStorage := newSuccessMemStorage()

		s := createServer(t, true, "./unknown directory/unknown.json", 0)
		s.storage = &notEmptyStorage

		// Инициализация роутера
		saveMiddleware := s.getSaveMiddleware()
		handlerSave := func(resp http.ResponseWriter, req *http.Request) {}

		router := chi.Router(chi.NewRouter())
		router.Post("/save", saveMiddleware(handlerSave))
		s.router = &router

		// Запуск сервера
		errChan := listenServer(s)
		defer close(errChan)
		time.Sleep(100 * time.Millisecond)

		// Делаем запрос к серверу
		client := resty.New()
		url := "http://" + s.config.Address.String()

		resp, err := client.R().Post(url + "/save")
		require.NoError(t, err)

		assert.Empty(t, resp.String())
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		// Ожидаем запись в файл (но её быть не должно)
		time.Sleep(100 * time.Millisecond)

		// Проверка файла с данными хранилища
		_, err = os.ReadFile(s.config.FileStoragePath)
		require.Error(t, err)

		// Замена на корректный файл
		tempFile, err := os.CreateTemp(os.TempDir(), "storage*.json")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		s.config.FileStoragePath = tempFile.Name()

		// Второй запрос с сохранением данных хранилища
		resp, err = client.R().Post(url + "/save")
		require.NoError(t, err)

		assert.Empty(t, resp.String())
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		// Ожидаем запись в файл (но её быть не должно)
		time.Sleep(100 * time.Millisecond)

		// Проверка файла с данными хранилища
		data, err := os.ReadFile(s.config.FileStoragePath)
		require.NoError(t, err)
		require.Empty(t, data)

		// Завершаем сервер
		select {
		case err := <-errChan:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			fmt.Println("Server forced shutdown")
		}
	})

	t.Run("not sync save storage", func(t *testing.T) {
		notEmptyStorage := newSuccessMemStorage()

		s := createServer(t, true, "", 500*time.Millisecond)
		s.storage = &notEmptyStorage

		// Инициализация роутера
		handler := func(resp http.ResponseWriter, req *http.Request) {}
		handlerSave := s.getSaveMiddleware()(handler)

		router := chi.Router(chi.NewRouter())
		router.Post("/nosave", handler)
		router.Post("/save", handlerSave)
		s.router = &router

		// Запуск сервера
		errChan := listenServer(s)
		defer close(errChan)
		time.Sleep(100 * time.Millisecond)

		// Делаем запрос к серверу
		client := resty.New()
		url := "http://" + s.config.Address.String()

		resp, err := client.R().Post(url + "/nosave")
		require.NoError(t, err)

		assert.Empty(t, resp.String())
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		// Ожидаем запись в файл (но её быть не должно)
		time.Sleep(100 * time.Millisecond)

		// Проверка файла с данными хранилища
		data, err := os.ReadFile(s.config.FileStoragePath)
		require.NoError(t, err)
		assert.Empty(t, data)

		// Второй запрос, который не должен сохранить
		resp, err = client.R().Post(url + "/save")
		require.NoError(t, err)

		assert.Empty(t, resp.String())
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		// Ожидаем запись в файл (но её быть не должно)
		time.Sleep(100 * time.Millisecond)

		// Проверка файла с данными хранилища
		data, err = os.ReadFile(s.config.FileStoragePath)
		require.NoError(t, err)
		assert.Empty(t, data)

		// Ждём ещё время, чтобы прошло пол секунды в сумме (~300 миллисекунд прошло)
		time.Sleep(400 * time.Millisecond) // ~700 миллисекунд

		// Проверка файла с данными хранилища
		data, err = os.ReadFile(s.config.FileStoragePath)
		require.NoError(t, err)

		metrics := []model.Metrics{}
		err = json.Unmarshal(data, &metrics)
		require.NoError(t, err)

		storage := make(storage.MemStorage)
		for _, metric := range metrics {
			storage[metric.ID] = metric
		}

		assert.Equal(t, notEmptyStorage, storage)

		// Завершаем сервер
		select {
		case err := <-errChan:
			require.NoError(t, err)
		case <-time.After(3 * time.Second):
			fmt.Println("Server forced shutdown")
		}
	})
}
