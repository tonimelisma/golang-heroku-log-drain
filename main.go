package main

import (
	"fmt"
	"io"
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

func writeLogLn(logFileHandle *os.File, line string) {
	n, err := logFileHandle.WriteString(line + "\n")
	if err != nil {
		log.Fatal("Error writing to logfile:", n, err.Error())
	}
	if n != len(line)+1 {
		log.Fatal("Didn't write full string:", line)
	}
}

func loggingHandler(w http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, req.Method, req.URL.Path)

	if req.Method != "POST" {
		log.Println("Invalid method:", req.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	logFileMutex.Lock()
	logFileHandle, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, logFileMode)
	if err != nil {
		log.Fatal("Error opening logfile:", err.Error())
	}
	defer logFileHandle.Close()

	writeLogLn(logFileHandle, "New request: "+req.Method)
	for name, headers := range req.Header {
		for _, h := range headers {
			writeLogLn(logFileHandle, fmt.Sprintf("%v: %v", name, h))
		}
	}
	writeLogLn(logFileHandle, "")
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println("couldn't read body")
		w.WriteHeader(http.StatusBadRequest)
	}
	writeLogLn(logFileHandle, string(requestBody))
	logFileMutex.Unlock()

	w.WriteHeader(200)
}

func main() {
	fmt.Println("Starting up golang-heroku-log-drain...")

	http.HandleFunc("/log", loggingHandler)
	http.ListenAndServe(ipAddress+":"+port, nil)
}
