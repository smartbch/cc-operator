package main

import "time"

const (
	minEnclaveNodeCount = 5
	minSameRespCount    = 3

	sigCacheMaxCount   = 10000
	sigCacheExpiration = 2 * time.Hour

	getSigHashesInterval      = 1 * time.Minute
	checkEnclaveNodesInterval = 1 * time.Hour
	newEnclaveNodesDelayTime  = 6 * time.Hour
)
