package model_test

import (
	"testing"

	"github.com/skayfish/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SF TODO
func TestMetrics_Valid(t *testing.T) {
	value := -53.12
	delta := int64(770)

	tests := []struct {
		test         string
		metric       model.Metrics
		errorMessage string
	}{
		{
			test:         "gauge value empty",
			metric:       model.Metrics{MType: model.Gauge},
			errorMessage: model.ErrGaugeValueIsEmpty.Error(),
		},
		{
			test:         "counter delta empty",
			metric:       model.Metrics{MType: model.Counter},
			errorMessage: model.ErrCounterDeltaIsEmpty.Error(),
		},
		{
			test:         "unrecognized metric type",
			metric:       model.Metrics{MType: "unknown"},
			errorMessage: model.ErrUnrecognizedMetricType.Error(),
		},
		{
			test:   "valid",
			metric: model.Metrics{MType: model.Gauge, Value: &value},
		},
		{
			test:   "valid",
			metric: model.Metrics{MType: model.Counter, Delta: &delta},
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			err := tt.metric.Valid()
			if tt.errorMessage == "" {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
				assert.Equal(t, tt.errorMessage, err.Error())
			}
		})
	}
}
