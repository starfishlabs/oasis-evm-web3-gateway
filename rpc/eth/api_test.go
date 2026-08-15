package eth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/oasisprotocol/oasis-core/go/common/logging"
	"github.com/oasisprotocol/oasis-core/go/common/quantity"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/client"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"
)

func TestRejectOverlong(t *testing.T) {
	var b hexutil.Big
	err := json.Unmarshal([]byte("\"0xf1111111122222222333333334444444455555555666666667777777788888888\""), &b)
	if err != nil {
		t.Logf("overlong unmarshal failed (expected): %v\n", err)
	} else {
		t.Errorf("overlong unmarshal didn't reject")
	}
}

func testAPI() *PublicAPI {
	return &PublicAPI{
		ctx:    context.Background(),
		Logger: logging.GetLogger("eth_rpc_test"),
	}
}

func withMinGasPriceQuery(t *testing.T, fn func(context.Context, client.RuntimeClient) (map[types.Denomination]types.Quantity, error)) {
	t.Helper()
	orig := queryMinGasPrice
	queryMinGasPrice = fn
	t.Cleanup(func() { queryMinGasPrice = orig })
}

func TestMaxPriorityFeePerGasMatchesGasPrice(t *testing.T) {
	const want uint64 = 100
	withMinGasPriceQuery(t, func(context.Context, client.RuntimeClient) (map[types.Denomination]types.Quantity, error) {
		return map[types.Denomination]types.Quantity{
			types.NativeDenomination: *quantity.NewFromUint64(want),
		}, nil
	})

	api := testAPI()
	gasPrice, err := api.GasPrice()
	if err != nil {
		t.Fatalf("GasPrice: %v", err)
	}
	tip, err := api.MaxPriorityFeePerGas()
	if err != nil {
		t.Fatalf("MaxPriorityFeePerGas: %v", err)
	}
	if gasPrice.ToInt().Cmp(tip.ToInt()) != 0 {
		t.Fatalf("MaxPriorityFeePerGas %s != GasPrice %s", tip.ToInt(), gasPrice.ToInt())
	}
	if gasPrice.ToInt().Uint64() != want {
		t.Fatalf("GasPrice = %s, want %d", gasPrice.ToInt(), want)
	}
}

func TestMaxPriorityFeePerGasRuntimeFailure(t *testing.T) {
	withMinGasPriceQuery(t, func(context.Context, client.RuntimeClient) (map[types.Denomination]types.Quantity, error) {
		return nil, errors.New("runtime unavailable")
	})

	api := testAPI()
	got, err := api.MaxPriorityFeePerGas()
	if !errors.Is(err, ErrInternalError) {
		t.Fatalf("MaxPriorityFeePerGas error = %v, want ErrInternalError", err)
	}
	if got != nil {
		t.Fatalf("MaxPriorityFeePerGas value = %v, want nil on runtime failure", got)
	}

	// The same failure must also surface from eth_gasPrice — never a zero/fabricated fee.
	got, err = api.GasPrice()
	if !errors.Is(err, ErrInternalError) {
		t.Fatalf("GasPrice error = %v, want ErrInternalError", err)
	}
	if got != nil {
		t.Fatalf("GasPrice value = %v, want nil on runtime failure", got)
	}
}

func TestMaxPriorityFeePerGasHexQuantityEncoding(t *testing.T) {
	withMinGasPriceQuery(t, func(context.Context, client.RuntimeClient) (map[types.Denomination]types.Quantity, error) {
		return map[types.Denomination]types.Quantity{
			types.NativeDenomination: *quantity.NewFromUint64(0xff),
		}, nil
	})

	api := testAPI()
	got, err := api.MaxPriorityFeePerGas()
	if err != nil {
		t.Fatalf("MaxPriorityFeePerGas: %v", err)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	const want = `"0xff"`
	if string(raw) != want {
		t.Fatalf("hex-quantity encoding = %s, want %s", raw, want)
	}

	var decoded hexutil.Big
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("result is not a hex quantity: %v (%s)", err, raw)
	}
	if decoded.ToInt().Uint64() != 0xff {
		t.Fatalf("decoded hex quantity = %s, want 255", decoded.ToInt())
	}
}
