package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type logHeaders struct {
	contentType string
	msgCount    int
	frameID     int
	drainToken  string
}

type logMessage struct {
	priority  int
	version   int
	timestamp time.Time
	hostname  string
	appname   string
	procid    string
	message   string
}

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
				i, err := strconv.Atoi(h)
				if err == nil {
					thisLogHeaders.frameID = i
				}
			case "Logplex-Drain-Token":
				thisLogHeaders.drainToken = h
			}
		}
	}

	if thisLogHeaders.contentType != "application/logplex-1" {
		err = error.Error("invalid content-type: " + thisLogHeaders.contentType)
		return thisLogHeaders, err
	}

	return
}

func parseLogBody(requestBody io.ReadCloser) (thisLogMessage logMessage, err error) {
	// TODO write parsing logic
	reader := bufio.NewReader(requestBody)
	octetLength, err := reader.ReadString(byte(" "))

	if err != nil {
		return thisLogMessage, err
	}

	fmt.Println("length", octetLength)
	thisLogMessage.message = "diipa daapa duupa"

	// TODO create array instead of single message
	return thisLogMessage, nil
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

	thisLogHeaders, err := parseLogHeaders(req.Header)
	if err != nil {
		log.Print("couldn't parse body", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	writeLogLn(logFileHandle, fmt.Sprintf("frame %v, messages %v, drain token %v", thisLogHeaders.frameID, thisLogHeaders.msgCount, thisLogHeaders.drainToken))

	thisLogMessage, err := parseLogBody(req.Body)
	if err != nil {
		log.Print("couldn't parse body", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	writeLogLn(logFileHandle, fmt.Sprintf("%v", thisLogMessage.message))

	logFileMutex.Unlock()
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

	fmt.Println("Opening HTTP server on", host+":"+port)
	http.HandleFunc("/log", loggingHandler)
	err = http.ListenAndServeTLS(host+":"+port, sslCert, sslKey, nil)
	if err != nil {
		log.Fatal("Error starting HTTP server:", err.Error())
	}
}
