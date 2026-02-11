package main

import (
	"github.com/skayfish/metrics/internal/flags"
	"github.com/spf13/pflag"
)

// Парсит флаги, указанные при запуске программы
//
//	@returns данные о хосте и порте
func parseFlags() flags.NetAddress {
	addr := flags.NetAddress{Host: "localhost", Port: 8080}
	pflag.VarP(&addr, "address", "a", "Server listening address in host:port format")
	pflag.Parse()

	return addr
}
