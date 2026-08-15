package runtime

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"bitshinbasic/internal/netenet"
	"bitshinbasic/internal/value"
)

const netPrefix = "~MB|"

type netPeerInfo struct {
	id    int
	name  string
	ready bool
	rtt   float64
}

type netSess struct {
	isHost    bool
	localID   int
	name      string
	room      string
	ready     bool
	peers     map[int]*netPeerInfo
	nextPID   int
	pingSeq   int
	lastPing  time.Time
	jsonQ     []string
	jsonPeer  int
	lastJSON  string
	replicate map[int]bool
	tick      int
	rpcs      map[string]bool
}

func newNetSess() *netSess {
	return &netSess{
		localID:   -1,
		name:      "Player",
		peers:     map[int]*netPeerInfo{},
		nextPID:   1,
		replicate: map[int]bool{},
		rpcs:      map[string]bool{},
	}
}

func (w *World) netUpdate() {
	if w.sess == nil {
		w.sess = newNetSess()
	}
	if w.net == nil {
		return
	}
	if w.scriptOwnsNet() {
		return
	}
	for _, ev := range w.net.Service() {
		w.routeNet(ev)
	}
	if time.Since(w.sess.lastPing) > 2*time.Second {
		w.sess.lastPing = time.Now()
		w.sess.pingSeq++
		w.netBroadcastProto("PING|" + strconv.Itoa(w.sess.pingSeq) + "|" + strconv.FormatInt(time.Now().UnixMilli(), 10))
	}
	w.sess.tick++
	w.netSendReplicas()
}

func (w *World) routeNet(ev netenet.Event) {
	switch ev.Kind {
	case netenet.EventConnect:
		w.ensurePeer(ev.Peer)
		if w.sess.isHost {
			w.netSendProto(ev.Peer, "WELCOME|"+strconv.Itoa(ev.Peer)+"|"+w.sess.name+"|"+w.sess.room)
			if w.sess.name != "" {
				w.netSendProto(ev.Peer, "NAME|"+w.sess.name)
			}
			if w.sess.room != "" {
				w.netSendProto(ev.Peer, "ROOM|"+w.sess.room)
			}
		}
		w.netInbox = append(w.netInbox, ev)
	case netenet.EventDisconnect:
		delete(w.sess.peers, ev.Peer)
		w.netInbox = append(w.netInbox, ev)
	case netenet.EventReceive:
		if w.handleProto(ev) {
			return
		}
		w.netInbox = append(w.netInbox, ev)
	}
}

func (w *World) handleProto(ev netenet.Event) bool {
	msg := ev.Message
	if !strings.HasPrefix(msg, netPrefix) {
		return false
	}
	body := strings.TrimPrefix(msg, netPrefix)
	kind, rest, _ := strings.Cut(body, "|")
	p := w.ensurePeer(ev.Peer)
	switch strings.ToUpper(kind) {
	case "WELCOME":
		parts := strings.SplitN(rest, "|", 3)
		if len(parts) > 0 {
			if id, err := strconv.Atoi(parts[0]); err == nil {
				w.sess.localID = id
			}
		}
		if len(parts) > 1 {
			p.name = parts[1]
		}
		if len(parts) > 2 {
			w.sess.room = parts[2]
		}
	case "NAME":
		p.name = rest
	case "READY":
		p.ready = rest == "1" || strings.EqualFold(rest, "true")
	case "ROOM":
		w.sess.room = rest
	case "PING":
		w.netSendProto(ev.Peer, "PONG|"+rest)
	case "PONG":
		parts := strings.Split(rest, "|")
		if len(parts) >= 2 {
			if sent, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err == nil {
				p.rtt = float64(time.Now().UnixMilli() - sent)
			}
		}
	case "JSON":
		w.sess.jsonQ = append(w.sess.jsonQ, rest)
		w.sess.jsonPeer = ev.Peer
		w.sess.lastJSON = rest
	case "SNAP":
		_ = w.netApplySnapshot(rest)
	case "RPC", "CALL":
		name, arg, _ := strings.Cut(rest, "|")
		w.netInvoke(name, arg, ev.Peer)
	default:
		return false
	}
	return true
}

func (w *World) ensurePeer(id int) *netPeerInfo {
	if w.sess.peers[id] == nil {
		w.sess.peers[id] = &netPeerInfo{id: id}
	}
	return w.sess.peers[id]
}

func (w *World) netSendProto(peer int, body string) {
	if w.net == nil {
		return
	}
	_ = w.net.Send(peer, netPrefix+body, true)
}

func (w *World) netBroadcastProto(body string) {
	if w.net == nil {
		return
	}
	_ = w.net.Broadcast(netPrefix+body, true)
}

func (w *World) netInvoke(name, arg string, peer int) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" || !w.sess.rpcs[key] || w.runner == nil {
		return
	}
	_, _ = w.runner.CallNamed(key, []value.Value{value.Str(arg), value.Num(float64(peer))})
}

type snapEnt struct {
	ID int     `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	Z  float64 `json:"z"`
	P  float64 `json:"p"`
	Ya float64 `json:"ya"`
	R  float64 `json:"r"`
}

type snapMsg struct {
	T int       `json:"t"`
	E []snapEnt `json:"e"`
}

func (w *World) netSnapshot() string {
	msg := snapMsg{T: w.sess.tick}
	for id := range w.sess.replicate {
		e := w.ents[id]
		if e == nil || e.node == nil {
			if s := w.sprites[id]; s != nil {
				msg.E = append(msg.E, snapEnt{ID: id, X: s.x, Y: s.y})
			}
			continue
		}
		pos := worldPos(e.node.GetNode())
		x, y, z := fromG3N(pos.X, pos.Y, pos.Z)
		msg.E = append(msg.E, snapEnt{
			ID: id, X: float64(x), Y: float64(y), Z: float64(z),
			P: float64(e.pitch), Ya: float64(e.yaw), R: float64(e.roll),
		})
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (w *World) netSendReplicas() {
	if w.net == nil || len(w.sess.replicate) == 0 {
		return
	}
	_ = w.net.Broadcast(netPrefix+"SNAP|"+w.netSnapshot(), false)
}

func (w *World) netApplySnapshot(s string) error {
	var msg snapMsg
	if err := json.Unmarshal([]byte(s), &msg); err != nil {
		return err
	}
	if msg.T > w.sess.tick {
		w.sess.tick = msg.T
	}
	for _, se := range msg.E {
		if e := w.ents[se.ID]; e != nil && e.node != nil {
			gx, gy, gz := toG3N(float32(se.X), float32(se.Y), float32(se.Z))
			e.node.GetNode().SetPosition(gx, gy, gz)
			e.pitch, e.yaw, e.roll = float32(se.P), float32(se.Ya), float32(se.R)
			w.applyRot(e)
			continue
		}
		if sp := w.sprites[se.ID]; sp != nil {
			sp.x, sp.y = se.X, se.Y
		}
	}
	return nil
}

func (w *World) netAllReady() bool {
	if !w.sess.ready {
		return false
	}
	if w.net == nil {
		return w.sess.ready
	}
	ids := w.net.PeerIDs()
	if len(ids) == 0 {
		return w.sess.ready
	}
	for _, id := range ids {
		p := w.sess.peers[id]
		if p == nil || !p.ready {
			return false
		}
	}
	return true
}

func (w *World) netRTT(peer int) float64 {
	if peer <= 0 {
		for _, p := range w.sess.peers {
			if p.rtt > 0 {
				return p.rtt
			}
		}
		return 0
	}
	if p := w.sess.peers[peer]; p != nil {
		return p.rtt
	}
	return 0
}
