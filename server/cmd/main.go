package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	l "github.com/bkenks/bs3-logger"
	"github.com/bkenks/bs3/internal/api"
	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/vault"
	"github.com/charmbracelet/log"
)

type bs3Flag interface {
	GetValue() *bool
}

type verbose struct {
	value *bool
}

func (v verbose) GetValue() *bool { return v.value }

func main() {
	// ~~~ Output Printing ~~~
	l.PrintBS3()

	// -----------------------------------------------------
	// Flag Handling
	// -----------------------------------------------------
	// handles optional flags passed to command

	var bs3Flags []bs3Flag

	bs3Flags = append(bs3Flags,
		verbose{
			value: flag.Bool("verbose", false, "enable debug logging"),
		},
	)

	// ~~~ Set Log Level ~~~
	flag.Parse() // Parse provided flags
	for _, f := range bs3Flags {
		if *f.GetValue() {
			switch f.(type) {
			case verbose:
				l.Logger.SetLevel(log.DebugLevel)
			}
		}
	}
	// -----------------------------------------------------
	// END "Flag Handling"
	// -----------------------------------------------------

	// -----------------------------------------------------
	// Vault Configuration
	// -----------------------------------------------------

	var vault vault.Vault
	var server api.Server
	var err error

	// Check if db exists, set state on vault struct
	if err = vault.CheckVaultState(); err != nil {
		l.LogError(
			l.Logger.Error,
			"faulty vault state", "err", err)
		os.Exit(1)
	}
	server.Vault = &vault // Set vault pointer to vault
	l.LogAddInfo(
		l.Logger.Debug,
		"checked vault state",
		"state", server.Vault.GetState(),
	)
	// -----------------------------------------------------
	// END "Vault Configuration"
	// -----------------------------------------------------

	// -----------------------------------------------------
	// Shutdown Handling Setup
	// -----------------------------------------------------
	// handle shutdowns gracefully

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// -----------------------------------------------------
	// END "Shutdown Handling Setup"
	// -----------------------------------------------------

	// -----------------------------------------------------
	// Serve HTTP
	// -----------------------------------------------------
	// server HTTP to handle requests

	// ~~~ HTTP Logger ~~~
	// wrap defeault charm logger with standard logger for http package
	httpLogger := l.Logger.With("https") // logger for http server to better identify logs
	stdLogWrapper := httpLogger.StandardLog(log.StandardLogOptions{
		ForceLevel: log.ErrorLevel,
	})

	// ~~~ Configure API Routes ~~~
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	// ~~~ Set Open Port ~~~
	port := os.Getenv(constants.ENV_VAR_API_PORT)
	if port == "" {
		port = "8080" // default
		l.LogAddInfo(
			l.Logger.Debug,
			"no custom port found in env vars, using default",
			"port", port,
		)
	}

	// ~~~ HTTP Server Object ~~~
	httpServer := &http.Server{
		Addr:     ":" + port,
		Handler:  mux,
		ErrorLog: stdLogWrapper,
	}

	// Run http server
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.LogError(
				l.Logger.Error,
				"http server failed", "err", err)
		}
	}()

	l.LogAddInfo(
		l.Logger.Info,
		"vault api running",
		"port", port,
	)
	server.InitialToken() // prints initial token if needed
	go vault.StartTokenCleanup(ctx, 24*time.Hour)
	// -----------------------------------------------------
	// END "Serve HTTP"
	// -----------------------------------------------------

	// -----------------------------------------------------
	// Shutdown Handling
	// -----------------------------------------------------
	// handle shutdowns gracefully

	// ~~~ Print Shutdown Instructions ~~~
	shutdownSignal := "ctrl + c"
	l.Logger.Info("shutdown: " + shutdownSignal)
	// fmt.Println("-----------------------------------------------")
	l.PrintLogSeperator()

	// ~~~ Wait For Shutdown Signal ~~~
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		l.LogError(
			l.Logger.Error,
			"forced shutdown", "err", err)
	}

	// ~~~ Vault Graceful Close ~~~
	vault.Close()

	// ~~~ Inform User of Shutdown ~~~
	fmt.Println()
	l.Logger.Info("shutdown complete")
	// -----------------------------------------------------
	// END "Shutdown Handling"
	// -----------------------------------------------------

}
