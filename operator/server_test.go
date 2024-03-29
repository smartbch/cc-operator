package operator

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	gethacc "github.com/ethereum/go-ethereum/accounts"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/cc-operator/sbch"
	"github.com/smartbch/cc-operator/utils"
)

func TestInit(t *testing.T) {
	sbchClient := &sbchRpcClient{}
	signer = newSigner(nil, sbchClient)
}

func TestHandleCert(t *testing.T) {
	oldCertBytes := certBytes
	certBytes = []byte{0x12, 0x34}
	defer func() { certBytes = oldCertBytes }()

	require.Equal(t, `{"success":true,"result":"0x1234"}`,
		mustCallHandler("/cert"))
}

func TestHandleCertReport(t *testing.T) {
	require.True(t, !sgxMode)
	require.Equal(t, `{"success":false,"error":"non-SGX mode"}`,
		mustCallHandler("/cert-report"))
}

func TestHandlePubKey(t *testing.T) {
	oldPubkeyBytes := pubKeyBytes
	pubKeyBytes = []byte{0x12, 0x34}
	defer func() { pubKeyBytes = oldPubkeyBytes }()

	require.Equal(t, `{"success":true,"result":"0x1234"}`,
		mustCallHandler("/pubkey"))
}

func TestHandlePubkeyReport(t *testing.T) {
	require.True(t, !sgxMode)
	require.Equal(t, `{"success":false,"error":"non-SGX mode"}`,
		mustCallHandler("/pubkey-report"))
}

func TestHandleJwtToken(t *testing.T) {
	require.True(t, !sgxMode)
	require.Equal(t, `{"success":false,"error":"non-SGX mode"}`,
		mustCallHandler("/pubkey-jwt"))
}

func TestHandleSig(t *testing.T) {
	for _, path := range []string{"/sig", "/sig?", "/sig?x=123"} {
		require.Equal(t, `{"success":false,"error":"missing query parameter: hash"}`,
			mustCallHandler(path))
	}

	require.NoError(t, signer.sigCache.Set("1234", []byte{0x56, 0x78}))
	_ = signer.timeCache.Set("1234", utils.GetTimestampFromTSC()-10)

	for _, path := range []string{"/sig?hash=0x4321", "/sig?hash=4321"} {
		require.Equal(t, `{"success":false,"error":"no signature found:Key not found."}`,
			mustCallHandler(path))
	}

	for _, path := range []string{"/sig?hash=0x1234", "/sig?hash=1234"} {
		require.Equal(t, `{"success":true,"result":"0x5678"}`,
			mustCallHandler(path))
	}
}

func TestHandleCurrNodes(t *testing.T) {
	_currClusterClient := signer.sbchClient.currClusterClient
	signer.sbchClient.currClusterClient = &sbch.ClusterClient{
		PublicNodes: []sbch.NodeInfo{
			{
				ID:      1234,
				PbkHash: [32]byte{0xce, 0x12, 0x34},
				RpcUrl:  "rpc1234",
				Intro:   "node1234",
			},
		},
	}
	defer func() { signer.sbchClient.currClusterClient = _currClusterClient }()

	expected := `{"success":true,"result":{"status":"ok","currNodes":[{"id":1234,"pbkHash":"0xce12340000000000000000000000000000000000000000000000000000000000","rpcUrl":"rpc1234","intro":"node1234"}]}}`
	require.Equal(t, expected, mustCallHandler("/info"))
}

func TestHandleNewNodes(t *testing.T) {
	_currClusterClient := signer.sbchClient.currClusterClient
	signer.sbchClient.currClusterClient = &sbch.ClusterClient{
		PublicNodes: []sbch.NodeInfo{
			{
				ID:      1234,
				PbkHash: [32]byte{0xce, 0x12, 0x34},
				RpcUrl:  "rpc1234",
				Intro:   "node1234",
			},
		},
	}
	_newClusterClient := signer.sbchClient.newClusterClient
	signer.sbchClient.nodesChangedTime = time.Unix(1671681687, 0)
	signer.sbchClient.newClusterClient = &sbch.ClusterClient{
		PublicNodes: []sbch.NodeInfo{
			{
				ID:      2345,
				PbkHash: [32]byte{0xce, 0x23, 0x45},
				RpcUrl:  "rpc2345",
				Intro:   "node2345",
			},
		},
	}
	defer func() {
		signer.sbchClient.currClusterClient = _currClusterClient
		signer.sbchClient.newClusterClient = _newClusterClient
	}()

	expected := `{"success":true,"result":{"status":"ok","currNodes":[{"id":1234,"pbkHash":"0xce12340000000000000000000000000000000000000000000000000000000000","rpcUrl":"rpc1234","intro":"node1234"}],"newNodes":[{"id":2345,"pbkHash":"0xce23450000000000000000000000000000000000000000000000000000000000","rpcUrl":"rpc2345","intro":"node2345"}],"nodesChangedTime":1671681687}}`
	require.Equal(t, expected, mustCallHandler("/info"))
}

func TestHandleStats(t *testing.T) {
	require.Equal(t, `{"success":true,"result":{"status":"ok"}}`,
		mustCallHandler("/info"))

	suspended.Store(true)
	defer func() { suspended = atomic.Value{} }()

	require.Equal(t, `{"success":true,"result":{"status":"suspended"}}`,
		mustCallHandler("/info"))
	require.Equal(t, `{"success":false,"error":"suspended"}`,
		mustCallHandler("/sig?hash=1234"))
}

func TestHandleSuspend(t *testing.T) {
	require.Equal(t, `{"success":false,"error":"missing query parameter: sig"}`,
		mustCallHandler("/suspend"))
	require.Equal(t, `{"success":false,"error":"missing query parameter: ts"}`,
		mustCallHandler("/suspend?sig=1234"))
	require.Equal(t, `{"success":false,"error":"ts too old"}`,
		mustCallHandler(fmt.Sprintf("/suspend?sig=1234&ts=%d", time.Now().Unix()-suspendTsDiffMaxSeconds*2)))
	require.Equal(t, `{"success":false,"error":"ts too new"}`,
		mustCallHandler(fmt.Sprintf("/suspend?sig=1234&ts=%d", time.Now().Unix()+suspendTsDiffMaxSeconds*2)))
	require.Equal(t, `{"success":false,"error":"invalid signature length"}`,
		mustCallHandler(fmt.Sprintf("/suspend?sig=1234&ts=%d", time.Now().Unix()+suspendTsDiffMaxSeconds)))

	key1, addr1 := genKeyAndAddr()
	_, addr2 := genKeyAndAddr()
	key3, _ := genKeyAndAddr()

	ts := time.Now().Unix()
	pk := "0x" + hex.EncodeToString(pubKeyBytes)
	sig1, _ := crypto.Sign(gethacc.TextHash([]byte(fmt.Sprintf("%s,%d", pk, ts))), key1)
	sig3, _ := crypto.Sign(gethacc.TextHash([]byte(fmt.Sprintf("%s,%d", pk, ts))), key3)

	signer.sbchClient.allMonitorMap = map[gethcmn.Address]bool{
		addr1: true,
		addr2: true,
	}
	defer func() {
		signer.sbchClient.allMonitorMap = map[gethcmn.Address]bool{}
		suspended = atomic.Value{}
	}()

	require.Equal(t, `{"success":false,"error":"not monitor"}`,
		mustCallHandler(fmt.Sprintf("/suspend?sig=%s&ts=%d", hex.EncodeToString(sig3), ts)))

	// ok
	require.Equal(t, `{"success":true,"result":"ok"}`,
		mustCallHandler(fmt.Sprintf("/suspend?sig=%s&ts=%d", hex.EncodeToString(sig1), ts)))
	require.True(t, suspended.Load().(bool))
}

func mustCallHandler(path string) string {
	resp, err := callHandler(path)
	if err != nil {
		panic(err)
	}
	return resp
}
func callHandler(path string) (string, error) {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()

	mux := createHttpHandlers()
	mux.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	return string(data), err
}

func genKeyAndAddr() (*ecdsa.PrivateKey, gethcmn.Address) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}
