package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bucloud/hwapi"
)

type simpleESHostsResponse struct {
	Scope          *Scope          `json:"scope,omitempty"`
	OriginPullHost *OriginPullHost `json:"originPullHost,omitempty"`
	HostHash       string          `json:"host_hash,omitempty"`
}

type OriginPullHost struct {
	Secondary *hwapi.Origin `json:"secondary,omitempty"`
	Primary   *hwapi.Origin `json:"primary,omitempty"`
	Path      string        `json:"path,omitempty"`
	ID        int           `json:"id,omitempy"`
}

type Scope struct {
	ID          int    `json:"id,omitempty"`
	Path        string `json:"path,omitempty"`
	Name        string `json:"name,omitempty"`
	Platform    string `json:"platform,omitempty"`
	DeletedDate string `json:"deletedDate,omitempty"`
	UpdatedDate string `json:"updatedDate,omitempty"`
	CreatedDate string `json:"createdDate,omitempty"`
}
type customConfiguration struct {
	*hwapi.Configuration
	AccountHash string `json:"account_hash,omitempty"`
	HostHash    string `json:"host_hash,omitempty"`
}

type originsMap map[string]*hwapi.Origin

func (o originsMap) originComparison(compareOrigin *simpleESHostsResponse) bool {
	// configure doesn't contains any origin info return true
	if compareOrigin.OriginPullHost == nil {
		return true
	}
	// test primary&backup separately
	return (compareOrigin.OriginPullHost.Primary == nil || (o[fmt.Sprintf("%d", compareOrigin.OriginPullHost.Primary.Id)].UpdatedDate == compareOrigin.OriginPullHost.Primary.UpdatedDate)) &&
		(compareOrigin.OriginPullHost.Secondary == nil || (o[fmt.Sprintf("%d", compareOrigin.OriginPullHost.Secondary.Id)].UpdatedDate == compareOrigin.OriginPullHost.Secondary.UpdatedDate))
}

func (as *accountSpace) getHosts() map[string]map[string]*hwapi.Host {
	a := as.Hosts.Load()
	if a != nil {
		return a.(map[string]map[string]*hwapi.Host)
	}
	return map[string]map[string]*hwapi.Host{}
}

func (as *accountSpace) setHosts(h map[string]map[string]*hwapi.Host) {
	as.Hosts.Store(h)
}

func (as *accountSpace) setHostsByKey(h string, v map[string]*hwapi.Host) {
	temp := as.getHosts()
	temp[h] = v
	as.setHosts(temp)
}

func (as *accountSpace) getHostByHash(h string) *hwapi.Host {
	for _, ah := range as.getHosts() {
		if ah[h] != nil {
			return ah[h]
		}
	}
	return nil
}

func (as *accountSpace) setHostForAccount(accountHash string, hostHash string, v *hwapi.Host) {
	tempA := as.getHosts()
	if tempA[accountHash] == nil {
		tempA[accountHash] = map[string]*hwapi.Host{}
	}
	tempA[accountHash][hostHash] = v
	as.Hosts.Store(tempA)
}
func (as *accountSpace) hostsSync() {
	wg.Add(1)
	for {
		tempStartTimeStamp := time.Now()
		for _, account := range as.getAccounts() {
			// Get cert list
			as.log.Printf("Begin parse hosts for account %s\n", account)
			if account.AccountStatus == "DELETED" {
				continue
			}
			hosts, e := as.API.GetHosts(account.AccountHash)
			if e != nil {
				as.log.Println(e.Error())
				continue
			}

			as.setHostsByKey(account.AccountHash, hosts.Hosts())

			origins, e := as.API.GetOrigins(account.AccountHash)
			if e != nil {
				as.log.Println(e.Error())
				continue
			}
			// create new origins list with origins.ID as key
			om := originsMap{}
			for _, o := range origins.List {
				om[fmt.Sprintf("%d", o.Id)] = o
			}

			// Parse deleted hosts
			esHosts, e := searchDoc(
				es.Search.WithIndex(configureIndex),
				es.Search.WithIgnoreUnavailable(true),
				es.Search.WithAllowPartialSearchResults(false),
				es.Search.WithBody(strings.NewReader(`{"query":{"bool":{"must":[{"match":{"account_hash":"`+account.AccountHash+`"}},{"bool":{"should":[{"range":{"scope.deletedDate":{"gte":"now"}}},{"bool":{"must_not":[{"exists":{"field":"scope.deletedDate"}}]}}]}}]}}}`)),
				es.Search.WithSource([]string{"scope.*", "originPullHost.*", "host_hash"}...),
				es.Search.WithSort("scope.updatedDate:desc"),
				es.Search.WithSize(10000),
			)
			if e != nil {
				as.log.Printf("Search hosts list in es failed, %s", e.Error())
				continue
			}
			// Check updated/newlyAdded
			uniqueESHosts := uniqueHostsFromES(esHosts)

			hcList := []string{}
			for _, h := range hosts.List {
				hcList = append(hcList, h.HashCode)
				for _, s := range h.Scopes {
					// Get configure for this host
					if s.Platform == "ALL" {
						continue
					}

					updateNeeded := false

					// check whether contains new configure
					if !(uniqueESHosts[fmt.Sprintf("%s-%d", h.HashCode, s.ID)] != nil &&
						uniqueESHosts[fmt.Sprintf("%s-%d", h.HashCode, s.ID)].Scope.UpdatedDate == s.UpdatedDate &&
						om.originComparison(uniqueESHosts[fmt.Sprintf("%s-%d", h.HashCode, s.ID)])) {
						updateNeeded = true
					}

					if updateNeeded {
						// Get current scopes's configureation
						if cc, ee := as.API.GetConfiguration(account.AccountHash, h.HashCode, s.ID); ee != nil {
							log.Printf("Get configuration for host %s scopeID %d failed, %s", h.HashCode, s.ID, ee.Error())
						} else {
							if !(cc.OriginPullHost == nil || cc.OriginPullHost.Primary == nil) {
								cc.OriginPullHost.Primary = om[fmt.Sprintf("%0.f", cc.OriginPullHost.Primary.(float64))]
							}
							if !(cc.OriginPullHost == nil || cc.OriginPullHost.Secondary == nil) {
								cc.OriginPullHost.Secondary = om[fmt.Sprintf("%0.f", cc.OriginPullHost.Secondary.(float64))]
							}
							if cc.Scope.CreatedDate == "" {
								cc.Scope.CreatedDate = "2020-01-01 01:01:01"
							}
							if cc.Scope.UpdatedDate == "" {
								cc.Scope.UpdatedDate = "2020-01-01 01:01:01"
							}
							if _, r := indexDoc(configureIndex, fmt.Sprintf("%s-%d-%s", h.HashCode, s.ID, cc.Scope.UpdatedDate), &customConfiguration{
								cc,
								account.AccountHash,
								h.HashCode,
							}); r != nil {
								as.log.Printf("Update host configure failed %s", r.Error())
							}
						}
					}
				}
			}

			// check deleted Host
			for _, cc := range uniqueESHosts {
				if !inSlice(hcList, cc.HostHash) {
					// maybe deleted, update configure in ES
					// updateDoc(conf)
					if _, r := updateDoc(configureIndex, `{"query":{"match":{"host_hash":"`+cc.HostHash+`"}},"conflicts":"proceed","script":{"source":"ctx._source.scope.deletedDate=\"`+time.Now().UTC().Format(ISO8601)+`\""}}`); r != nil {
						as.log.Printf("Update host configure failed %s", r.Error())
					}
				}
			}

		}

		spent := time.Since(tempStartTimeStamp).Seconds()
		as.log.Printf("Last hosts list sync spent %f seconds, next round will start after %f seconds", spent, 20*60-spent)
		if len(as.getAccounts()) <= 1 {
			time.Sleep(time.Second * 10)
		} else {
			time.Sleep(time.Duration(20*60-spent) * time.Second)
		}
	}
}

func uniqueHostsFromES(hosts *esSearchResponse) map[string]*simpleESHostsResponse {
	res := map[string]*simpleESHostsResponse{}
	for _, h := range hosts.Hits.Hits {
		// try convert map[string]interface{} to simpleESHostsResponse
		v := &simpleESHostsResponse{}
		s, _ := json.Marshal(h.Source)
		json.Unmarshal(s, v)
		if res[fmt.Sprintf("%s-%d", v.HostHash, v.Scope.ID)] == nil {
			res[fmt.Sprintf("%s-%d", v.HostHash, v.Scope.ID)] = v
		}
	}
	return res
}

func searchOriginsByID(originsList *hwapi.OriginList, oID string) *hwapi.Origin {
	for _, o := range originsList.List {
		if fmt.Sprintf("%d", o.Id) == oID {
			return o
		}
	}
	return nil
}
