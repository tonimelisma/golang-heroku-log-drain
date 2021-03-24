package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

const ipAddress = "0.0.0.0"
const port = "8443"
const logFilePath = "/tmp/heroku.log"
const logFileMode = 0644

var logFileMutex sync.Mutex

func loggingHandler(w http.ResponseWriter, req *http.Request) {

	logFileMutex.Lock()
	logFileHandle, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, logFileMode)
	if err != nil {
		log.Fatal("Error opening logfile:", err.Error())
	}
	defer logFileHandle.Close()

	log.Printf("New request %v", req.Method)
	n, err := logFileHandle.WriteString(req.Host)
	if err != nil {
		log.Fatal("Error writing to logfile:", n, err.Error())
	}
	if n != len(req.Host) {
		log.Println("Didn't write full string:", req.Host)
	}

	for name, headers := range req.Header {
		for _, h := range headers {
			log.Printf("%v: %v\n", name, h)
		}
	}
	log.Printf("\n")
	logFileMutex.Unlock()

	w.WriteHeader(200)
}

func main() {
	fmt.Println("Starting up golang-heroku-log-drain...")

	http.HandleFunc("/log", loggingHandler)
	http.ListenAndServe(ipAddress+":"+port, nil)
}
