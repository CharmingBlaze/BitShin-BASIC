package runtime

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"moonbasic/internal/netenet"
	"moonbasic/internal/value"
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
	started   bool
	info      map[string]string
	rosterRaw string
	smoothMS  int
	blends    map[int]*netBlend
	remote    map[int]snapEnt
}

func newNetSess() *netSess {
	return &netSess{
		localID:   -1,
		name:      "Player",
		peers:     map[int]*netPeerInfo{},
		nextPID:   1,
		replicate: map[int]bool{},
		rpcs:      map[string]bool{},
		smoothMS:  100,
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
	w.netSmooth()
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
			w.publishRoster()
			w.publishInfo()
		}
		w.netInbox = append(w.netInbox, ev)
	case netenet.EventDisconnect:
		delete(w.sess.peers, ev.Peer)
		if w.sess.isHost {
			w.publishRoster()
		}
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
		if w.sess.rosterRaw != "" {
			w.applyRoster(w.sess.rosterRaw)
		}
	case "NAME":
		p.name = rest
		if w.sess.isHost {
			w.publishRoster()
		}
	case "READY":
		p.ready = rest == "1" || strings.EqualFold(rest, "true")
		if w.sess.isHost {
			w.publishRoster()
		}
	case "ROSTER":
		w.applyRoster(rest)
	case "START":
		w.sess.started = true
	case "INFO":
		key, val, _ := strings.Cut(rest, "|")
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			break
		}
		if w.sess.info == nil {
			w.sess.info = map[string]string{}
		}
		w.sess.info[key] = val
		if w.sess.isHost {
			w.netBroadcastProto("INFO|" + key + "|" + val)
		}
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
	case "OWN":
		if w.sess.isHost {
			w.storeRemote(rest)
			_ = w.netQueueSnapshot(rest)
		}
	case "SNAP":
		_ = w.netQueueSnapshot(rest)
	case "RPC", "CALL":
		name, arg, _ := strings.Cut(rest, "|")
		w.netInvoke(name, arg, ev.Peer)
	default:
		return false
	}
	return true
}

func (w *World) ensureSess() *netSess {
	if w.sess == nil {
		w.sess = newNetSess()
	}
	return w.sess
}

func netField(s string) string {
	s = strings.NewReplacer("|", " ", ",", " ", ";", " ", "\n", " ").Replace(s)
	s = strings.TrimSpace(s)
	if s == "" {
		return "Player"
	}
	return s
}

func (w *World) publishRoster() {
	if w.net == nil || w.sess == nil || !w.sess.isHost {
		return
	}
	ready := 0
	if w.sess.ready {
		ready = 1
	}
	parts := []string{fmt.Sprintf("%d,%s,%d", w.sess.localID, netField(w.sess.name), ready)}
	ids := make([]int, 0, len(w.sess.peers))
	for id := range w.sess.peers {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		p := w.sess.peers[id]
		flag := 0
		if p.ready {
			flag = 1
		}
		parts = append(parts, fmt.Sprintf("%d,%s,%d", id, netField(p.name), flag))
	}
	w.netBroadcastProto("ROSTER|" + strings.Join(parts, ";"))
}

func (w *World) applyRoster(body string) {
	w.sess.rosterRaw = body
	next := map[int]*netPeerInfo{}
	if body == "" {
		w.sess.peers = next
		return
	}
	for _, part := range strings.Split(body, ";") {
		f := strings.Split(part, ",")
		if len(f) < 3 {
			continue
		}
		id, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		ready := f[2] == "1"
		if id == w.sess.localID {
			w.sess.ready = ready
			if f[1] != "" {
				w.sess.name = f[1]
			}
			continue
		}
		next[id] = &netPeerInfo{id: id, name: f[1], ready: ready}
	}
	w.sess.peers = next
}

func (w *World) netLobby() string {
	w.netUpdate()
	s := w.ensureSess()
	started := 0
	if s.started {
		started = 1
	}
	selfReady := 0
	if s.ready {
		selfReady = 1
	}
	var b strings.Builder
	fmt.Fprintf(&b, "room %s\nstarted %d\n%d %s %d\n", s.room, started, s.localID, netField(s.name), selfReady)
	ids := make([]int, 0, len(s.peers))
	for id := range s.peers {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		p := s.peers[id]
		flag := 0
		if p.ready {
			flag = 1
		}
		fmt.Fprintf(&b, "%d %s %d\n", id, netField(p.name), flag)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (w *World) startGame() int {
	s := w.ensureSess()
	if w.net != nil && !s.isHost {
		return 0
	}
	s.started = true
	w.netBroadcastProto("START")
	w.publishRoster()
	return 1
}

func (w *World) setNetInfo(key, val string) {
	s := w.ensureSess()
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return
	}
	if w.net != nil && !s.isHost {
		w.netSendProto(1, "INFO|"+key+"|"+val)
		return
	}
	if s.info == nil {
		s.info = map[string]string{}
	}
	s.info[key] = val
	w.netBroadcastProto("INFO|" + key + "|" + val)
}

func (w *World) publishInfo() {
	if w.net == nil || w.sess == nil || !w.sess.isHost {
		return
	}
	keys := make([]string, 0, len(w.sess.info))
	for k := range w.sess.info {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		w.netBroadcastProto("INFO|" + k + "|" + w.sess.info[k])
	}
}

func (w *World) netInfo(key string) string {
	w.netUpdate()
	s := w.ensureSess()
	if s.info == nil {
		return ""
	}
	return s.info[strings.ToLower(strings.TrimSpace(key))]
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
	seen := map[int]bool{}
	for id := range w.sess.replicate {
		if se, ok := w.currentPose(id); ok {
			msg.E = append(msg.E, se)
			seen[id] = true
		}
	}
	for id, se := range w.sess.remote {
		if seen[id] || w.sess.replicate[id] {
			continue
		}
		msg.E = append(msg.E, se)
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (w *World) netSendReplicas() {
	if w.net == nil || w.sess == nil {
		return
	}
	if !w.sess.isHost {
		if len(w.sess.replicate) == 0 {
			return
		}
		w.netSendProto(1, "OWN|"+w.netSnapshot())
		return
	}
	if len(w.sess.replicate) == 0 && len(w.sess.remote) == 0 {
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
		w.applyPose(se.ID, se.X, se.Y, se.Z, se.P, se.Ya, se.R)
	}
	return nil
}

type netBlend struct {
	x0, y0, z0  float64
	p0, ya0, r0 float64
	x1, y1, z1  float64
	p1, ya1, r1 float64
	t0          time.Time
}

func (w *World) currentPose(id int) (snapEnt, bool) {
	if e := w.ents[id]; e != nil && e.node != nil {
		w.refreshWorldMatrices()
		pos := worldPos(e.node.GetNode())
		x, y, z := fromG3N(pos.X, pos.Y, pos.Z)
		return snapEnt{
			ID: id, X: float64(x), Y: float64(y), Z: float64(z),
			P: float64(e.pitch), Ya: float64(e.yaw), R: float64(e.roll),
		}, true
	}
	if sp := w.sprites[id]; sp != nil {
		return snapEnt{ID: id, X: sp.x, Y: sp.y}, true
	}
	return snapEnt{}, false
}

func (w *World) applyPose(id int, x, y, z, p, ya, r float64) {
	if e := w.ents[id]; e != nil && e.node != nil {
		gx, gy, gz := toG3N(float32(x), float32(y), float32(z))
		e.node.GetNode().SetPosition(gx, gy, gz)
		e.pitch, e.yaw, e.roll = float32(p), float32(ya), float32(r)
		w.applyRot(e)
		return
	}
	if sp := w.sprites[id]; sp != nil {
		sp.x, sp.y = x, y
	}
}

func (w *World) storeRemote(s string) {
	var msg snapMsg
	if err := json.Unmarshal([]byte(s), &msg); err != nil {
		return
	}
	if w.sess.remote == nil {
		w.sess.remote = map[int]snapEnt{}
	}
	for _, se := range msg.E {
		if w.sess.replicate[se.ID] {
			continue
		}
		w.sess.remote[se.ID] = se
	}
}

func (w *World) netQueueSnapshot(s string) error {
	var msg snapMsg
	if err := json.Unmarshal([]byte(s), &msg); err != nil {
		return err
	}
	if msg.T > w.sess.tick {
		w.sess.tick = msg.T
	}
	for _, se := range msg.E {
		if w.sess.replicate[se.ID] {
			continue
		}
		if _, ok := w.currentPose(se.ID); !ok {
			continue
		}
		if w.sess.smoothMS <= 0 {
			w.applyPose(se.ID, se.X, se.Y, se.Z, se.P, se.Ya, se.R)
			continue
		}
		cur, _ := w.currentPose(se.ID)
		if w.sess.blends == nil {
			w.sess.blends = map[int]*netBlend{}
		}
		w.sess.blends[se.ID] = &netBlend{
			x0: cur.X, y0: cur.Y, z0: cur.Z, p0: cur.P, ya0: cur.Ya, r0: cur.R,
			x1: se.X, y1: se.Y, z1: se.Z, p1: se.P, ya1: se.Ya, r1: se.R,
			t0: time.Now(),
		}
	}
	return nil
}

func (w *World) netSmooth() {
	if w.sess == nil || len(w.sess.blends) == 0 {
		return
	}
	dur := time.Duration(w.sess.smoothMS) * time.Millisecond
	now := time.Now()
	for id, b := range w.sess.blends {
		u := 1.0
		if dur > 0 {
			u = float64(now.Sub(b.t0)) / float64(dur)
			if u < 0 {
				u = 0
			}
			if u > 1 {
				u = 1
			}
		}
		w.applyPose(id,
			b.x0+(b.x1-b.x0)*u, b.y0+(b.y1-b.y0)*u, b.z0+(b.z1-b.z0)*u,
			b.p0+(b.p1-b.p0)*u, b.ya0+(b.ya1-b.ya0)*u, b.r0+(b.r1-b.r0)*u,
		)
		if u >= 1 {
			delete(w.sess.blends, id)
		}
	}
}

func (w *World) netAllReady() bool {
	if w.sess == nil || !w.sess.ready {
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
