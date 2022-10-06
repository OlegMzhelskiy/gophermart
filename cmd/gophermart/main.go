package main

import (
	"fmt"
	"github.com/OlegMzhelskiy/gophermart/internal/apiserver"
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := apiserver.NewConfig()
	err := run(cfg)
	if err != nil {
		cfg.Logger.Fatal("http-server failed", err)
	}
}

func run(cfg apiserver.Config) error {
	store, err := storage.NewSQLStore(cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("db connection error: %w", err)
	}
	cfg.Store = store
	srv := apiserver.NewServer(cfg)

	go func() {
		if err := srv.Run(); err != nil {
			cfg.Logger.Fatal("http-server failed", err)
			//return err
		}
	}()

	signalChanel := make(chan os.Signal, 1)
	signal.Notify(signalChanel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-signalChanel

	srv.Stop()
	return nil
}
