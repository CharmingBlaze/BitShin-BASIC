//go:build enet

package netenet

import (
	"github.com/codecat/go-enet"
)

func init() { initNet() }

func initNet() { enet.Initialize() }

type enetHost struct {
	h     enet.Host
	peers map[int]enet.Peer
	next  int
}

func Listen(port int) (Host, error) {
	return ListenMax(port, 32)
}

func ListenMax(port, maxPeers int) (Host, error) {
	if maxPeers < 1 {
		maxPeers = 32
	}
	h, err := enet.NewHost(enet.NewListenAddress(uint16(port)), uint64(maxPeers), 2, 0, 0)
	if err != nil {
		return nil, err
	}
	return &enetHost{h: h, peers: map[int]enet.Peer{}, next: 1}, nil
}

func Dial(ip string, port int) (Host, error) {
	h, err := enet.NewHost(nil, 1, 2, 0, 0)
	if err != nil {
		return nil, err
	}
	eh := &enetHost{h: h, peers: map[int]enet.Peer{}, next: 1}
	p, err := h.Connect(enet.NewAddress(ip, uint16(port)), 2, 0)
	if err != nil {
		h.Destroy()
		return nil, err
	}
	eh.peers[1] = p
	eh.next = 2
	return eh, nil
}

func (e *enetHost) Backend() string { return BackendENet }

func (e *enetHost) Service() []Event {
	var out []Event
	for {
		ev := e.h.Service(0)
		if ev == nil || ev.GetType() == enet.EventNone {
			break
		}
		switch ev.GetType() {
		case enet.EventConnect:
			id := e.track(ev.GetPeer())
			out = append(out, Event{Kind: EventConnect, Peer: id, IP: peerIP(ev.GetPeer())})
		case enet.EventReceive:
			pkt := ev.GetPacket()
			id := e.track(ev.GetPeer())
			out = append(out, Event{Kind: EventReceive, Peer: id, Message: string(pkt.GetData()), IP: peerIP(ev.GetPeer())})
			pkt.Destroy()
		case enet.EventDisconnect:
			id := e.track(ev.GetPeer())
			out = append(out, Event{Kind: EventDisconnect, Peer: id, IP: peerIP(ev.GetPeer())})
		}
	}
	return out
}

func (e *enetHost) track(p enet.Peer) int {
	for id, existing := range e.peers {
		if existing == p {
			return id
		}
	}
	id := e.next
	e.next++
	e.peers[id] = p
	return id
}

func (e *enetHost) Connect(ip string, port int) (int, error) {
	p, err := e.h.Connect(enet.NewAddress(ip, uint16(port)), 2, 0)
	if err != nil {
		return 0, err
	}
	id := e.next
	e.next++
	e.peers[id] = p
	return id, nil
}

func (e *enetHost) Send(peer int, msg string, reliable bool) error {
	p := e.peers[peer]
	if p == nil {
		return errNoPeer
	}
	flags := enet.PacketFlags(0)
	if reliable {
		flags = enet.PacketFlagReliable
	}
	return p.SendString(msg, 0, flags)
}

func (e *enetHost) Broadcast(msg string, reliable bool) error {
	flags := enet.PacketFlags(0)
	if reliable {
		flags = enet.PacketFlagReliable
	}
	return e.h.BroadcastString(msg, 0, flags)
}

func (e *enetHost) Disconnect(peer int) {
	if p := e.peers[peer]; p != nil {
		p.Disconnect(0)
		delete(e.peers, peer)
	}
}

func (e *enetHost) Close() { e.h.Destroy() }

func (e *enetHost) PeerCount() int { return len(e.peers) }

func (e *enetHost) PeerIDs() []int {
	ids := make([]int, 0, len(e.peers))
	for id := range e.peers {
		ids = append(ids, id)
	}
	return ids
}

func (e *enetHost) HasPeer(id int) bool {
	_, ok := e.peers[id]
	return ok
}

func (e *enetHost) Connected() bool { return e.h != nil }

func (e *enetHost) PeerIP(id int) string {
	if p := e.peers[id]; p != nil {
		return peerIP(p)
	}
	return ""
}

func peerIP(p enet.Peer) string {
	if p == nil {
		return ""
	}
	return p.GetAddress().String()
}
