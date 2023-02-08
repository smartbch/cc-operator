//go:build sgx

package utils

import (
	"github.com/edgelesssys/ego/eclient"
)

func VerifyRemoteReport(reportBytes []byte) (attestation.Report, error) {
	return eclient.VerifyRemoteReport(reportBytes)
}
