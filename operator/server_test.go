package operator

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	gethacc "github.com/ethereum/go-ethereum/accounts"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/ccoperator/sbch"
)

func TestHandleCert(t *testing.T) {
	oldCertBytes := certBytes
	certBytes = []byte{0x12, 0x34}
	defer func() { certBytes = oldCertBytes }()

	require.Equal(t, `{"success":true,"result":"0x1234"}`,
		mustCallHandler("/cert"))
}

func TestHandlePubKey(t *testing.T) {
	oldPubkeyBytes := pubKeyBytes
	pubKeyBytes = []byte{0x12, 0x34}
	defer func() { pubKeyBytes = oldPubkeyBytes }()

	require.Equal(t, `{"success":true,"result":"0x1234"}`,
		mustCallHandler("/pubkey"))
}

func TestHandleReport(t *testing.T) {
	require.True(t, integrationTestMode)
	require.Equal(t, `{"success":false,"error":"integrationTestMode!"}`,
		mustCallHandler("/report"))
}

func TestHandleJwtToken(t *testing.T) {
	require.True(t, integrationTestMode)
	require.Equal(t, `{"success":false,"error":"integrationTestMode!"}`,
		mustCallHandler("/jwt"))
}

func TestHandleSig(t *testing.T) {
	for _, path := range []string{"/sig", "/sig?", "/sig?x=123"} {
		require.Equal(t, `{"success":false,"error":"missing query parameter: hash"}`,
			mustCallHandler(path))
	}

	require.NoError(t, sigCache.Set("1234", []byte{0x56, 0x78}))

	for _, path := range []string{"/sig?hash=0x4321", "/sig?hash=4321"} {
		require.Equal(t, `{"success":false,"error":"no signature found"}`,
			mustCallHandler(path))
	}

	for _, path := range []string{"/sig?hash=0x1234", "/sig?hash=1234"} {
		require.Equal(t, `{"success":true,"result":"0x5678"}`,
			mustCallHandler(path))
	}
}

func TestHandleCurrNodes(t *testing.T) {
	_currClusterClient := currClusterClient
	currClusterClient = &sbch.ClusterClient{
		AllNodes: []sbch.NodeInfo{
			{
				ID:      1234,
				PbkHash: [32]byte{0xce, 0x12, 0x34},
				RpcUrl:  "rpc1234",
				Intro:   "node1234",
			},
		},
	}
	defer func() { currClusterClient = _currClusterClient }()

	expected := `{"success":true,"result":[{"id":1234,"pbkHash":"0xce12340000000000000000000000000000000000000000000000000000000000","rpcUrl":"rpc1234","intro":"node1234"}]}`
	require.Equal(t, expected, mustCallHandler("/nodes"))
}

func TestHandleNewNodes(t *testing.T) {
	_newClusterClient := newClusterClient
	newClusterClient = &sbch.ClusterClient{
		AllNodes: []sbch.NodeInfo{
			{
				ID:      2345,
				PbkHash: [32]byte{0xce, 0x23, 0x45},
				RpcUrl:  "rpc2345",
				Intro:   "node2345",
			},
		},
	}
	defer func() { newClusterClient = _newClusterClient }()

	expected := `{"success":true,"result":[{"id":2345,"pbkHash":"0xce23450000000000000000000000000000000000000000000000000000000000","rpcUrl":"rpc2345","intro":"node2345"}]}`
	require.Equal(t, expected, mustCallHandler("/newNodes"))
}

func TestHandleStats(t *testing.T) {
	require.Equal(t, `{"success":true,"result":"ok"}`,
		mustCallHandler("/status"))

	suspended.Store(true)
	defer func() { suspended = atomic.Value{} }()

	require.Equal(t, `{"success":true,"result":"suspended"}`,
		mustCallHandler("/status"))
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
	sig1, _ := crypto.Sign(gethacc.TextHash([]byte(fmt.Sprintf("%d", ts))), key1)
	sig3, _ := crypto.Sign(gethacc.TextHash([]byte(fmt.Sprintf("%d", ts))), key3)

	monitorAddresses = []gethcmn.Address{addr1, addr2}
	defer func() {
		monitorAddresses = nil
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
	data, err := ioutil.ReadAll(res.Body)
	return string(data), err
}

func genKeyAndAddr() (*ecdsa.PrivateKey, gethcmn.Address) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}
