package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/skayfish/metrics/internal/model"
	"github.com/skayfish/metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UnknownStorage struct{}

func (UnknownStorage) Update(m model.Metrics) (*model.Metrics, error) { return nil, nil }
func (UnknownStorage) UpdateContext(ctx context.Context, m model.Metrics) (*model.Metrics, error) {
	return nil, nil
}
func (UnknownStorage) Get(id string) (*model.Metrics, error) { return nil, nil }
func (UnknownStorage) GetContext(ctx context.Context, id string) (*model.Metrics, error) {
	return nil, nil
}
func (UnknownStorage) GetAll() ([]model.Metrics, error)                           { return nil, nil }
func (UnknownStorage) GetAllContext(ctx context.Context) ([]model.Metrics, error) { return nil, nil }
func (UnknownStorage) Close() error                                               { return nil }

// Проверяет работу обработчика запроса на проверку соединения
func TestBaseController_Ping(t *testing.T) {
	tests := []struct {
		test          string
		databaseError error
		statusCode    int
	}{
		{
			test:          "database: success",
			databaseError: nil,
			statusCode:    http.StatusOK,
		},
		{
			test:          "database: error",
			databaseError: errors.New("some error"),
			statusCode:    http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDatabase := storage.NewMockDatabaseStorage(ctrl)
			mockDatabase.EXPECT().PingContext(gomock.Any()).Times(1).Return(tt.databaseError)

			controller := NewBaseController(mockDatabase)

			router := chi.NewRouter()
			router.Get("/ping", controller.Ping)
			server := httptest.NewServer(router)
			defer server.Close()

			request := resty.New().R()
			resp, err := request.Get(server.URL + "/ping")
			require.NoError(t, err)

			assert.Equal(t, tt.statusCode, resp.StatusCode())
			assert.Equal(t, "", resp.String())
		})
	}

	t.Run("memory storage", func(t *testing.T) {
		storage := storage.NewMemStorage()
		controller := NewBaseController(&storage)

		router := chi.NewRouter()
		router.Get("/ping", controller.Ping)
		server := httptest.NewServer(router)
		defer server.Close()

		request := resty.New().R()
		resp, err := request.Get(server.URL + "/ping")
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Equal(t, "", resp.String())
	})

	t.Run("unknown storage", func(t *testing.T) {
		storage := UnknownStorage{}
		controller := NewBaseController(&storage)

		router := chi.NewRouter()
		router.Get("/ping", controller.Ping)
		server := httptest.NewServer(router)
		defer server.Close()

		request := resty.New().R()
		resp, err := request.Get(server.URL + "/ping")
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
		assert.Equal(t, "", resp.String())
	})
}
