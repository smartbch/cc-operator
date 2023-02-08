//go:build !sgx

package utils

import (
	"errors"

	"github.com/edgelesssys/ego/attestation"
)

func VerifyRemoteReport(reportBytes []byte) (report attestation.Report, err error) {
	err = errors.New("no build flag: sgx")
	return
}
