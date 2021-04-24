package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	"github.com/logrusorgru/aurora/v3"
	"github.com/tonimelisma/rfc5424"
)

type logHeaders struct {
	contentType string
	msgCount    int
	frameID     string
	drainToken  string
}

var logFileMutex sync.Mutex

func writeLogLn(path string, line string) {
	// TODO get file mode from .env file
	logFileHandle, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal("Error opening logfile:", err.Error())
	}
	defer logFileHandle.Close()

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

	if os.Getenv("LOG_DRAIN_TOKEN") != "" {
		if os.Getenv("LOG_DRAIN_TOKEN") != thisLogHeaders.drainToken {
			log.Println(req.RemoteAddr, req.Method, req.URL.Path, thisLogHeaders.drainToken, thisLogHeaders.frameID, http.StatusBadRequest, "invalid drain token")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
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

	// TODO implement duplicate detection via rotating slice of frame IDs

	logFileMutex.Lock()

	for _, msg := range messageArray {
		logFileDir := filepath.Join(os.Getenv("LOG_DIRECTORY"), thisLogHeaders.drainToken)
		// TODO get directory mode from .env file
		os.MkdirAll(logFileDir, 0755)
		logFilePath := filepath.Join(logFileDir, fmt.Sprintf("%v-%v-%v.log", msg.Hostname, msg.AppName, msg.ProcID))
		logLine := aurora.Sprintf(aurora.Magenta("%v %v %v %v %v"), msg.Timestamp, msg.Severity, aurora.Green(msg.AppName), aurora.Green(msg.ProcID), aurora.White(msg.Message))
		writeLogLn(logFilePath, logLine)
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

	if os.Getenv("LOG_DRAIN_TOKEN") == "" {
		log.Println("Drain token not defined - potential security issue: will allow all log messages")
	}

	if os.Getenv("LOG_DIRECTORY") == "" || os.Getenv("LOG_DIR_MODE") == "" || os.Getenv("LOG_FILE_MODE") == "" {
		log.Fatal("Environment variables LOG_DIRECTORY, LOG_DIR_MODE and LOG_FILE_MODE not all set")
	}
	// TODO implement log directory and file mode

	http.HandleFunc("/log", loggingHandler)
	fmt.Println("HTTP server started on", host+":"+port)
	err = http.ListenAndServeTLS(host+":"+port, sslCert, sslKey, nil)
	if err != nil {
		log.Fatal("Error from HTTP server:", err.Error())
	}
}
