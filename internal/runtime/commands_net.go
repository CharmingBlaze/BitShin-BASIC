package runtime

import (
	"strconv"
	"strings"
	"time"

	"bitshinbasic/internal/netenet"
	"bitshinbasic/internal/value"
)

func (w *World) netCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	host := n(func(a []value.Value) (value.Value, error) {
		h, err := netenet.Listen(argI(a, 0, 1234))
		if err != nil {
			return value.Value{}, err
		}
		w.net = h
		w.sess = newNetSess()
		w.sess.isHost = true
		w.sess.localID = 0
		return z()
	})
	connect := n(func(a []value.Value) (value.Value, error) {
		if w.net == nil {
			h, err := netenet.Dial(argS(a, 0), argI(a, 1, 1234))
			if err != nil {
				return value.Value{}, err
			}
			w.net = h
			if w.sess == nil {
				w.sess = newNetSess()
			}
			w.sess.isHost = false
			if w.sess.localID < 0 {
				w.sess.localID = 1
			}
			w.netBroadcastProto("NAME|" + w.sess.name)
			return value.Num(1), nil
		}
		id, err := w.net.Connect(argS(a, 0), argI(a, 1, 1234))
		return value.Num(float64(id)), err
	})
	closeNet := n(func(a []value.Value) (value.Value, error) {
		if w.net == nil {
			return z()
		}
		if len(a) > 0 {
			w.net.Disconnect(argI(a, 0, 1))
			delete(w.sess.peers, argI(a, 0, 1))
			return z()
		}
		w.net.Close()
		w.net = nil
		w.netInbox = nil
		w.sess = newNetSess()
		return z()
	})
	recv := n(func(a []value.Value) (value.Value, error) {
		w.netUpdate()
		if len(w.netInbox) == 0 {
			w.netMsg, w.netPeer, w.netKind = "", 0, 0
			return value.Num(0), nil
		}
		ev := w.netInbox[0]
		w.netInbox = w.netInbox[1:]
		w.netMsg, w.netPeer, w.netKind = ev.Message, ev.Peer, ev.Kind
		return value.Num(1), nil
	})
	m := map[string]cmd{
		"nethost":    host,
		"hostnet":    host,
		"netconnect": connect,
		"connectnet": connect,
		"netclose":   closeNet,
		"netupdate": n(func(a []value.Value) (value.Value, error) {
			w.netUpdate()
			return z()
		}),
		"netsend": n(func(a []value.Value) (value.Value, error) {
			if w.net == nil {
				return z()
			}
			peer := argI(a, 0, 1)
			msg := argS(a, 1)
			rel := true
			if len(a) >= 3 {
				rel = argI(a, 2, 1) != 0
			}
			return value.Num(0), w.net.Send(peer, msg, rel)
		}),
		"netsendreliable": n(func(a []value.Value) (value.Value, error) {
			if w.net == nil {
				return z()
			}
			return value.Num(0), w.net.Send(argI(a, 0, 1), argS(a, 1), true)
		}),
		"netbroadcast": n(func(a []value.Value) (value.Value, error) {
			if w.net == nil {
				return z()
			}
			return value.Num(0), w.net.Broadcast(argS(a, 0), argI(a, 1, 1) != 0)
		}),
		"netrecv":     recv,
		"netrecvmsg":  n(func(a []value.Value) (value.Value, error) { return value.Str(w.netMsg), nil }),
		"netmsg":      n(func(a []value.Value) (value.Value, error) { return value.Str(w.netMsg), nil }),
		"netrecvpeer": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.netPeer)), nil }),
		"netpeer": n(func(a []value.Value) (value.Value, error) {
			if len(a) == 0 {
				return value.Num(float64(w.netPeer)), nil
			}
			if w.net == nil {
				return value.Num(0), nil
			}
			ids := w.net.PeerIDs()
			i := argI(a, 0, 0)
			if i < 0 || i >= len(ids) {
				return value.Num(0), nil
			}
			return value.Num(float64(ids[i])), nil
		}),
		"netevent": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.netKind)), nil }),
		"netpeercount": n(func(a []value.Value) (value.Value, error) {
			if w.net == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.net.PeerCount())), nil
		}),
		"netconnected": n(func(a []value.Value) (value.Value, error) {
			if w.net == nil {
				return value.Num(0), nil
			}
			if w.sess.isHost || w.net.Connected() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"netbackend": n(func(a []value.Value) (value.Value, error) {
			if w.net != nil {
				return value.Str(w.net.Backend()), nil
			}
			for _, h := range w.nets {
				return value.Str(h.Backend()), nil
			}
			return value.Str(""), nil
		}),
		"netdisconnect": n(func(a []value.Value) (value.Value, error) {
			if w.net != nil {
				id := argI(a, 0, 1)
				w.net.Disconnect(id)
				delete(w.sess.peers, id)
			}
			return z()
		}),
		"netping": n(func(a []value.Value) (value.Value, error) {
			if w.net == nil {
				return z()
			}
			w.sess.pingSeq++
			body := "PING|" + itoa(w.sess.pingSeq) + "|" + itoa64(nowMS())
			peer := argI(a, 0, 0)
			if peer > 0 {
				w.netSendProto(peer, body)
			} else {
				w.netBroadcastProto(body)
			}
			return z()
		}),
		"netrtt": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.netRTT(argI(a, 0, 0))), nil
		}),
		"netplayername": n(func(a []value.Value) (value.Value, error) {
			peer := argI(a, 0, -1)
			if peer >= 0 {
				if p := w.sess.peers[peer]; p != nil {
					return value.Str(p.name), nil
				}
				return value.Str(""), nil
			}
			return value.Str(w.sess.name), nil
		}),
		"setnetplayername": n(func(a []value.Value) (value.Value, error) {
			w.sess.name = argS(a, 0)
			if w.net != nil && w.sess.name != "" {
				w.netBroadcastProto("NAME|" + w.sess.name)
			}
			return z()
		}),
		"netlocalid": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.sess.localID)), nil
		}),
		"netready": n(func(a []value.Value) (value.Value, error) {
			on := true
			if len(a) > 0 {
				on = argI(a, 0, 1) != 0
			}
			w.sess.ready = on
			flag := "0"
			if on {
				flag = "1"
			}
			w.netBroadcastProto("READY|" + flag)
			return z()
		}),
		"netallready": n(func(a []value.Value) (value.Value, error) {
			if w.netAllReady() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"netroom": n(func(a []value.Value) (value.Value, error) {
			if len(a) > 0 {
				w.sess.room = argS(a, 0)
				w.netBroadcastProto("ROOM|" + w.sess.room)
			}
			return value.Str(w.sess.room), nil
		}),
		"setnetroom": n(func(a []value.Value) (value.Value, error) {
			w.sess.room = argS(a, 0)
			w.netBroadcastProto("ROOM|" + w.sess.room)
			return z()
		}),
		"netsendjson": n(func(a []value.Value) (value.Value, error) {
			if w.net == nil {
				return z()
			}
			j := argS(a, 0)
			peer := argI(a, 1, 0)
			body := "JSON|" + j
			if peer > 0 {
				w.netSendProto(peer, body)
			} else {
				w.netBroadcastProto(body)
			}
			return z()
		}),
		"netrecvjson": n(func(a []value.Value) (value.Value, error) {
			w.netUpdate()
			if len(w.sess.jsonQ) == 0 {
				return value.Str(""), nil
			}
			s := w.sess.jsonQ[0]
			w.sess.jsonQ = w.sess.jsonQ[1:]
			w.sess.lastJSON = s
			return value.Str(s), nil
		}),
		"netreplicate": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			on := true
			if len(a) >= 2 {
				on = argI(a, 1, 1) != 0
			}
			if on {
				w.sess.replicate[id] = true
			} else {
				delete(w.sess.replicate, id)
			}
			return z()
		}),
		"netapplysnapshot": n(func(a []value.Value) (value.Value, error) {
			if err := w.netApplySnapshot(argS(a, 0)); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		"netsnapshot": n(func(a []value.Value) (value.Value, error) {
			return value.Str(w.netSnapshot()), nil
		}),
		"nettick": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.sess.tick)), nil
		}),
		"netcall": n(func(a []value.Value) (value.Value, error) {
			name := argS(a, 0)
			arg := argS(a, 1)
			peer := argI(a, 2, 0)
			body := "RPC|" + name + "|" + arg
			if w.net == nil {
				w.netInvoke(name, arg, 0)
				return z()
			}
			if peer > 0 {
				w.netSendProto(peer, body)
			} else {
				w.netBroadcastProto(body)
			}
			return z()
		}),
		"netregister": n(func(a []value.Value) (value.Value, error) {
			key := strings.ToLower(strings.TrimSpace(argS(a, 0)))
			if key != "" {
				w.sess.rpcs[key] = true
			}
			return z()
		}),
	}
	createHost := n(func(a []value.Value) (value.Value, error) {
		port := argI(a, 0, 27015)
		max := argI(a, 1, 32)
		h, err := netenet.ListenMax(port, max)
		if err != nil {
			return value.Value{}, err
		}
		return value.Num(float64(w.bindScriptHost(h))), nil
	})
	dialClient := n(func(a []value.Value) (value.Value, error) {
		h, err := netenet.Dial(argS(a, 0), argI(a, 1, 27015))
		if err != nil {
			return value.Value{}, err
		}
		return value.Num(float64(w.bindScriptHost(h))), nil
	})
	connectHost := n(func(a []value.Value) (value.Value, error) {
		if len(a) >= 1 && a[0].Kind == value.KindStr {
			return dialClient(a)
		}
		h := w.scriptHost(argI(a, 0, 0))
		if h == nil {
			return value.Num(0), nil
		}
		peer, err := h.Connect(argS(a, 1), argI(a, 2, 27015))
		return value.Num(float64(peer)), err
	})
	sendOnHost := n(func(a []value.Value) (value.Value, error) {
		h := w.scriptHost(argI(a, 0, 0))
		if h == nil {
			return z()
		}
		rel := true
		if len(a) >= 4 {
			rel = argI(a, 3, 1) != 0
		}
		return value.Num(0), h.Send(argI(a, 1, 1), argS(a, 2), rel)
	})
	sendNetworkMsg := n(func(a []value.Value) (value.Value, error) {
		h := w.activeHost()
		if h == nil {
			return z()
		}
		rel := true
		if len(a) >= 3 {
			rel = argI(a, 2, 1) != 0
		}
		return value.Num(0), h.Send(argI(a, 0, 1), argS(a, 1), rel)
	})
	sendNetwork := n(func(a []value.Value) (value.Value, error) {
		if len(a) >= 3 && a[2].Kind == value.KindStr {
			return sendOnHost(a)
		}
		return sendNetworkMsg(a)
	})
	disconnectNet := n(func(a []value.Value) (value.Value, error) {
		hostID := 0
		peer := argI(a, 0, 1)
		if len(a) >= 2 {
			hostID = argI(a, 0, 0)
			peer = argI(a, 1, 1)
		}
		h := w.scriptHost(hostID)
		if h == nil {
			h = w.activeHost()
		}
		if h != nil {
			h.Disconnect(peer)
		}
		return z()
	})
	closeHost := n(func(a []value.Value) (value.Value, error) {
		id := argI(a, 0, 0)
		h := w.nets[id]
		if h == nil && (len(a) == 0 || id == 0) {
			h = w.activeHost()
			for hid, hh := range w.nets {
				if hh == h {
					id = hid
					break
				}
			}
		}
		if h != nil {
			if w.net == h {
				w.net = nil
			}
			h.Close()
			delete(w.nets, id)
			delete(w.netQ, id)
		}
		return z()
	})
	getPeerIP := n(func(a []value.Value) (value.Value, error) {
		if len(a) > 0 {
			peer := argI(a, 0, 0)
			if h := w.scriptHost(argI(a, 1, 0)); h != nil {
				return value.Str(h.PeerIP(peer)), nil
			}
			if w.net != nil {
				return value.Str(w.net.PeerIP(peer)), nil
			}
		}
		return value.Str(w.netIP), nil
	})
	m["createhost"] = createHost
	m["createnetworkhost"] = createHost
	m["connecthost"] = connectHost
	m["createnetworkclient"] = dialClient
	m["connectnetwork"] = dialClient
	m["connect"] = dialClient
	m["pollnetwork"] = n(func(a []value.Value) (value.Value, error) {
		return w.pollNetwork(argI(a, 0, 0)), nil
	})
	m["sendnet"] = sendOnHost
	m["sendnetwork"] = sendNetwork
	m["sendnetworkmessage"] = sendNetworkMsg
	m["disconnectnetwork"] = disconnectNet
	m["networkdisconnect"] = disconnectNet
	m["closehost"] = closeHost
	m["closenetworkhost"] = closeHost
	m["closenetwork"] = closeHost
	m["getneteventtype"] = n(func(a []value.Value) (value.Value, error) {
		return value.Num(float64(w.netKind)), nil
	})
	m["getnetworkeventtype"] = n(func(a []value.Value) (value.Value, error) {
		return value.Num(float64(w.netKind)), nil
	})
	m["getnetworkpeerid"] = n(func(a []value.Value) (value.Value, error) {
		return value.Num(float64(w.netPeer)), nil
	})
	m["getnetworkdata"] = n(func(a []value.Value) (value.Value, error) {
		return value.Str(w.netMsg), nil
	})
	m["getnetpeerip"] = getPeerIP
	m["getnetworkpeerip"] = getPeerIP
	m["netpeerip"] = n(func(a []value.Value) (value.Value, error) {
		if w.net != nil && len(a) > 0 {
			return value.Str(w.net.PeerIP(argI(a, 0, 0))), nil
		}
		return value.Str(w.netIP), nil
	})
	m["net_none"] = n(func(a []value.Value) (value.Value, error) { return value.Num(float64(netenet.EventNone)), nil })
	m["net_connect"] = n(func(a []value.Value) (value.Value, error) { return value.Num(float64(netenet.EventConnect)), nil })
	m["net_receive"] = n(func(a []value.Value) (value.Value, error) { return value.Num(float64(netenet.EventReceive)), nil })
	m["net_recv"] = n(func(a []value.Value) (value.Value, error) { return value.Num(float64(netenet.EventReceive)), nil })
	m["net_disconnect"] = n(func(a []value.Value) (value.Value, error) { return value.Num(float64(netenet.EventDisconnect)), nil })
	return m
}

func (w *World) bindScriptHost(h netenet.Host) int {
	id := w.nextNet
	w.nextNet++
	if w.nets == nil {
		w.nets = map[int]netenet.Host{}
	}
	w.nets[id] = h
	if w.net == nil {
		w.net = h
	}
	return id
}

func (w *World) scriptHost(id int) netenet.Host {
	if h := w.nets[id]; h != nil {
		return h
	}
	if id == 0 {
		return w.activeHost()
	}
	return nil
}

func (w *World) activeHost() netenet.Host {
	if w.net != nil {
		return w.net
	}
	if h := w.nets[w.nextNet-1]; h != nil {
		return h
	}
	for _, h := range w.nets {
		return h
	}
	return nil
}

func (w *World) scriptOwnsNet() bool {
	if w.net == nil {
		return false
	}
	for _, h := range w.nets {
		if h == w.net {
			return true
		}
	}
	return false
}

func (w *World) pollNetwork(hostID int) value.Value {
	h := w.scriptHost(hostID)
	ev := netenet.Event{}
	if h != nil {
		if w.netQ == nil {
			w.netQ = map[int][]netenet.Event{}
		}
		q := append(w.netQ[hostID], h.Service()...)
		if len(q) > 0 {
			ev = q[0]
			q = q[1:]
		}
		w.netQ[hostID] = q
	}
	w.netKind, w.netPeer, w.netMsg, w.netIP = ev.Kind, ev.Peer, ev.Message, ev.IP
	return packNetEvent(ev)
}

func packNetEvent(ev netenet.Event) value.Value {
	v := value.StructOf("netevent", []string{"type", "peer", "peerid", "message$", "msg$", "data$", "ip$", "peerip$"})
	v.SetField("type", value.Num(float64(ev.Kind)))
	v.SetField("peer", value.Num(float64(ev.Peer)))
	v.SetField("peerid", value.Num(float64(ev.Peer)))
	v.SetField("message", value.Str(ev.Message))
	v.SetField("msg", value.Str(ev.Message))
	v.SetField("data", value.Str(ev.Message))
	v.SetField("ip", value.Str(ev.IP))
	v.SetField("peerip", value.Str(ev.IP))
	return v
}

func itoa(n int) string { return strconv.Itoa(n) }

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

func nowMS() int64 { return time.Now().UnixMilli() }
