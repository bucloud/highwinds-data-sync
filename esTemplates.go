package main

// func checkTemplates() {
// 	templates := map[string]string{
// 		"certs_config_template": `{"mappings":
// 			{
// 				"_doc": {
// 				  "_source": {
// 					"enabled": true
// 				  },
// 				  "dynamic_date_formats" : [
// 					"strict_date_optional_time",
// 					"yyyy/MM/dd HH:mm:ss Z||yyyy/MM/dd Z"
// 				  ],
// 				  "properties": {
// 					"updatedDate": {
// 					  "type": "date",
// 					  "doc_values": true
// 					},
// 					"createdDate": {
// 					  "type": "date",
// 					  "doc_values": true
// 					},
// 					"expirationDate": {
// 					  "type": "date",
// 					  "doc_values": true

// 					}
// 				  }
// 				}
// 			  }
// 		}`,
// 		"status_template": `{"mappings":
// 		{
// 			"_doc": {
// 			  "_source": {
// 				"enabled": false
// 			  },
// 			  "properties": {
// 				"usageTime": {
// 				  "type": "date",
// 				  "doc_values": true
// 				},
// 				"requestsCountTotal": {
// 				  "type": "long",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"rps": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"status_code": {
// 				  "type": "integer"
// 				}
// 			  }
// 			}
// 		  }
// 	}`,
// 		"storage_template": `{"mappings":
// 		{
// 			"_doc": {
// 			  "_source": {
// 				"enabled": false
// 			  },
// 			  "properties": {
// 				"usageTime": {
// 				  "type": "date",
// 				  "doc_values": true
// 				},
// 				"edgeStorageTotalB": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"edgeFileCountTotal": {
// 				  "type": "long",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"edgeFileSizeMeanB": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				}
// 			  }
// 			}
// 		  }
// 	  }
// }`,
// 		"transfer_template": `{"mappings":
// 		{
// 			"_doc": {
// 			  "_source": {
// 				"enabled": false
// 			  },
// 			  "properties": {
// 				"usageTime": {
// 				  "type": "date",
// 				  "doc_values": true
// 				},
// 				"xferUsedTotalMB": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"xferAttemptedTotalMB": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"durationTotal": {
// 				  "type": "long",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"requestsCountTotal": {
// 				  "type": "long",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"lastUpdatedTime": {
// 				  "type": "date",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"xferRateMbps": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"userXferRateMbps": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"rps": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"completionRatio": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				},
// 				"responseSizeMeanMB": {
// 				  "type": "float",
// 				  "doc_values": true,
// 				  "index": false
// 				}
// 			  }
// 			}
// 		  }
// }`,
// 	}
// }
