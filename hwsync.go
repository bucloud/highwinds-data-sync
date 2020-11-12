package main

import (
	"log"
	"net"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bucloud/hwapi"
	"github.com/elastic/go-elasticsearch/v7"
)

type metricQuery struct {
	Account    string
	SubAccount hwapi.SimpleAccount
	Host       hwapi.Host
	StartDate  time.Time
	EndDate    time.Time
	CreateDate time.Time
}

type timeGroup struct {
	StartDate time.Time
	EndDate   time.Time
}

type metricQueryResult struct {
	Time    time.Time
	Result  bool
	Message string
}

type dataClassify struct {
	POPs        []*hwapi.POP
	GroupBy     string
	Granularity string
}

type accountSpace struct {
	SuperAccount string
	// Accounts     []*hwapi.SimpleAccount
	// Certs        map[string][]*hwapi.Certificate
	// Hosts    map[string]map[string]*hwapi.Host
	Accounts atomic.Value
	Certs    atomic.Value
	Hosts    atomic.Value
	JobPool  atomic.Value
	API      *hwapi.HWApi
	log      log.Logger
}

const (
	// ISO8601 time format
	ISO8601 string = "2006-01-02T15:04:05Z"
	// DateTime date format
	DateTime string = "20060102"
)

var (
	tokenList map[string]string = map[string]string{}
	popList   *hwapi.POPs       = &hwapi.POPs{}
	wg        sync.WaitGroup
	es        *elasticsearch.Client
	rlog      log.Logger
)

func getPOPInfoByPOPCode(id string) *hwapi.POP {
	for _, pop := range popList.List {
		if strings.ToUpper(pop.Code) == strings.ToUpper(id) {
			return pop
		}
	}
	return &hwapi.POP{}
}

func main() {
	// try use first token to update POPs list
	tr := http.Transport{
		IdleConnTimeout: idleConnTimeout,
		Proxy:           nil,
		MaxConnsPerHost: maxConn,
		DialContext: (&net.Dialer{
			Timeout:   connTimeout,
			KeepAlive: keepAlive,
			DualStack: true,
		}).DialContext,
	}
	rlog.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	rlog.SetPrefix("Default ")
	rlog.SetOutput(logOutPut)
	go func(token string) {
		wg.Add(1)
		for {
			api := hwapi.Init(&tr)
			api.AuthToken = &hwapi.AuthToken{
				TokenType:   "Bearer",
				AccessToken: token,
			}
			pl, e := api.GetPoPs()
			if e != nil {
				rlog.Fatal("Get POPs list failed " + e.Error())
			} else {
				rlog.Printf("Get POPs list succeed, example popInfo %v", pl.List[0])
				popList = pl
			}
			time.Sleep(time.Hour * 6)
		}
	}(func(m map[string]string) string {
		t := reflect.ValueOf(m).MapRange()
		t.Next()
		return t.Value().String()
	}(tokenList))
	// try init es instance
	elastic, e := initES()
	if e != nil {
		rlog.Fatalf("Init elasticsearch client failed %s", e.Error())
	} else {
		es = elastic
	}
	go func(i time.Duration) {
		for {
			debug.FreeOSMemory()
			time.Sleep(i)
		}
	}(gcInterval)
	// exists when one of process exists
	for name, token := range tokenList {
		tempChan := make(chan metricQuery, numWorker*2)
		as := &accountSpace{
			SuperAccount: name,
			Accounts:     atomic.Value{},
			Hosts:        atomic.Value{},
			Certs:        atomic.Value{},
			log:          log.Logger{},
		}
		as.JobPool.Store([]*metricQuery{})
		api := hwapi.Init(&tr)
		api.AuthToken = &hwapi.AuthToken{
			TokenType:   "Bearer",
			AccessToken: token,
		}
		as.API = api
		as.log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		as.log.SetPrefix(name + " ")
		as.log.SetOutput(logOutPut)
		// Start accounts sync process
		go as.accountsSync()
		// Start Certlist sync process
		if certificatesSync {
			go as.certsSync()
		}
		// // Start Hosts sync process
		if configureSync {
			go as.hostsSync()
		}
		// Start data sync worker
		for i := 0; i <= numWorker; i++ {
			go as.getDataWorker(tempChan)
		}
		// Start Job Pool detect process
		go func(ss *accountSpace) {
			wg.Add(1)
			for {
				if len(ss.JobPool.Load().([]*metricQuery)) > 0 && len(tempChan) <= int(float32(numWorker)*0.8) {
					tempChan <- *ss.consumeJob()
					ss.log.Printf("channel buffer usage %d, job pool size %d", len(tempChan)/cap(tempChan), len(ss.JobPool.Load().([]*metricQuery)))
				}

				time.Sleep(time.Millisecond * 100)
			}
		}(as)
		// Start data sync process
		go func(ss *accountSpace) {
			wg.Add(1)
			for {
				tempStartTimeStamp := time.Now().UTC()
				jT := &timeGroup{
					StartDate: tempStartTimeStamp.Add(-buckets - late),
					EndDate:   tempStartTimeStamp.Add(-late),
				}
				if fixTime != "" {
					if nt, e := time.Parse(ISO8601, fixTime); e != nil {
						ss.log.Fatalf("fixTime error, must in ISO8601(%s) format", ISO8601)
					} else {
						jT.StartDate = nt
						jT.EndDate = nt.Add(buckets)
					}
				}
				for _, account := range ss.getAccounts() {
					if account.AccountStatus == "DELETED" {
						continue
					}
					// get alive hostHash list
					for _, h := range ss.transferDataOfAccount(account.AccountHash) {
						if ss.getHosts()[account.AccountHash] == nil {
							// get hosts list
							if ah, e := ss.API.GetHosts(account.AccountHash); e == nil {
								ss.setHostsByKey(account.AccountHash, ah.Hosts())
							} else {
								ss.log.Printf("Account %s doesn't owned hosts list in mem and failed in api fetch %s", account.AccountHash, e.Error())
								continue
							}
						}
						if ss.getHosts()[account.AccountHash][h] == nil {
							if ah, e := ss.API.GetHost(account.AccountHash, h); e == nil {
								ss.setHostForAccount(account.AccountHash, h, ah)
							} else {
								ss.log.Printf("Account %s hostlist found in mem but hostHash %s not included and failed in api fetch %s", account.AccountHash, h, e.Error())
								continue
							}
						}
						ss.addMetricsJob(&metricQuery{
							Account:    ss.SuperAccount,
							Host:       *ss.getHostByHash(h),
							SubAccount: *account,
							StartDate:  jT.StartDate,
							EndDate:    jT.EndDate,
							CreateDate: time.Now().UTC(),
						})
					}
				}
				// for a := range ss.getHosts() {
				// 	account := ss.getAccountByAccountHash(a)
				// 	if account == nil {
				// 		continue
				// 	}
				// 	for _, h := range ss.transferDataOfAccount(a) {
				// 		// Throw job to jobPool
				// 		ss.addMetricsJob(&metricQuery{
				// 			Account:    ss.SuperAccount,
				// 			Host:       *h,
				// 			SubAccount: *account,
				// 			StartDate:  tempStartTimeStamp.Add(-time.Minute * 135),
				// 			EndDate:    tempStartTimeStamp.Add(-time.Minute * 15),
				// 			CreateDate: time.Now().UTC(),
				// 		})
				// 	}
				// }
				ss.log.Printf("Last data sync job spent %f seconds, next round will start after %f seconds", time.Since(tempStartTimeStamp).Seconds(), 60*5-time.Since(tempStartTimeStamp).Seconds())
				if len(ss.getHosts()) == 0 {
					time.Sleep(time.Second * 3)
				} else {
					time.Sleep(time.Duration(300-tempStartTimeStamp.Unix()%300) * time.Second)
				}
			}
		}(as)
	}
	wg.Wait()
}
