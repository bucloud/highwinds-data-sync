package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"syscall"
	"time"

	_ "net/http/pprof"
)

var (
	numWorker              int           = 20
	maxConn                int           = 20
	connTimeout            time.Duration = time.Second * 60
	idleConnTimeout        time.Duration = time.Second * 30
	keepAlive              time.Duration = time.Second * 60
	buckets                time.Duration = time.Hour * 2
	late                   time.Duration = time.Second * 15
	fixTime                string        = ""
	gcInterval             time.Duration = time.Minute
	configureSync          bool          = true
	certificatesSync       bool          = true
	enablePPROF            bool          = false
	pprofHost              string        = "localhost:6060"
	esHost                 string        = "http://localhost:9200"
	esUserName             string        = "elastic"
	esPassword             string        = "changeme"
	esRetries              int           = 3
	esWorker               int           = 2
	certificatesIndex      string        = "config_certs"
	configureIndex         string        = "config_hosts"
	indexPrefix            string        = ""
	enableESLog            bool          = false
	esLoggerEnabelRequest  bool          = false
	esLoggerEnableResponse bool          = false
	logOutPut              io.Writer
)

func init() {
	// Parse value from ENV
	// elasticsearch
	if v, b := os.LookupEnv("esHost"); b {
		esHost = v
	}
	if v, b := os.LookupEnv("esUserName"); b {
		esUserName = v
	}
	if v, b := os.LookupEnv("esPassword"); b {
		esPassword = v
	}
	if v, b := os.LookupEnv("indexPrefix"); b {
		indexPrefix = v
	}
	if v, b := os.LookupEnv("configureIndex"); b {
		configureIndex = v
	}
	if v, b := os.LookupEnv("certificatesIndex"); b {
		certificatesIndex = v
	}
	if v, b := os.LookupEnv("storeMetrics"); b {
		storeMetrics = stringToMetrics(v)
	}
	var tokens string
	if v, b := os.LookupEnv("tokens"); b {
		tokens = v
	}

	// Rewrite value from flag
	// common flags
	flag.IntVar(&numWorker, "worker", numWorker, "Set worker for data sync process")
	flag.IntVar(&maxConn, "maxConn", maxConn, "Set max connection for HWApi")
	flag.DurationVar(&connTimeout, "timeout", connTimeout, "Set timeout for API connect")
	flag.DurationVar(&idleConnTimeout, "idleConnTimeout", idleConnTimeout, "Set idle connection timeout")
	flag.DurationVar(&keepAlive, "k", keepAlive, "Set keep alive timeout")
	flag.DurationVar(&buckets, "bs", buckets, "Set bucket size for data sync, unit minute")
	flag.DurationVar(&late, "d", late, "Set offset for data sync, unit minute")
	flag.DurationVar(&gcInterval, "gc", gcInterval, "Set gc interval")
	flag.BoolVar(&configureSync, "config", configureSync, "Enable configure sync (pull configur into ES)")
	flag.BoolVar(&certificatesSync, "cert", certificatesSync, "Enable certificates sync (pull certificates into ES)")
	flag.StringVar(&fixTime, "time", fixTime, "Set fixed time, all data sync job would run with startDate fixTime and endDate fixTime+buckets")
	flag.StringVar(&tokens, "tokens", "", "Set tokens list, accounthash must included, for example a1b1c1d1:abcdef123asd;")
	for _, k := range strings.Split(tokens, ";") {
		for _, k2 := range strings.Split(k, ":") {
			tokenList[string(k2[0])] = string(k2[1])
		}
	}
	if len(tokenList) < 1 {
		log.Panicf("tokens is required")
	}

	var ts string
	ts = metricsToString(storeMetrics)
	flag.StringVar(&ts, "metrics", ts, "metrics you want to synchronised, provide metrics type and names, for example transfer:rps,xferRateMbps;status:rps")
	storeMetrics = stringToMetrics(ts)

	flag.BoolVar(&enablePPROF, "pprof", enablePPROF, "Enable PPROF")
	flag.StringVar(&pprofHost, "pprofHost", pprofHost, "Set listen host for PPROF")

	// es configure from command flags
	flag.StringVar(&esHost, "esHost", esHost, "Set elasticsearch server login username, esHost avaialble in ENV")
	flag.StringVar(&esUserName, "esUser", esUserName, "Set elasticsearch server login username, esUserName available in ENV")
	flag.StringVar(&esPassword, "esPWD", esPassword, "Set elasticsearch server login password, esPassword available in ENV")
	flag.IntVar(&esRetries, "esRetires", esRetries, "Set elasticsearch maxRetries")
	flag.IntVar(&esWorker, "esWorker", esWorker, "Set elasticsearch max workers")
	flag.BoolVar(&enableESLog, "eslog", enableESLog, "Enable elasticsearch log")
	flag.BoolVar(&esLoggerEnabelRequest, "loggerReq", esLoggerEnabelRequest, "Enable elasticsearch request body in logger")
	flag.BoolVar(&esLoggerEnableResponse, "loggerRes", esLoggerEnableResponse, "Enable elasticsearch response body in logger")
	flag.StringVar(&indexPrefix, "indexPrefix", indexPrefix, "Set elasticsearch index prefix, usable when testing, indexPrefix available in ENV")
	flag.StringVar(&configureIndex, "hostsIndex", configureIndex, "Set index name which contains hosts configuration, configureIndex available in ENV")
	flag.StringVar(&certificatesIndex, "certsIndex", certificatesIndex, "Set index name which contains certificates info, certificates available in ENV")

	outPut := *flag.String("out", "/dev/stdout", "Set log output file")

	flag.Parse()

	logOutPut = os.NewFile(uintptr(syscall.Stdout), outPut)
	if enablePPROF {
		go func() {
			log.Println(http.ListenAndServe(pprofHost, nil))
		}()
	}
}

func searchIndexByValue(m []string, s string) int {
	for i, v := range m {
		if strings.ToUpper(v) == strings.ToUpper(s) {
			return i
		}
	}
	return 0
}

func inSlice(sl []string, s string) bool {
	for _, v := range sl {
		if strings.ToUpper(v) == strings.ToUpper(s) {
			return true
		}
	}
	return false
}

func stringToMetrics(s string) map[string][]string {
	// split string to array by ;
	r := map[string][]string{}
	as := strings.Split(s, ";")
	for _, k := range as {
		// parse type and keys from string
		ts := strings.Split(k, ":")
		if len(ts) != 1 {
			log.Panicf("metrics format failed, found %d colon in metric string, 1 is exptected", len(ts))
		}
		switch ts[0] {
		case "transfer", "storage", "status":
			r[ts[0]] = strings.Split(ts[1], ",")
		default:
			log.Panicf("metric %s is not supported", ts[0])
		}
	}
	return r
}

func metricsToString(v map[string][]string) string {
	t := reflect.ValueOf(v).MapKeys()
	var s []string
	for _, k := range t {
		ts := k.String()
		ts += ":" + strings.Join(v[ts], ",")
		s = append(s, ts)
	}
	return strings.Join(s, ";")
}

// func sliceComparison(sl map[str])
