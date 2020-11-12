package main

import (
	"strings"
	"time"

	"github.com/bucloud/hwapi"
)

type certificateIncludeAccount struct {
	AccountHash string `json:"account_hash,omitempty"`
	*hwapi.Certificate
}

func (as *accountSpace) getCerts() map[string][]*hwapi.Certificate {
	a := as.Certs.Load()
	if a != nil {
		return a.(map[string][]*hwapi.Certificate)
	}
	return map[string][]*hwapi.Certificate{}
}

func (as *accountSpace) setCerts(c map[string][]*hwapi.Certificate) {
	as.Certs.Store(c)
}

func (as *accountSpace) setCertsByKey(k string, v []*hwapi.Certificate) {
	temp := as.getCerts()
	temp[k] = v
	as.setCerts(temp)
}

func (as *accountSpace) certsSync() {
	wg.Add(1)
	for {
		tempStartTimeStamp := time.Now()
		for _, account := range as.getAccounts() {
			// Get cert list
			certs, e := as.API.GetCertificates(account.AccountHash)
			if e != nil {
				as.log.Println(e.Error())
			} else {
				as.setCertsByKey(account.AccountHash, certs.List)

				// Get cert list in es
				esCerts, err := searchDoc(
					es.Search.WithIgnoreUnavailable(true),
					es.Search.WithIndex(certificatesIndex),
					es.Search.WithSource([]string{"*Date", "fingerprint"}...),
					es.Search.WithSize(1000),
					es.Search.WithBody(strings.NewReader(`{"query":{"bool":{"must":[{"match":{"account_hash":"`+account.AccountHash+`"}},{"bool":{"should":[{"range":{"deletedDate":{"gte":"now"}}},{"bool":{"must_not":[{"exists":{"field":"deletedDate"}}]}}]}}]}}}`)),
				)
				if err != nil {
					as.log.Fatalf("Search certificates in es failed %s", err.Error())
				} else {
					// Check added certs
					as.log.Printf("Search certificates in for account %s succeed, spent %d seconds, isTimedout %v", account.AccountHash, esCerts.Took, esCerts.Timedout)
					esCertsString := esCerts.Hits.toString()
					fList := []string{}
					for _, c := range certs.List {
						fList = append(fList, c.Fingerprint)
						if strings.Index(esCertsString, c.Fingerprint) < 0 {
							// c.Fingerprint newlyAdded
							if _, e := indexDoc(certificatesIndex, account.AccountHash+"-"+c.Fingerprint, &certificateIncludeAccount{
								account.AccountHash,
								c,
							}); e != nil {
								as.log.Fatalf("Index certificate failed commonName %s certificateID %s error %s", c.CommonName, c.Fingerprint, e.Error())
							} else {
								as.log.Printf("Index certificate succeed commonName %s certificacteID %s", c.CommonName, c.Fingerprint)
							}
						}
					}

					// Check deleted certs
					for _, c := range esCerts.Hits.Hits {
						if !inSlice(fList, c.Source["fingerprint"].(string)) {
							if _, e := updateDoc(certificatesIndex, `{"conflicts":"proceed","query": { "ids": {"values": ["`+c.ID+`"]}},"script": {"source": "ctx._source[\"deletedDate\"]=\"`+time.Now().UTC().Format(ISO8601)+`\"", "lang": "painless"}}`); e != nil {
								as.log.Fatalf("Update deletedDate for cert %s failed, %s", c.Source["commonName"], e.Error())
							} else {
								as.log.Printf("Update deletedDate for cert %s succeed", c.Source["commonName"])
							}
						}
					}
				}
			}
		}

		as.log.Printf("Last certificates sync spent %f seconds, next round will start after %f seconds", time.Since(tempStartTimeStamp).Seconds(), 20*60-time.Since(tempStartTimeStamp).Seconds())
		if time.Since(tempStartTimeStamp).Seconds() <= 3 {
			time.Sleep(10 * time.Second)
		} else {
			time.Sleep(time.Duration(20*60-time.Since(tempStartTimeStamp).Seconds()) * time.Second)
		}

	}
}
