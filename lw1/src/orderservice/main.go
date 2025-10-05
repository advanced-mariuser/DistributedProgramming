package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"orderservice/transport"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	file, err := os.OpenFile("orderservice.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
		defer file.Close()
	}

	serverUrl := ":8080"
	log.WithFields(log.Fields{"url": serverUrl}).Info("Starting server")

	killSignalChan := getKillSignalChan()
	srv := startServer(serverUrl)

	waitForKillSignalChan(killSignalChan)
	srv.Shutdown(context.Background())
}

func startServer(serverUrl string) *http.Server {
	router := transport.Router()
	srv := &http.Server{Addr: serverUrl, Handler: router}
	go func() {
		log.Fatal(http.ListenAndServe(serverUrl, router))
	}()

	return srv
}

func getKillSignalChan() chan os.Signal {
	osKillSignalChan := make(chan os.Signal, 1)
	signal.Notify(osKillSignalChan, os.Kill, os.Interrupt, syscall.SIGTERM)
	return osKillSignalChan
}

func waitForKillSignalChan(killSignalChan <-chan os.Signal) {
	killSignal := <-killSignalChan
	switch killSignal {
	case os.Interrupt:
		log.Info("Got SIGINT...")
	case syscall.SIGTERM:
		log.Info("Got SIGTERM...")
	}
}
