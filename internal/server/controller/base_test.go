package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/skayfish/metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Проверяет работу обработчика запроса на проверку соединения
func TestBaseController_Ping(t *testing.T) {
	tests := []struct {
		test          string
		databaseError error
		statusCode    int
	}{
		{
			test:          "success",
			databaseError: nil,
			statusCode:    http.StatusOK,
		},
		{
			test:          "error",
			databaseError: errors.New("some error"),
			statusCode:    http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDatabase := storage.NewMockDatabase(ctrl)
			mockDatabase.EXPECT().PingContext(gomock.Any()).Times(1).Return(tt.databaseError)

			controller := NewBaseController(storage.Storage(mockDatabase))

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
}
