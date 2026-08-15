package runtime

import (
	"testing"
	"time"

	"bitshinbasic/internal/netenet"
	"bitshinbasic/internal/value"
)

func TestCreateNetworkHostRoundtrip(t *testing.T) {
	w := New(".")
	host, err := w.Call("createnetworkhost", []value.Value{value.Num(18767), value.Num(8)})
	if err != nil {
		t.Skip(err)
	}
	cli, err := w.Call("createnetworkclient", []value.Value{value.Str("127.0.0.1"), value.Num(18767)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Call("sendnetwork", []value.Value{cli, value.Num(1), value.Str("hello host")})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	gotRecv := false
	for time.Now().Before(deadline) {
		ev, err := w.Call("pollnetwork", []value.Value{host})
		if err != nil {
			t.Fatal(err)
		}
		typ, _ := ev.Field("type")
		if typ.Number() == float64(netenet.EventReceive) {
			data, _ := ev.Field("data")
			if data.String() != "hello host" {
				t.Fatalf("data %q", data.String())
			}
			peer, _ := ev.Field("peerid")
			if peer.Number() == 0 {
				t.Fatal("PeerID")
			}
			gotRecv = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !gotRecv {
		t.Fatal("no NET_RECEIVE")
	}
	_, _ = w.Call("closenetworkhost", []value.Value{cli})
	_, _ = w.Call("closenetworkhost", []value.Value{host})
}
