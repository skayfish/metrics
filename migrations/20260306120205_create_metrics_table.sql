-- +goose Up
-- Создание схемы для метрик
CREATE SCHEMA IF NOT EXISTS metrics_schema_test;

-- Создание таблицы метрик
CREATE TABLE IF NOT EXISTS metrics_schema_test.metrics (
    id VARCHAR (100) NOT NULL UNIQUE,
    metric_type VARCHAR(10) NOT NULL CHECK (metric_type IN ('gauge', 'counter')),
    delta BIGINT,
    value DOUBLE PRECISION,
    hash VARCHAR(255)
);

-- +goose Down
-- Удаление таблицы метрик
DROP TABLE IF EXISTS metrics_schema_test.metrics;

-- Удаление схемы для метрик
DROP SCHEMA IF EXISTS metrics_schema_test CASCADE;
