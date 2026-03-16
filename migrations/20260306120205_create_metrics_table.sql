-- +goose Up
-- Создание схемы для метрик
CREATE SCHEMA metrics_schema;

-- Создание таблицы метрик
CREATE TABLE metrics_schema.metrics (
    id VARCHAR (100) NOT NULL UNIQUE,
    type VARCHAR(10) NOT NULL CHECK (type IN ('gauge', 'counter')),
    delta BIGINT,
    value DOUBLE PRECISION,
    hash VARCHAR(255)
);

-- +goose Down
-- Удаление таблицы метрик
DROP TABLE IF EXISTS metrics_schema.metrics;

-- Удаление схемы для метрик
DROP SCHEMA IF EXISTS metrics_schema CASCADE;
