package runtime

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

const discoveryPort = 27016

func discField(s string) string {
	s = strings.NewReplacer("|", " ", "\n", " ", "\r", " ").Replace(s)
	s = strings.TrimSpace(s)
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

func (w *World) writeBeacon() {
	if w == nil || w.discPort == 0 || w.sess == nil || !w.sess.isHost {
		return
	}
	w.discMsg.Store(fmt.Sprintf("~MB|HERE|%d|%s|%s", w.discPort, discField(w.sess.room), discField(w.sess.name)))
}

func (w *World) beaconText() string {
	v := w.discMsg.Load()
	s, _ := v.(string)
	return s
}

func (w *World) startAnnounce(port int) {
	w.stopAnnounce()
	if port <= 0 {
		port = defaultNetPort
	}
	w.discPort = port
	w.writeBeacon()
	stop := make(chan struct{})
	w.discStop = stop
	go announceLoop(w, stop)
}

func (w *World) stopAnnounce() {
	if w.discStop != nil {
		close(w.discStop)
		w.discStop = nil
	}
	w.discPort = 0
}

func announceLoop(w *World, stop chan struct{}) {
	sendDiscovery(w.beaconText())
	tick := time.NewTicker(400 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			sendDiscovery(w.beaconText())
		}
	}
}

func discoveryTargets() []*net.UDPAddr {
	seen := map[string]bool{}
	var out []*net.UDPAddr
	add := func(ip net.IP) {
		if ip == nil {
			return
		}
		ip = ip.To4()
		if ip == nil {
			return
		}
		key := ip.String()
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, &net.UDPAddr{IP: ip, Port: discoveryPort})
	}
	add(net.IPv4(127, 0, 0, 1))
	add(net.IPv4bcast)
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil || len(ipnet.Mask) != net.IPv4len {
				continue
			}
			ip4 := ipnet.IP.To4()
			bcast := make(net.IP, net.IPv4len)
			for i := 0; i < net.IPv4len; i++ {
				bcast[i] = ip4[i] | ^ipnet.Mask[i]
			}
			add(bcast)
		}
	}
	return out
}

func sendDiscovery(payload string) {
	if payload == "" {
		return
	}
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return
	}
	defer conn.Close()
	buf := []byte(payload)
	for _, addr := range discoveryTargets() {
		_, _ = conn.WriteTo(buf, addr)
	}
}

func findGames(ms int) string {
	if ms <= 0 {
		ms = 800
	}
	if ms > 5000 {
		ms = 5000
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: discoveryPort})
	if err != nil {
		return ""
	}
	defer conn.Close()
	deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
	_ = conn.SetReadDeadline(deadline)
	found := map[string]string{}
	buf := make([]byte, 512)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		port, room, name, ok := parseHere(string(buf[:n]))
		if !ok || addr == nil || addr.IP == nil {
			continue
		}
		ip := addr.IP.To4()
		if ip == nil {
			ip = addr.IP
		}
		key := ip.String() + "|" + port
		found[key] = ip.String() + "|" + port + "|" + room + "|" + name
	}
	if len(found) == 0 {
		return ""
	}
	lines := make([]string, 0, len(found))
	for _, line := range found {
		lines = append(lines, line)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func parseHere(msg string) (port, room, name string, ok bool) {
	const prefix = "~MB|HERE|"
	if !strings.HasPrefix(msg, prefix) {
		return "", "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(msg, prefix), "|", 3)
	if len(parts) < 3 || parts[0] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
