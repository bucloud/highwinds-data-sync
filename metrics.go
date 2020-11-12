package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bucloud/hwapi"
)

var (
	storeMetrics map[string][]string = map[string][]string{
		"transfer": []string{
			"xferUsedTotalMB",
			"xferAttemptedTotalMB",
			"durationTotal",
			"requestsCountTotal",
			"rps",
			"lastUpdatedTime",
			"xferRateMbps",
			"userXferRateMbps",
			"completionRatio",
			"responseSizeMeanMB",
		},
		"status": []string{
			"rps",
			"requestsCountTotal",
		},
		"storage": []string{
			"edgeStorageTotalB",
			"edgeFileCountTotal",
			"edgeFileSizeMeanB",
		},
	}
)

func (as *accountSpace) addMetricsJob(job *metricQuery) {
	added := false
	jp := as.JobPool.Load().([]*metricQuery)
	for _, j := range jp {
		if j.Host.HashCode == job.Host.HashCode {
			as.log.Printf("delayed job detected, update origin job created at %s update enddate from %s to %s for %s", j.CreateDate.Format(ISO8601), j.EndDate.Format(ISO8601), job.EndDate.Format(ISO8601), j.Host.Name)
			j.EndDate = job.EndDate
			added = true
		}
	}
	if !added {
		jp = append(jp, job)
		as.JobPool.Store(jp)
	}
}

func (as *accountSpace) consumeJob() *metricQuery {
	j := as.JobPool.Load().([]*metricQuery)
	if len(j) == 1 {
		as.JobPool.Store([]*metricQuery{})
	} else {
		as.JobPool.Store(j[1:])
	}
	return j[0]
}

// Run host transfer test
func (as *accountSpace) transferDataOfAccount(a string) []string {
	as.log.Printf("Run transfer test for account %s", a)
	h := []string{}
	dataTest, err := as.API.GetAnalytics("transfer", a, map[string]string{
		"endDate":   time.Now().UTC().Add(-time.Minute * 10).Format(ISO8601),
		"startDate": time.Now().UTC().Add(-time.Hour * 2).Format(ISO8601),
		"platforms": "DELIVERY",
		"groupBy":   "host",
	})
	if err != nil {
		as.log.Println(err.Error())
		return h
	}
	if len(dataTest.Series) == 0 {
		as.log.Printf("account %s doesn't contain any alive hosts", a)
	}
	for _, d := range dataTest.Series {
		if len(d.Data) != 0 {
			h = append(h, d.Key)
		}
	}
	return h
}

func (as *accountSpace) getDataWorker(q <-chan metricQuery) {
	// get transfer data
	for j := range q {
		as.log.Println("Receive data sync job, begin sync data for " + j.Account + " -> " + j.SubAccount.AccountHash + " : " + j.Host.Name + " with startDate " + j.StartDate.Format(ISO8601))
		for _, platform := range []string{"CDS", "SDS", "CDD", "SDD", "CDI", "SDI"} {
			var tt string
			var ignorePOPs []string
			switch platform {
			case "CDD", "SDD":
				tt = "SHIELDING"
			case "CDI", "SDI":
				tt = "INGEST"
			default:
				tt = "DELIVERY"
			}
			for dataType, cl := range storeMetrics {
				// storage only accept CDS,SDS as groupby
				// status accept statusCode as groupby and prefer use that instead of POPCode when parse statusCode
				// run data fetch directly when dataType == transfer
				// create fake loop
				g := &dataClassify{
					GroupBy: "POP",
					POPs: []*hwapi.POP{&hwapi.POP{
						Code: "",
					}},
					Granularity: "PT5M",
				}
				switch dataType {
				case "status":
					g.GroupBy = "status"
					g.POPs = popList.List
				case "storage":
					g.Granularity = ""
				default:
					g.GroupBy = "POP"
					g.Granularity = "PT5M"
				}
				for _, pop := range g.POPs {
					if g.GroupBy != "POP" && !inSlice(ignorePOPs, pop.Code) {
						continue
					}
					d, e := as.API.GetAnalytics(dataType, j.SubAccount.AccountHash, map[string]string{
						"endDate":     j.EndDate.Format(ISO8601),
						"startDate":   j.StartDate.Format(ISO8601),
						"platforms":   platform,
						"granularity": g.Granularity,
						"hosts":       j.Host.HashCode,
						"groupBy":     g.GroupBy,
						"pops":        pop.Code,
					})
					if e != nil {
						as.log.Println(e.Error())
					} else {
						dataPoint := []string{}
						popInfo := &hwapi.POP{}
						for _, dd := range d.Series {
							if len(dd.Data) == 0 {
								continue
							}
							// Build data point&run insert data to Elasticsearch

							if strings.ToUpper(dd.Type) == "POP" {
								popInfo = getPOPInfoByPOPCode(dd.Key)
							} else {
								popInfo = pop
							}
							metricIndex := map[string]int{}
							for _, cm := range cl {
								metricIndex[cm] = searchIndexByValue(dd.Metrics, cm)
							}
							if dataType == "transfer" {
								ignorePOPs = append(ignorePOPs, dd.Key)
							}
							for _, dp := range dd.Data {
								tempDP := fmt.Sprintf(`"type":"%s","host_name":"%s","host_hash":"%s","service":"%s","platform":"%s","account_hash":"%s","account_name":"%s","pop_code":"%s","pop_group":"%s","pop_name":"%s","pop_country":"%s","pop_region":"%s","usageTime":%0f`,
									tt,
									j.Host.Name,
									j.Host.HashCode,
									platform,
									"highwinds",
									j.SubAccount.AccountHash,
									j.SubAccount.AccountName,
									popInfo.Code,
									popInfo.Group,
									popInfo.Name,
									popInfo.Country,
									popInfo.Region,
									dp[0])
								if dataType == "status" {
									tempDP += fmt.Sprintf(`,"status_code":%s`, dd.Key)
								}
								for mi, mv := range metricIndex {
									tempDP += fmt.Sprintf(`,"%s":%f`, mi, dp[mv])
								}
								// create bulk index info
								dataPoint = append(dataPoint, fmt.Sprintf(`{"index":{"_index":"%s","_id":"%s"}}`+"\n"+`{%s}`, dataType+indexPrefix+"_"+time.Unix(int64(dp[0]/1000), 0).Format(DateTime), fmt.Sprintf("%f%s%s%s%s", dp[0], j.Host.HashCode, popInfo.Code, platform, dd.Key), tempDP))
							}
						}
						if len(dataPoint) != 0 {
							if _, e := indexMetrics(strings.Join(dataPoint, "\n")+"\r\n", dataType); e != nil {
								as.log.Printf("Index metrics for host %s data %s failed, %s", j.Host.HashCode, dataType, e.Error())
							}
						}
					}
				}
			}
		}
	}
}
