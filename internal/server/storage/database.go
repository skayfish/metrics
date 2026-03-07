package storage

import (
	"context"
	"database/sql"
)

// Интерфейс базы данных
type Database interface {
	// Проверяет, что соединение с базой данных всё ещё активно
	//
	//	@param ctx контекст для завершения
	//	@returns error ошибку, если соединение не активно
	PingContext(ctx context.Context) error

	// Выполняет запрос, возвращающий строки из базы данных — обычно это запрос SELECT.
	//
	//	@param ctx   контекст для завершения
	//	@param query запрос в виде строки
	//	@param args  аргументы запроса
	//	@returns *sql.Rows результирующие строки, в случае успеха
	//	@returns error ошибку, в ином случае
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// Выполняет запрос, который, как ожидается, вернёт не более одной строки.
	//
	// QueryRowContext всегда возвращает ненулевое значение. Ошибки откладываются до тех пор,
	// пока не будет вызван метод Scan у объекта [Row].
	//
	// Если запрос не выбирает ни одной строки, метод [*Row.Scan] вернёт [ErrNoRows].
	// В противном случае [*Row.Scan] считывает первую выбранную строку и игнорирует остальные.
	//
	//	@param ctx   контекст для завершения
	//	@param query запрос в виде строки
	//	@param args  аргументы запроса
	//	@returns *sql.Row запрошенная строка
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

	// Выполняет подготовленный оператор с указанными аргументами и
	// возвращает объект [Result], содержащий сводную информацию о результате выполнения оператора.
	//
	//	@param ctx   контекст для завершения
	//	@param query запрос в виде строки
	//	@param args  аргументы запроса
	//	@returns sql.Result объект, содержащий сводную информацию о результате выполнения оператора в случае успеха
	//	@returns error ошибку в ином случае
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
