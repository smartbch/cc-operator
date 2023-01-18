package operator

import (
	"time"
)

const (
	keyFile = "/data/key.txt"

	sigCacheMaxCount    = 100000
	sigCacheExpiration  = 24 * time.Hour
	timeCacheMaxCount   = 200000
	timeCacheExpiration = 24 * time.Hour

	getSigHashesInterval = 10 * time.Second
	checkNodesInterval   = 6 * time.Minute
	newNodesDelayTime    = 6 * time.Hour
	clientReqTimeout     = 5 * time.Minute

	redeemPublicityPeriod  = 25  // * 60
	convertPublicityPeriod = 100 // * 60
)
