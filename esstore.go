package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/elastic/go-elasticsearch/v7/estransport"
)

type metricStruct struct {
	Type                      string  `json:"type,omitempty"` // DELIVERY, SHIELDING, INGEST
	HostName                  string  `json:"host_name,omitempty"`
	HostHash                  string  `json:"host_hash,omitempty"`
	Service                   string  `json:"service,omitempty"`
	Platform                  string  `json:"platform,omitempty"`
	AccountHash               string  `json:"account_hash,omitempty"`
	AccountName               string  `json:"account_name,omitempty"`
	StatusCode                string  `json:"status_code,omitempty"`
	POPCode                   string  `json:"pop_code,omitempty"`
	POPGroup                  string  `json:"pop_group,omitempty"`
	POPName                   string  `json:"pop_name,omitempty"`
	POPCountry                string  `json:"pop_country,omitempty"`
	POPRegion                 string  `json:"pop_region,omitempty"`
	TimeStamp                 float64 `json:"timestamp,omitempty"`
	EdgeStorageTotalB         float32 `json:"edgeStorageTotalB,omitempty"`         // total bytes stored on every edge, may contain duplicate assets
	EdgeStorageMaxB           float32 `json:"edgeStorageMaxB,omitempty"`           // maximum five minute bucket of bytes stored
	EdgeStorageMaxUsageTime   float32 `json:"edgeStorageMaxUsageTime,omitempty"`   // timestamp at which maximum bytes stored occurred
	EdgeStorageMinB           float32 `json:"edgeStorageMinB,omitempty"`           // minimum five minute bucket of bytes stored
	EdgeStorageMinUsageTime   float32 `json:"edgeStorageMinUsageTime,omitempty"`   // timestamp at which minimum bytes stored occurred
	EdgeStorageMeanB          float32 `json:"edgeStorageMeanB,omitempty"`          // average of five minute bucket bytes stored
	EdgeFileCountTotal        float32 `json:"edgeFileCountTotal,omitempty"`        // total assets stored on every edge, may contain duplicate assets
	EdgeFileCountMax          float32 `json:"edgeFileCountMax,omitempty"`          // maximum five minute bucket of files stored
	EdgeFileCountMaxUsageTime float32 `json:"edgeFileCountMaxUsageTime,omitempty"` // timestamp at which maximum files stored occurred
	EdgeFileCountMin          float32 `json:"edgeFileCountMin,omitempty"`          // minimum five minute bucket of files stored
	EdgeFileCountMinUsageTime float32 `json:"edgeFileCountMinUsageTime,omitempty"` // timestamp at which minimum files stored occurred
	EdgeFileCountMean         float32 `json:"edgeFileCountMean,omitempty"`         // average of five minute bucket files stored
	EdgeFileSizeMeanB         float32 `json:"edgeFileSizeMeanB,omitempty"`         // average of five minute bucket bytes stored
	LastUpdatedTime           float32 `json:"lastUpdatedTime,omitempty"`           // last time this bucket was updated
	XferUsedTotalMB           float32 `json:"xferUsedTotalMB,omitempty"`           // total MB transferred
	XferUsedMinMB             float32 `json:"xferUsedMinMB,omitempty"`             // minimum five minute bucket of MB transferred
	XferUsedMaxMB             float32 `json:"xferUsedMaxMB,omitempty"`             // maximum five minute bucket of MB transferred
	XferUsedMeanMB            float32 `json:"xferUsedMeanMB,omitempty"`            // average of five minute buckets of MB transferred
	XferAttemptedTotalMB      float32 `json:"xferAttemptedTotalMB,omitempty"`      // total MB attempted
	DurationTotal             float32 `json:"durationTotal,omitempty"`             // total transfer time of all requests
	XferRateMaxMbps           float32 `json:"xferRateMaxMbps,omitempty"`           // maximum transfer rate
	XferRateMaxUsageTime      float32 `json:"xferRateMaxUsageTime,omitempty"`      // timestamp at which maximum transfer rate occurred
	XferRateMinMbps           float32 `json:"xferRateMinMbps,omitempty"`           // minimum transfer rate
	XferRateMinUsageTime      float32 `json:"xferRateMinUsageTime,omitempty"`      // timestamp at which minimum transfer rate occurred
	XferRateMeanMbps          float32 `json:"xferRateMeanMbps,omitempty"`          // average transfer rate
	RequestsCountTotal        float32 `json:"requestsCountTotal,omitempty"`        // total requests
	RequestsCountMin          float32 `json:"requestsCountMin,omitempty"`          // minimum five minute bucket of requests per second
	RequestsCountMax          float32 `json:"requestsCountMax,omitempty"`          // maximum five minute bucket of requests per second
	RequestsCountMean         float32 `json:"requestsCountMean,omitempty"`         // average of five minute bucket requests per second
	RpsMax                    float32 `json:"rpsMax,omitempty"`                    // maximum requests per second
	RpsMaxUsageTime           float32 `json:"rpsMaxUsageTime,omitempty"`           // timestamp at which maximum requests per second occurred
	RpsMin                    float32 `json:"rpsMin,omitempty"`                    // minimum requests per second
	RpsMinUsageTime           float32 `json:"rpsMinUsageTime,omitempty"`           // timestamp at which minimum requests per second occurred
	RpsMean                   float32 `json:"rpsMean,omitempty"`                   // mean requests per second
	XferRateMbps              float32 `json:"xferRateMbps,omitempty"`              // average transfer rate in Mbps
	UserXferRateMbps          float32 `json:"userXferRateMbps,omitempty"`          // total transfer divided by duration
	Rps                       float32 `json:"rps,omitempty"`                       // requests per second, calculated as total requests divided by number of seconds in bucket
	CompletionRatio           float32 `json:"completionRatio,omitempty"`           // completed requests divided by attempted requests
	ResponseSizeMeanMB        float32 `json:"responseSizeMeanMB,omitempty"`        // total MB transferred divided by number of requests
	PeakToMeanMBRatio         float32 `json:"peakToMeanMBRatio,omitempty"`         // maximum transfer rate divided by mean transfer rate
	PeakToMeanRequestsRatio   float32 `json:"peakToMeanRequestsRatio,omitempty"`   // maximum requests per second divided by mean requests per second

	DataType string `json:"-"` // transfer, status, storage
}

type esIndexResponse struct {
	Index       string `json:"_index,omitempty"`   // Index name
	Type        string `json:"_type,omitempty"`    // Doc type, deprecated es default doctype is doc, but currently use _doc as default in application
	ID          string `json:"_id,omitempty"`      // Doc ID
	Version     int    `json:"_version,omitempty"` // version number, multiple doc version is not available, just use to check whether doc updated
	Result      string `json:"result,omitempty"`   // created, updated ...
	Shards      shards `json:"_shards,omitempty"`
	SeqNo       int    `json:"_seq_no,omitempty"`       // sequence number
	PrimaryTerm int    `json:"_primary_term,omitempty"` // Primary shard number
}

type esSearchResponse struct {
	Took     int    `json:"_took,omitempty"`     // Took time, unit seconds
	Timedout bool   `json:"timed_out,omitempty"` // true if handle request timeedout
	Shards   shards `json:"_shards,omitempty"`   // Search Shards info
	Hits     hits   `json:"hits,omitempty"`      // Search result
}

type hits struct {
	Total    map[string]interface{} `json:"total,omitempty"`     // hits shards info
	MaxScore float32                `json:"max_score,omitempty"` // matched content max_score
	Hits     []*hitsDocs            `json:"hits,omitempty"`      // hits docs
}

type hitsDocs struct {
	Index  string                 `json:"_index,omitempty"`  // hited index name
	Type   string                 `json:"_type,omitempty"`   // Doc type, deprecated
	ID     string                 `json:"_id,omitempty"`     // Doc ID
	Score  float32                `json:"_score,omitempty"`  // matched doc score
	Source map[string]interface{} `json:"_source,omitempty"` // Document
}

// Shards info, used for search response
type shards struct {
	Total      int `json:"total,omitempty"`      // total shards searched
	Successful int `json:"successful,omitempty"` // shards successed
	Skipped    int `json:"skipped,omitempty"`    // shards skipped
	Failed     int `json:"failed,omitempty"`     // shards failed
}

type esErrorResponse struct {
	ESError `json:"error,omitempty"`
}

// ESError common es error response
type ESError struct {
	// sample data
	// 	  "root_cause" : [
	// 		{
	// 		  "type" : "mapper_parsing_exception",
	// 		  "reason" : "failed to parse field [createdDate] of type [date] in document with id 'x8x9y9g5-8073f65b22ad65c5f9480d637c6048837dda420105677b88f300f1e60dbc9f2c'. Preview of field's value: '2020-03-17 07:32:42'"
	// 		}
	// 	  ],
	RootCause []map[string]string `json:"root_cause,omitempty"`

	Type   string `json:"type,omitempty"`
	Reason string `json:"reason,omitempty"`

	// sample data
	// 	  "caused_by" : {
	// 		"type" : "illegal_argument_exception",
	// 		"reason" : "failed to parse date field [2020-03-17 07:32:42] with format [strict_date_optional_time||epoch_millis]",
	// 		"caused_by" : {
	// 		  "type" : "date_time_parse_exception",
	// 		  "reason" : "Failed to parse with all enclosed parsers"
	// 		}
	// 	  }
	CausedBy map[string]interface{} `json:"caused_by,omitempty"` //
	Status   int                    `json:"status,omitempty"`    // status Code
}

// esBulkErrorResponse
type esBulkErrorResponse struct {
	// Time spent
	Took int `json:"took"`

	// errors could been true when anyone of the action failed
	Errors bool `json:"errors"`

	// error info
	// sample data
	// {"index":{"_index":"transfer_20200902","_type":"_doc","_id":"1599003000000.000000f6g4s8v3DC1CDSDC1","status":400,"error":{"type":"mapper_parsing_exception","reason":"failed to parse","caused_by":{"type":"json_parse_exception","reason":"Unexpected character ('\\' (code 92)): expected a valid value (JSON String, Number, Array, Object or token 'null', 'true' or 'false')\n at [Source: (org.elasticsearch.common.bytes.AbstractBytesReference$MarkSupportingStreamInputWrapper); line: 1, column: 88]"}}}}
	Items []map[string]bulkItemErrorDetails
}

type bulkItemErrorDetails struct {
	Index string `json:"_index"`
	Type  string `json:"_type"`
	ID    string `json:"_id"`

	// response code
	Status int `json:"status"`
	Error  ESError
}

const (
// RetryOnConflict enable conflict
// RetryOnConflict *int = 1
)

func initES() (*elasticsearch.Client, error) {
	esConfig := elasticsearch.Config{
		Addresses:     []string{esHost},
		Username:      esUserName,
		Password:      esPassword,
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    esRetries,
	}
	if enableESLog {
		esConfig.Logger = &estransport.TextLogger{
			Output:             logOutPut,
			EnableRequestBody:  esLoggerEnabelRequest,
			EnableResponseBody: esLoggerEnableResponse,
		}
	}
	esClient, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, err
	}
	res, e := esClient.Info()
	if e != nil {
		return nil, e
	}
	if res.IsError() {
		return nil, errors.New(res.String())
	}
	r := map[string]interface{}{}
	if err = json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}
	rlog.Printf("ESInfo %s %s %s %s %s %s %s %s", "clientVersion", elasticsearch.Version, "serverVersion", r["version"].(map[string]interface{})["number"], "clusterName", r["cluster_name"], "clusterUUID", r["cluster_uuid"])

	return esClient, nil
}

func indexMetrics(b string, index string) (bool, error) {
	bi, err := es.Bulk(bytes.NewReader([]byte(b)), es.Bulk.WithDocumentType("_doc"), es.Bulk.WithErrorTrace())
	defer bi.Body.Close()
	if err != nil {
		return false, err
	}

	if bi.IsError() {
		r := esErrorResponse{}
		e := json.NewDecoder(bi.Body).Decode(&r)
		if e != nil {
			return false, fmt.Errorf("[%s] Error index metrics, reason %s, unexpected error %s", bi.Status(), r.Reason, e.Error())
		}
		return false, fmt.Errorf("[%s] Error index metrics, reason %s", bi.Status(), r.Reason)
	}
	var r esIndexResponse
	if err := json.NewDecoder(bi.Body).Decode(&r); err != nil {
		return false, fmt.Errorf("Error parsing the response body: %s", err)
	}
	return true, nil
}

func searchDoc(search ...func(*esapi.SearchRequest)) (*esSearchResponse, error) {
	// Perform the search request.
	res, err := es.Search(search...)
	defer res.Body.Close()

	if err != nil {
		return nil, err
	}

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s %s: %s", res.Status(),
			e["error"].(map[string]interface{})["type"],
			e["error"].(map[string]interface{})["reason"])

	}
	r := &esSearchResponse{}

	if err := json.NewDecoder(res.Body).Decode(r); err != nil {
		return nil, fmt.Errorf("Error parsing the response body: %s", err)
	}
	return r, nil
}

func indexDoc(index string, id string, body interface{}) (bool, error) {
	b, e := json.Marshal(body)

	if e != nil {
		return false, e
	}
	res, err := es.Index(index, bytes.NewReader(b), es.Index.WithDocumentID(id), es.Index.WithRefresh("true"), es.Index.WithDocumentType("_doc"))
	defer res.Body.Close()
	if err != nil {
		return false, err
	}

	if res.IsError() {
		r := esErrorResponse{}
		json.NewDecoder(res.Body).Decode(&r)
		return false, fmt.Errorf("[%s] Error indexing document ID=%s, responseBody %s, unexpected", res.Status(), id, r.Reason)
	}
	var r esIndexResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return false, fmt.Errorf("Error parsing the response body: %s", err)
	}
	return true, nil
}

func updateDoc(index string, body string) (bool, error) {
	res, err := es.UpdateByQuery([]string{index}, es.UpdateByQuery.WithBody(bytes.NewBufferString(body)), es.UpdateByQuery.WithConflicts("proceed"))
	defer res.Body.Close()
	if err != nil {
		return false, err
	}

	if res.IsError() {
		r := esErrorResponse{}
		e := json.NewDecoder(res.Body).Decode(&r)
		if e != nil {
			return false, fmt.Errorf("[%s] Error updating document, reason %s, unexpected error %s", res.Status(), r.Reason, e.Error())
		}
		return false, fmt.Errorf("[%s] Error updating document, reason %s", res.Status(), r.Reason)
	}
	var r esIndexResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return false, fmt.Errorf("Error parsing the response body: %s", err)
	}
	return true, nil
}

func (hc *hits) toString() string {
	b, _ := json.Marshal(hc)
	return string(b)
}
