package runtime

import (
	"strings"
	"testing"
	"time"

	"github.com/g3n/engine/core"

	"moonbasic/internal/netenet"
	"moonbasic/internal/value"
)

func TestWaitingRoomAndInfo(t *testing.T) {
	host := New(".")
	client := New(".")
	if _, err := host.Call("setnetplayername", []value.Value{value.Str("Ada")}); err != nil {
		t.Fatal(err)
	}
	ok, err := host.Call("hostgame", []value.Value{value.Num(18769)})
	if err != nil {
		t.Skip(err)
	}
	if ok.Int() != 1 {
		t.Fatalf("host %v", ok)
	}
	_, _ = host.Call("setnetroom", []value.Value{value.Str("harbor")})
	_, _ = host.Call("setnetinfo", []value.Value{value.Str("map"), value.Str("harbor")})
	if _, err := client.Call("setnetplayername", []value.Value{value.Str("Bo")}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Call("joingame", []value.Value{value.Str("127.0.0.1"), value.Num(18769)}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	sawBo := false
	for time.Now().Before(deadline) {
		_, _ = host.Call("netupdate", nil)
		_, _ = client.Call("netupdate", nil)
		lobby, _ := host.Call("netlobby", nil)
		if strings.Contains(lobby.String(), "Bo") && strings.Contains(lobby.String(), "harbor") {
			sawBo = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !sawBo {
		lobby, _ := host.Call("netlobby", nil)
		t.Fatalf("lobby %q", lobby.String())
	}
	_, _ = host.Call("netready", []value.Value{value.Num(1)})
	_, _ = client.Call("netready", []value.Value{value.Num(1)})
	ready := false
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, _ = host.Call("netupdate", nil)
		_, _ = client.Call("netupdate", nil)
		all, _ := host.Call("netallready", nil)
		if all.Int() == 1 {
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("not all ready")
	}
	started, err := host.Call("startgame", nil)
	if err != nil || started.Int() != 1 {
		t.Fatalf("start %v %v", started, err)
	}
	_, _ = client.Call("setnetinfo", []value.Value{value.Str("score"), value.Str("3")})
	gotScore := false
	gotStart := false
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, _ = host.Call("netupdate", nil)
		_, _ = client.Call("netupdate", nil)
		st, _ := client.Call("netstarted", nil)
		info, _ := host.Call("netinfo", []value.Value{value.Str("score")})
		if st.Int() == 1 {
			gotStart = true
		}
		if info.String() == "3" {
			gotScore = true
		}
		if gotStart && gotScore {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("started %v score %v", gotStart, gotScore)
}

func putNetEnt(w *World, id int) {
	n := core.NewNode()
	w.scene.Add(n)
	w.ents[id] = &Entity{node: n}
}

func moveNetX(w *World, id int, x float32) {
	gx, gy, gz := toG3N(x, 0, 0)
	w.ents[id].node.GetNode().SetPosition(gx, gy, gz)
	w.refreshWorldMatrices()
}

func TestRelayAndSmooth(t *testing.T) {
	host := New(".")
	client := New(".")
	for _, id := range []int{1, 2} {
		putNetEnt(host, id)
		putNetEnt(client, id)
	}
	if _, err := host.Call("hostgame", []value.Value{value.Num(18770)}); err != nil {
		t.Skip(err)
	}
	if _, err := client.Call("joingame", []value.Value{value.Str("127.0.0.1"), value.Num(18770)}); err != nil {
		t.Fatal(err)
	}
	_, _ = host.Call("setnetsmooth", []value.Value{value.Num(0)})
	_, _ = client.Call("setnetsmooth", []value.Value{value.Num(0)})
	_, _ = host.Call("netreplicate", []value.Value{value.Num(1)})
	_, _ = client.Call("netreplicate", []value.Value{value.Num(2)})
	moveNetX(host, 1, 10)
	moveNetX(client, 2, 6)
	deadline := time.Now().Add(2 * time.Second)
	arrived := false
	for time.Now().Before(deadline) {
		_, _ = client.Call("netupdate", nil)
		_, _ = host.Call("netupdate", nil)
		_, _ = client.Call("netupdate", nil)
		ada, _ := client.currentPose(1)
		bo, _ := host.currentPose(2)
		if ada.X > 9 && bo.X > 5 {
			arrived = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !arrived {
		ada, _ := client.currentPose(1)
		bo, _ := host.currentPose(2)
		t.Fatalf("relay ada %v bo %v", ada.X, bo.X)
	}

	_, _ = client.Call("setnetsmooth", []value.Value{value.Num(0)})
	moveNetX(host, 1, 0)
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, _ = host.Call("netupdate", nil)
		_, _ = client.Call("netupdate", nil)
		ada, _ := client.currentPose(1)
		if ada.X < 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, _ = client.Call("setnetsmooth", []value.Value{value.Num(400)})
	moveNetX(host, 1, 10)
	_, _ = host.Call("netupdate", nil)
	deadline = time.Now().Add(2 * time.Second)
	gliding := false
	for time.Now().Before(deadline) {
		_, _ = client.Call("netupdate", nil)
		ada, _ := client.currentPose(1)
		if ada.X >= 5 {
			t.Fatalf("smooth jumped to %v", ada.X)
		}
		if ada.X > 0.2 {
			gliding = true
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if !gliding {
		ada, _ := client.currentPose(1)
		t.Fatalf("did not glide, x=%v", ada.X)
	}
	time.Sleep(450 * time.Millisecond)
	_, _ = client.Call("netupdate", nil)
	end, _ := client.currentPose(1)
	if end.X < 9 {
		t.Fatalf("smooth ended at %v", end.X)
	}
}

func TestFindGames(t *testing.T) {
	host := New(".")
	if _, err := host.Call("setnetplayername", []value.Value{value.Str("Ada")}); err != nil {
		t.Fatal(err)
	}
	if _, err := host.Call("hostgame", []value.Value{value.Num(18772)}); err != nil {
		t.Skip(err)
	}
	if _, err := host.Call("setnetroom", []value.Value{value.Str("harbor")}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var list string
	for time.Now().Before(deadline) {
		v, err := host.Call("findgames", []value.Value{value.Num(500)})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(v.String(), "127.0.0.1|18772|harbor|Ada") {
			list = v.String()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if list == "" {
		v, _ := host.Call("findgames", []value.Value{value.Num(700)})
		t.Fatalf("list %q", v.String())
	}
	if _, err := host.Call("netclose", nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(450 * time.Millisecond)
	v, err := host.Call("findgames", []value.Value{value.Num(600)})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(v.String(), "18772") {
		t.Fatalf("still listed %q", v.String())
	}
}

func TestHostGameJoinSay(t *testing.T) {
	host := New(".")
	client := New(".")
	ok, err := host.Call("hostgame", []value.Value{value.Num(18768)})
	if err != nil {
		t.Skip(err)
	}
	if ok.Int() != 1 {
		t.Fatalf("host %v", ok)
	}
	ok, err = client.Call("joingame", []value.Value{value.Str("127.0.0.1"), value.Num(18768)})
	if err != nil || ok.Int() != 1 {
		t.Fatalf("join %v %v", ok, err)
	}
	if _, err := client.Call("netsay", []value.Value{value.Str("hi")}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := host.Call("netrecv", nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.Int() == 1 {
			kind, _ := host.Call("netevent", nil)
			msg, _ := host.Call("netmsg", nil)
			if kind.Int() == netenet.EventReceive && msg.String() == "hi" {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("host did not hear hi")
}

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
