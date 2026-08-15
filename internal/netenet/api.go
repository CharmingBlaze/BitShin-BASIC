// Package netenet is host/connect/send. Default build uses UDP; -tags enet uses go-enet.
package netenet

const (
	BackendENet = "enet"
	BackendUDP  = "udp"

	EventNone       = 0
	EventConnect    = 1
	EventDisconnect = 2
	EventReceive    = 3
)

type Event struct {
	Kind    int // 0 none, 1 connect, 2 disconnect, 3 receive
	Peer    int
	Message string
	IP      string
}

// NetworkEvent is the script-facing poll result (Type / PeerID / Data).
type NetworkEvent struct {
	Type   int
	PeerID int
	Data   string
}

func Initialize() { initNet() }

type Host interface {
	Backend() string
	Service() []Event
	Connect(ip string, port int) (peer int, err error)
	Send(peer int, msg string, reliable bool) error
	Broadcast(msg string, reliable bool) error
	Disconnect(peer int)
	Close()
	PeerCount() int
	PeerIDs() []int
	HasPeer(id int) bool
	Connected() bool
	PeerIP(id int) string
}
