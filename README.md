# hwsync
Sync metrics, configuration into ES

# usage
	go build
	hwsync [options]
	
# available options
	Usage of ./hwsync:
		-bs duration
						Set bucket size for data sync, unit minute (default 2h0m0s)
		-cert
						Enable certificates sync (pull certificates into ES) (default true)
		-certsIndex string
						Set index name which contains certificates info, certificates available in ENV (default "config_certs")
		-config
						Enable configure sync (pull configur into ES) (default true)
		-d duration
						Set offset for data sync, unit minute (default 15s)
		-esHost string
						Set elasticsearch server login username, esHost avaialble in ENV (default "http://localhost:9200")
		-esPWD string
						Set elasticsearch server login password, esPassword available in ENV (default "changeme")
		-esRetires int
						Set elasticsearch maxRetries (default 3)
		-esUser string
						Set elasticsearch server login username, esUserName available in ENV (default "elastic")
		-esWorker int
						Set elasticsearch max workers (default 2)
		-eslog
						Enable elasticsearch log
		-gc duration
						Set gc interval (default 1m0s)
		-hostsIndex string
						Set index name which contains hosts configuration, configureIndex available in ENV (default "config_hosts")
		-idleConnTimeout duration
						Set idle connection timeout (default 30s)
		-indexPrefix string
						Set elasticsearch index prefix, usable when testing, indexPrefix available in ENV
		-k duration
						Set keep alive timeout (default 1m0s)
		-loggerReq
						Enable elasticsearch request body in logger
		-loggerRes
						Enable elasticsearch response body in logger
		-maxConn int
						Set max connection for HWApi (default 20)
		-metrics string
						metrics you want to synchronised, provide metrics type and names, for example transfer:rps,xferRateMbps;status:rps (default "transfer:xferUsedTotalMB,xferAttemptedTotalMB,durationTotal,requestsCountTotal,rps,lastUpdatedTime,xferRateMbps,userXferRateMbps,completionRatio,responseSizeMeanMB;status:rps,requestsCountTotal;storage:edgeStorageTotalB,edgeFileCountTotal,edgeFileSizeMeanB")
		-out string
						Set log output file (default "/dev/stdout")
		-pprof
						Enable PPROF
		-pprofHost string
						Set listen host for PPROF (default "localhost:6060")
		-time string
						Set fixed time, all data sync job would run with startDate fixTime and endDate fixTime+buckets
		-timeout duration
						Set timeout for API connect (default 1m0s)
		-tokens string
						Set tokens list, accounthash must included, for example a1b1c1d1:abcdef123asd;
		-worker int
						Set worker for data sync process (default 20)
						
						
		
Some options could been read from environment variables, #note, options in flags are prefered

- esHost
- esUserName
- esPassword
- indexPrefix
- configureIndex
- certificatesIndex
- storeMetrics
- tokens
