//go:build !enet

package netenet

import (
	"net"
	"strconv"
	"sync"
	"time"
)

// UDP stand-in so Net* commands work without libenet (go-enet on Windows
// links enet.lib for MSVC; MinGW/clang needs libenet.a — use -tags enet).

type udpHost struct {
	conn      *net.UDPConn
	peers     map[int]*net.UDPAddr
	rev       map[string]int
	next      int
	max       int
	inbox     []Event
	mu        sync.Mutex
	connected bool
}

func initNet() {}

func Listen(port int) (Host, error) {
	return ListenMax(port, 32)
}

func ListenMax(port, maxPeers int) (Host, error) {
	if maxPeers < 1 {
		maxPeers = 32
	}
	addr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	c, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	_ = c.SetReadDeadline(time.Time{})
	h := &udpHost{conn: c, peers: map[int]*net.UDPAddr{}, rev: map[string]int{}, next: 1, max: maxPeers}
	go h.readLoop()
	return h, nil
}

func Dial(ip string, port int) (Host, error) {
	raddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	c, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	h := &udpHost{conn: c, peers: map[int]*net.UDPAddr{1: raddr}, rev: map[string]int{raddr.String(): 1}, next: 2, connected: true}
	go h.readLoop()
	return h, nil
}

func (h *udpHost) Backend() string { return BackendUDP }

func (h *udpHost) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, addr, err := h.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		msg := string(buf[:n])
		h.mu.Lock()
		id, ok := h.rev[addr.String()]
		if !ok {
			if h.max > 0 && len(h.peers) >= h.max {
				h.mu.Unlock()
				continue
			}
			id = h.next
			h.next++
			h.peers[id] = addr
			h.rev[addr.String()] = id
			h.inbox = append(h.inbox, Event{Kind: EventConnect, Peer: id, IP: addr.IP.String()})
		}
		h.inbox = append(h.inbox, Event{Kind: EventReceive, Peer: id, Message: msg, IP: addr.IP.String()})
		h.mu.Unlock()
	}
}

func (h *udpHost) Service() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := h.inbox
	h.inbox = nil
	return out
}

func (h *udpHost) Connect(ip string, port int) (int, error) {
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return 0, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.next
	h.next++
	h.peers[id] = addr
	h.rev[addr.String()] = id
	_, _ = h.conn.WriteToUDP([]byte("HELLO"), addr)
	return id, nil
}

func (h *udpHost) Send(peer int, msg string, _ bool) error {
	h.mu.Lock()
	addr := h.peers[peer]
	connected := h.connected
	h.mu.Unlock()
	if connected || addr == nil {
		_, err := h.conn.Write([]byte(msg))
		return err
	}
	_, err := h.conn.WriteToUDP([]byte(msg), addr)
	return err
}

func (h *udpHost) Broadcast(msg string, _ bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var err error
	for _, addr := range h.peers {
		if _, e := h.conn.WriteToUDP([]byte(msg), addr); e != nil {
			err = e
		}
	}
	return err
}

func (h *udpHost) Disconnect(peer int) {
	h.mu.Lock()
	if addr := h.peers[peer]; addr != nil {
		delete(h.rev, addr.String())
	}
	delete(h.peers, peer)
	h.mu.Unlock()
}

func (h *udpHost) Close() { _ = h.conn.Close() }

func (h *udpHost) PeerCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.peers)
}

func (h *udpHost) PeerIDs() []int {
	h.mu.Lock()
	defer h.mu.Unlock()
	ids := make([]int, 0, len(h.peers))
	for id := range h.peers {
		ids = append(ids, id)
	}
	return ids
}

func (h *udpHost) HasPeer(id int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.peers[id]
	return ok
}

func (h *udpHost) Connected() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.conn != nil
}

func (h *udpHost) PeerIP(id int) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if addr := h.peers[id]; addr != nil {
		return addr.IP.String()
	}
	return ""
}
