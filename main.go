package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	"github.com/tonimelisma/rfc5424"
)

type logHeaders struct {
	contentType string
	msgCount    int
	frameID     string
	drainToken  string
}

var logFileMutex sync.Mutex
var logFileHandle *os.File

func writeLogLn(logFileHandle *os.File, line string) {
	n, err := logFileHandle.WriteString(line + "\n")
	if err != nil {
		log.Fatal("Error writing to logfile:", n, err.Error())
	}
	if n != len(line)+1 {
		log.Fatal("Didn't write full string:", line)
	}
}
func parseLogHeaders(requestHeaders http.Header) (thisLogHeaders logHeaders, err error) {
	for name, headers := range requestHeaders {
		for _, h := range headers {
			switch name {
			case "Content-Type":
				thisLogHeaders.contentType = h
			case "Logplex-Msg-Count":
				i, err := strconv.Atoi(h)
				if err == nil {
					thisLogHeaders.msgCount = i
				}
			case "Logplex-Frame-Id":
				thisLogHeaders.frameID = h
			case "Logplex-Drain-Token":
				thisLogHeaders.drainToken = h
			}
		}
	}

	if thisLogHeaders.contentType != "application/logplex-1" {
		err = errors.New("invalid content-type: " + thisLogHeaders.contentType)
		return thisLogHeaders, err
	}

	return
}

func loggingHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		log.Println(req.RemoteAddr, req.Method, req.URL.Path, http.StatusMethodNotAllowed, "invalid method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	thisLogHeaders, err := parseLogHeaders(req.Header)
	if err != nil {
		log.Println(req.RemoteAddr, req.Method, req.URL.Path, thisLogHeaders.drainToken, thisLogHeaders.frameID, http.StatusBadRequest, "couldn't parse headers:", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	messageArray, err := rfc5424.ParseMultiple(req.Body)
	if err != nil {
		log.Println(req.RemoteAddr, req.Method, req.URL.Path, thisLogHeaders.drainToken, thisLogHeaders.frameID, http.StatusBadRequest, "couldn't parse body:", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if thisLogHeaders.msgCount != len(messageArray) {
		log.Println(req.RemoteAddr, req.Method, req.URL.Path, thisLogHeaders.drainToken, thisLogHeaders.frameID, http.StatusBadRequest, "message count/header mismatch:", thisLogHeaders.msgCount, len(messageArray))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logFileMutex.Lock()

	for _, msg := range messageArray {
		writeLogLn(logFileHandle, fmt.Sprintf("%v %v.%v %v %v %v %v %v", msg.Timestamp, msg.Facility, msg.Severity, thisLogHeaders.drainToken, msg.Hostname, msg.AppName, msg.ProcID, msg.Message))
	}

	logFileMutex.Unlock()

	log.Println(req.RemoteAddr, req.Method, req.URL.Path, thisLogHeaders.drainToken, thisLogHeaders.frameID, http.StatusOK)
	w.WriteHeader(http.StatusOK)
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

	logFileHandle, err := os.OpenFile(os.Getenv("LOG_DIRECTORY"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal("Error opening logfile:", err.Error())
	}
	defer logFileHandle.Close()

	fmt.Println("Opening HTTP server on", host+":"+port)
	http.HandleFunc("/log", loggingHandler)
	err = http.ListenAndServeTLS(host+":"+port, sslCert, sslKey, nil)
	if err != nil {
		log.Fatal("Error starting HTTP server:", err.Error())
	}
}
