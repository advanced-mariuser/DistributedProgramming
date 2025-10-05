package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"urlsolver/mapping"
	"urlsolver/transport"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	file, err := os.OpenFile("urlsolver.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
		defer file.Close()
	}

	mappingFilePath := flag.String("file", "mapping.json", "Path to the URL mapping JSON file")
	fallbackUrl := flag.String("fallback-url", "https://google.com", "Fallback URL for not found short links")
	flag.Parse()

	urlMapping, err := mapping.LoadMapping(*mappingFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn("Mapping file not found, starting with empty set.")
			urlMapping = make(map[string]string)
		} else {
			log.WithError(err).Fatal("Failed to load URL mapping")
		}
	}

	serverUrl := ":8080"
	log.WithFields(log.Fields{"url": serverUrl}).Info("Starting server")

	port := strings.TrimPrefix(serverUrl, ":")

	killSignalChan := getKillSignalChan()
	srv := startServer(serverUrl, urlMapping, *fallbackUrl, *mappingFilePath, port)

	waitForKillSignalChan(killSignalChan)
	srv.Shutdown(context.Background())
}

func startServer(serverUrl string, urlMapping map[string]string, fallbackUrl, mappingFilePath string, port string) *http.Server {
	router := transport.Router(urlMapping, fallbackUrl, mappingFilePath, port)
	srv := &http.Server{Addr: serverUrl, Handler: router}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Failed to start server")
		}
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
