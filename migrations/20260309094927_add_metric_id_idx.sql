-- +goose Up
-- Создание индекса для столбца id в таблице metrics
CREATE UNIQUE INDEX metric_id_idx ON metrics_schema.metrics (id);

-- +goose Down
-- Удаление индекса для столбца id в таблице metrics
DROP INDEX metrics_schema.metric_id_idx;
