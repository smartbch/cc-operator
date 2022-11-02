package operator

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartbch/ccoperator/sbch"
)

func callHandler(path string) (string, error) {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()

	mux := createHttpHandlers()
	mux.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	return string(data), err
}

func TestHandleCert(t *testing.T) {
	oldCertBytes := certBytes
	certBytes = []byte{0x12, 0x34}
	defer func() { certBytes = oldCertBytes }()

	resp, err := callHandler("/cert")
	require.NoError(t, err)
	require.Equal(t, `{"success":true,"result":"0x1234"}`, resp)
}

func TestHandlePubKey(t *testing.T) {
	oldPubkeyBytes := pubKeyBytes
	pubKeyBytes = []byte{0x12, 0x34}
	defer func() { pubKeyBytes = oldPubkeyBytes }()

	resp, err := callHandler("/pubkey")
	require.NoError(t, err)
	require.Equal(t, `{"success":true,"result":"0x1234"}`, resp)
}

func TestHandleReport(t *testing.T) {
	require.True(t, integrationTestMode)
	resp, err := callHandler("/report")
	require.NoError(t, err)
	require.Equal(t, `{"success":false,"error":"integrationTestMode!"}`, resp)
}

func TestHandleJwtToken(t *testing.T) {
	require.True(t, integrationTestMode)
	resp, err := callHandler("/jwt")
	require.NoError(t, err)
	require.Equal(t, `{"success":false,"error":"integrationTestMode!"}`, resp)
}

func TestHandleSig(t *testing.T) {
	for _, path := range []string{"/sig", "/sig?", "/sig?x=123"} {
		resp, err := callHandler(path)
		require.NoError(t, err)
		require.Equal(t, `{"success":false,"error":"missing query parameter: hash"}`, resp)
	}

	require.NoError(t, sigCache.Set("1234", []byte{0x56, 0x78}))

	for _, path := range []string{"/sig?hash=0x4321", "/sig?hash=4321"} {
		resp, err := callHandler(path)
		require.NoError(t, err)
		require.Equal(t, `{"success":false,"error":"no signature"}`, resp)
	}

	for _, path := range []string{"/sig?hash=0x1234", "/sig?hash=1234"} {
		resp, err := callHandler(path)
		require.NoError(t, err)
		require.Equal(t, `{"success":true,"result":"0x5678"}`, resp)
	}
}

func TestHandleCurrNodes(t *testing.T) {
	oldRpcClientsInfo := rpcClientsInfo
	rpcClientsInfo = &sbch.RpcClientsInfo{
		AllNodes: []sbch.NodeInfo{
			{
				ID:      1234,
				PbkHash: [32]byte{0xce, 0x12, 0x34},
				RpcUrl:  "rpc1234",
				Intro:   "node1234",
			},
		},
	}
	defer func() { rpcClientsInfo = oldRpcClientsInfo }()

	resp, err := callHandler("/nodes")
	require.NoError(t, err)

	expected := `{"success":true,"result":[{"id":1234,"pbkHash":"0xce12340000000000000000000000000000000000000000000000000000000000","rpcUrl":"rpc1234","intro":"node1234"}]}`
	require.Equal(t, expected, resp)
}

func TestHandleNewNodes(t *testing.T) {
	oldRpcClientsInfo := newRpcClientsInfo
	newRpcClientsInfo = &sbch.RpcClientsInfo{
		AllNodes: []sbch.NodeInfo{
			{
				ID:      2345,
				PbkHash: [32]byte{0xce, 0x23, 0x45},
				RpcUrl:  "rpc2345",
				Intro:   "node2345",
			},
		},
	}
	defer func() { newRpcClientsInfo = oldRpcClientsInfo }()

	resp, err := callHandler("/newNodes")
	require.NoError(t, err)

	expected := `{"success":true,"result":[{"id":2345,"pbkHash":"0xce23450000000000000000000000000000000000000000000000000000000000","rpcUrl":"rpc2345","intro":"node2345"}]}`
	require.Equal(t, expected, resp)
}
