package main

import (
	"time"

	"github.com/skayfish/metrics/internal/agent"
	"github.com/skayfish/metrics/internal/flags"
	"github.com/spf13/pflag"
)

// SF TODO
func parseFlags() agent.Config {
	addr := flags.NetAddress{Host: "localhost", Port: 8080}

	pflag.VarP(&addr, "address", "a", "Server address in format host:port")
	pollInterval := pflag.UintP("poll-interval", "p", 2,
		"Metrics collection frequency, in seconds")
	reportInterval := pflag.UintP("report-interval", "r", 10,
		"Metrics sending frequency to the server, in seconds")
	isSecure := pflag.Bool("secure-connection", false,
		"Secure server connection [HTTP — disabled, HTTPS — enabled] (default false)")
	retryMaxWaitTime := pflag.Uint("retry-max-wait-time", 10,
		"Maximum wait time between connection attempts, in seconds")
	retryWaitTime := pflag.Uint("retry-wait-time", 2,
		"Connection retry interval, in seconds")

	pflag.Parse()

	return agent.Config{
		SecureConnection: *isSecure,
		Host:             addr.Host,
		Port:             addr.Port,
		RetryMaxWaitTime: time.Duration(*retryMaxWaitTime) * time.Second,
		RetryWaitTime:    time.Duration(*retryWaitTime) * time.Second,
		PollInterval:     time.Duration(*pollInterval) * time.Second,
		ReportInterval:   time.Duration(*reportInterval) * time.Second,
	}
}
