package operator

import "time"

const (
	integrationMode = true // set this to false in production mode

	keyFile = "/data/key.txt"

	minNodeCount = 2

	sigCacheMaxCount   = 10000
	sigCacheExpiration = 2 * time.Hour

	getSigHashesInterval = 5 * time.Second
	checkNodesInterval   = 1 * time.Hour
	newNodesDelayTime    = 6 * time.Hour

	attestationProviderURL = "https://shareduks.uks.attest.azure.net"
)
