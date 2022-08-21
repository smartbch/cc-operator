package main

import "time"

const (
	minNodeCount     = 5
	minSameRespCount = 3

	sigCacheMaxCount   = 10000
	sigCacheExpiration = 2 * time.Hour

	getSigHashesInterval = 1 * time.Minute
	checkNodesInterval   = 1 * time.Hour
	newNodesDelayTime    = 6 * time.Hour

	attestationProviderURL = "https://shareduks.uks.attest.azure.net"
)
