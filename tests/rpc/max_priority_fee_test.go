package rpc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestEth_MaxPriorityFeePerGas checks that eth_maxPriorityFeePerGas is
// discoverable on the public eth namespace (no extra gateway config), returns
// a well-formed hex quantity over HTTP, and that the same call over WebSocket
// yields an identical value.
func TestEth_MaxPriorityFeePerGas(t *testing.T) {
	httpRes := Call(t, "eth_maxPriorityFeePerGas", []interface{}{})
	httpFee := requireHexQuantity(t, httpRes.Result)

	wsRes := callWS(t, "eth_maxPriorityFeePerGas", []interface{}{})
	wsFee := requireHexQuantity(t, wsRes.Result)

	require.Equal(t, 0, httpFee.ToInt().Cmp(wsFee.ToInt()),
		"HTTP and WebSocket eth_maxPriorityFeePerGas differ: http=%s ws=%s",
		httpFee.ToInt(), wsFee.ToInt())
}

func requireHexQuantity(t *testing.T, raw json.RawMessage) *hexutil.Big {
	t.Helper()

	var encoded string
	require.NoError(t, json.Unmarshal(raw, &encoded), "result must be a JSON string, got %s", raw)
	require.True(t, strings.HasPrefix(encoded, "0x"), "hex quantity must be 0x-prefixed, got %q", encoded)
	require.Greater(t, len(encoded), len("0x"), "hex quantity must include digits after 0x, got %q", encoded)

	var q hexutil.Big
	require.NoError(t, json.Unmarshal(raw, &q), "result is not a hex quantity: %s", raw)
	return &q
}

func callWS(t *testing.T, method string, params interface{}) *Response {
	t.Helper()

	req, err := json.Marshal(CreateRequest(method, params))
	require.NoError(t, err)

	url, err := w3.GetWSEndpoint()
	require.NoError(t, err, "websocket endpoint")

	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err, "dial websocket")
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer conn.Close()

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, req), "write websocket request")

	_, msg, err := conn.ReadMessage()
	require.NoError(t, err, "read websocket response")

	rpcRes := new(Response)
	require.NoError(t, json.Unmarshal(msg, rpcRes), "decode websocket JSON-RPC response")
	require.Nil(t, rpcRes.Error, "websocket JSON-RPC error for %s: %+v", method, rpcRes.Error)
	return rpcRes
}
