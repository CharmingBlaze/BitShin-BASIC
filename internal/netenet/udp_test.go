package netenet

import (
	"testing"
	"time"
)

func TestUDPListenDialAndPeerIP(t *testing.T) {
	srv, err := ListenMax(18765, 8)
	if err != nil {
		t.Skip("port 18765 busy")
	}
	defer srv.Close()
	cli, err := Dial("127.0.0.1", 18765)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	if err := cli.Send(1, "ping", true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	var got Event
	for time.Now().Before(deadline) {
		for _, ev := range srv.Service() {
			if ev.Kind == EventReceive {
				got = ev
				break
			}
		}
		if got.Kind == EventReceive {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got.Kind != EventReceive || got.Message != "ping" {
		t.Fatalf("event %+v", got)
	}
	if got.Peer == 0 {
		t.Fatal("peer id")
	}
	if srv.PeerIP(got.Peer) == "" && got.IP == "" {
		t.Fatal("missing peer IP")
	}
	if srv.Backend() != BackendUDP {
		t.Fatalf("backend %q", srv.Backend())
	}
}
