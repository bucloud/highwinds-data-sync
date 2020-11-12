package main

import (
	"net/http"
	"time"
)

var (
	startDate  time.Time = time.Now().AddDate(0, 0, -1).UTC()
	endDate    time.Time = time.Now().UTC()
	authURL    string    = "https://hcs.hwcdn.net/stauth/v1.0"
	storageURL string    = "http://hcs.hwcdn.net/v1/AUTH_hwcdn-logstore"
)

type logSync struct {
	client   http.Client
	userName string
	password string
	method   string // use
}

func init() {
}

func (l *logSync) useFTP() {
	l.method = "FTP"
}

func (l *logSync) useHTTP() {
	l.method = "HTTP"
}

func (l *logSync) auth(username, password string) (string, error) {
	return "1", nil
}
