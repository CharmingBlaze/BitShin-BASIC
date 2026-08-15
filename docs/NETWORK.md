# Network

Two layers, same backends:

1. **Session** — one implicit host: `NetHost` / `NetConnect` / `NetSend` / `NetRecv`.
2. **Handles** — `CreateNetworkHost` returns an id; `PollNetwork` returns a **`netevent` struct**.

Default build is **UDP**. `go build -tags enet` uses **ENet** (`GetNetBackend$()` / `NetBackend$()` is `"udp"` or `"enet"`).

## `PollNetwork` return value

`PollNetwork` does **not** return a bare type integer. It returns a **struct** (`packNetEvent` in the runtime):

| Field | Also | Meaning |
| --- | --- | --- |
| `Type` | | `NET_NONE` 0, `NET_CONNECT` 1, `NET_DISCONNECT` 2, `NET_RECEIVE` / `NET_RECV` 3 |
| `PeerID` | `Peer` | Peer handle |
| `Data` | `Message` / `Msg` | Payload string (empty on connect/disconnect) |
| `PeerIP` | `IP` | Remote address string |

The same poll also stores the last event for:

`GetNetworkEventType()` / `GetNetEventType()`  
`GetNetworkPeerID()`  
`GetNetworkData$()`  
`GetNetworkPeerIP()` / `GetNetPeerIP()` / `NetPeerIP(peer)`

`PollNetwork(host)` uses that host handle. `PollNetwork()` (no args) uses host **0**, which is the active / World host.

Idle poll: `Type = NET_NONE` (0). Always poll once per frame; do not assume a client connected.

## Host commands

| Command | Meaning |
| --- | --- |
| `CreateNetworkHost(port [, maxPeers])` | Listen. Alias `CreateHost`. Default port 27015, max 32 |
| `CreateNetworkClient(ip$, port)` | Dial. Aliases `Connect` / `ConnectNetwork` |
| `ConnectHost(host, ip$, port)` | Connect an **existing** host handle. `ConnectHost(ip$, port)` is the same as dial |
| `PollNetwork(host)` | Next event **struct** (see above) |
| `SendNetworkMessage peer, data$ [, reliable]` | On the World / active host. Default reliable |
| `SendNetwork host, peer, data$ [, reliable]` | Named host. Also `SendNet` |
| `SendNetwork peer, data$ [, reliable]` | Active host (when the third arg is not a string) |
| `DisconnectNetwork peer` or `DisconnectNetwork host, peer` | |
| `CloseNetworkHost host` | Also `CloseHost` / `CloseNetwork` |

```basic
; examples/net_host.bb — build with -tags enet for ENet, else UDP.
Graphics3D(800, 600)
cam = CreateCamera()
CreateLight()

host = CreateNetworkHost(27015, 32)
Print("backend=" + GetNetBackend())
Print("NET_CONNECT=", NET_CONNECT, " DISCONNECT=", NET_DISCONNECT, " RECEIVE=", NET_RECEIVE)

frames = 0
While 1
    frames = frames + 1
    netEvent = PollNetwork(host)
    If netEvent.Type = NET_CONNECT Then
        Print("Player joined: " + netEvent.PeerIP)
    ElseIf netEvent.Type = NET_RECEIVE Then
        Print("recv peer=" + Str(netEvent.PeerID) + " data=" + netEvent.Data)
        SendNetworkMessage(netEvent.PeerID, "ack", True)
    ElseIf netEvent.Type = NET_DISCONNECT Then
        Print("Player left: " + Str(netEvent.PeerID))
    EndIf
    UpdateWorld
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
```

Client sketch:

```basic
client = CreateNetworkClient("127.0.0.1", 27015)
; …
ev = PollNetwork(client)
If ev.Type = NET_CONNECT Then
    SendNetwork(client, ev.PeerID, "hello", True)
EndIf
```

## Session API (still works)

One `w.net` slot. Fine for a single room.

| Command | Meaning |
| --- | --- |
| `NetHost` / `HostNet port` | Listen (default 1234) |
| `NetConnect` / `ConnectNet host$, port` | Dial |
| `NetClose` | Close session (or `NetClose peer` to drop one) |
| `NetSend peer, msg$ [, reliable]` | |
| `NetSendReliable peer, msg$` | |
| `NetBroadcast msg$ [, reliable]` | |
| `NetRecv()` | 1 if an event was taken; then use `NetMsg$` / `NetPeer` / `NetEvent` |
| `NetUpdate` | Pump without taking |
| `NetMsg$()` / `NetRecvMsg$()` / `GetNetMsg$()` | Last payload |
| `NetPeer()` / `GetNetPeer()` | Last peer |
| `NetEvent()` / `GetNetEvent()` | 1 connect, 2 disconnect, 3 receive |
| `NetPeerCount()` / `NetConnected()` | |
| `NetDisconnect peer` | |
| `NetPing [peer]` / `NetRTT([peer])` | App-level ping (`PING\|seq\|ms`) |
| `SetNetPlayerName s$` / `NetPlayerName$([peer])` | Omit peer = local |
| `NetLocalID()` | |
| `NetReady [on]` / `NetAllReady()` | |
| `NetRoom$()` / `SetNetRoom s$` | |
| `NetSendJSON json$ [, peer]` / `NetRecvJSON$()` | |
| `NetReplicate e [, on]` / `NetSnapshot$()` / `NetApplySnapshot json$` | Transform snapshot |
| `NetTick()` | |
| `NetRegister name$` / `NetCall name$ [, arg$] [, peer]` | RPC; local if no net |

`NetRecv` returns **1 or 0**, not a struct. Use `PollNetwork` if you want `netEvent.Type`.

## Constants

| Name | Value |
| --- | --- |
| `NET_NONE` | 0 |
| `NET_CONNECT` | 1 |
| `NET_DISCONNECT` | 2 |
| `NET_RECEIVE` / `NET_RECV` | 3 |

Same order as ENet (none, connect, disconnect, receive).

Also: `examples/net_echo.bb`.
