package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

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
	logFileHandle, err := os.OpenFile(os.Getenv("LOG_DIRECTORY"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
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

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Couldn't load .env file")
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "443"
	}
	sslCert := os.Getenv("SSL_CERT_FILE")
	if sslCert == "" {
		log.Fatal("Environment variable SSL_CERT_FILE not defined")
	}
	sslKey := os.Getenv("SSL_KEY_FILE")
	if sslKey == "" {
		log.Fatal("Environment variable SSL_KEY_FILE not defined")
	}

	if os.Getenv("LOG_DIRECTORY") == "" || os.Getenv("LOG_DIR_MODE") == "" || os.Getenv("LOG_FILE_MODE") == "" {
		log.Fatal("Environment variables LOG_DIRECTORY, LOG_DIR_MODE and LOG_FILE_MODE not all set")
	}
	// TODO implement log directory and file mode

	fmt.Println("Opening HTTP server on", host+":"+port)
	http.HandleFunc("/log", loggingHandler)
	err = http.ListenAndServeTLS(host+":"+port, sslCert, sslKey, nil)
	if err != nil {
		log.Fatal("Error starting HTTP server:", err.Error())
	}
}
