package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/urcomputeringpal/kubevalidator/validator"
)

func runWithContext(ctx context.Context) error {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}
	portInt, _ := strconv.Atoi(port)

	webhookSecret, ok := os.LookupEnv("WEBHOOK_SECRET")
	if !ok {
		return errors.New("WEBHOOK_SECRET required")
	}

	appID, ok := os.LookupEnv("APP_ID")
	if !ok {
		return errors.New("APP_ID required")
	}
	appIDInt, _ := strconv.Atoi(appID)

	privateKeyFile, ok := os.LookupEnv("PRIVATE_KEY_FILE")
	if !ok {
		return errors.New("PRIVATE_KEY_FILE required")
	}

	v := &validator.Server{
		Port:           portInt,
		WebhookSecret:  webhookSecret,
		AppID:          appIDInt,
		PrivateKeyFile: privateKeyFile,
	}

	return v.Run(ctx)
}

func cancelOnInterrupt(ctx context.Context, f context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-term:
			log.Println("Received SIGTERM, exiting gracefully...")
			f()
			os.Exit(0)
		case <-ctx.Done():
			os.Exit(0)
		}
	}
}

func run() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go cancelOnInterrupt(ctx, cancelFunc)

	return runWithContext(ctx)
}

func main() {
	if err := run(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		panic(err)
	}
}
