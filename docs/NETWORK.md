# Network

One player hosts. Friends join that game. Everything below is that one room.

Call `NetUpdate` once each frame so messages, the lobby, and movement stay current.

The game port is **27015**. The normal build is UDP. `go build -tags enet` uses ENet, which keeps packets that UDP might drop. `NetBackend$()` is `"udp"` or `"enet"`.

Examples: `examples/net_lobby.mb`, `examples/net_echo.mb`.

## Host

```basic
SetNetPlayerName("Ada")
If Not Net.Host() Then
    Print("Could not open the room")
    End
EndIf
SetNetRoom("harbor")
```

`Net.Host()` returns 1. It listens on 27015 and announces the room on the local network. `SetNetPlayerName` and `SetNetRoom` before or just after `HostGame` set the name friends will see.

`Net.Host(1973)` uses another game port. `NetClose` shuts the room and stops the announcement. `NetClose peer` or `NetDisconnect peer` drops one player.

## Find a game and join

On the same network, `FindGames$()` listens for about 800 milliseconds and returns one line per room:

```text
192.168.1.20|27015|harbor|Ada
```

That is `ip|port|room|name`. An empty string means nobody is hosting nearby. `Net.Find(2000)` waits longer. The longest wait is 5 seconds.

```basic
list$ = Net.Find()
Print(list$)
Net.Join("192.168.1.20", 27015)
```

`Net.Join("192.168.1.20")` uses port 27015. Call `SetNetPlayerName("Bo")` before joining.

`FindGames$` only hears rooms on the local network. A friend on another network needs the host's IP and an open game port.

## Waiting room

```basic
NetReady(1)

While Not NetStarted()
    NetUpdate()
    Print(NetLobby())
    If NetAllReady() Then StartGame()
Wend
```

`NetLobby$()` looks like this:

```text
room harbor
started 0
0 Ada 1
1 Bo 0
```

The first line is the room name. `started` is 1 after the match begins. Each player line is `id name ready`. The host's id is 0.

`NetReady(1)` marks this player ready. `NetAllReady()` is 1 when every joined player is ready. Only the host should call `StartGame()`. It returns 1 on the host and 0 on a client. Everyone else sees `NetStarted()` become 1.

A host with no friends yet is all-ready as soon as they call `NetReady(1)`.

## Chat

```basic
Net.Say("hello")

While 1
    NetUpdate()
    If NetRecv() And NetEvent() = NET_RECEIVE Then
        Print(NetPlayerName(NetPeer()) + ": " + NetMsg())
    EndIf
    Flip
Wend
```

`NetSay` reaches the room. The host sends it to every player. A client sends it to the host. `NetRecv()` returns 1 when a chat line is waiting, then `NetMsg$()` is the text and `NetPeer()` is who sent it. `NetEvent()` is 1 for connect, 2 for disconnect, and 3 for a message.

## Score, map, and other short facts

```basic
SetNetInfo("map", "harbor")
SetNetInfo("score", "3")
Print(NetInfo("map"))
```

The host keeps these values and sends them to players who join. A client's `SetNetInfo` goes to the host, then out to the room. Keys are short names such as `map` or `score`. Values are short strings.

## Who is where

Create the same entities in the same order on every machine. The first `CreateCube` is id 1 on every machine, the second is id 2, and so on. Each machine calls `NetReplicate` only for the entity it moves. The host passes the other players' positions on.

```basic
ada = CreateCube()
bo = CreateCube()
hosting = 1

If hosting Then
    NetReplicate(ada)
Else
    NetReplicate(bo)
EndIf

While 1
    NetUpdate()
    RenderWorld()
    Flip
Wend
```

Remote entities glide for 100 milliseconds between updates. `SetNetSmooth(0)` makes them jump to the latest position. `SetNetSmooth(200)` glides more slowly.

This follows poses that arrive. It does not predict a move that has not been sent yet.

## A JSON file

Small JSON can be sent as one message. Keep it under about 4KB.

```basic
doc = Json.Load("level.json")
NetSendJSON(JSON$(doc))

raw$ = NetRecvJSON()
If raw$ <> "" Then
    got = Json.Parse(raw$)
    Json.Save(got, "level.json")
EndIf
```

`NetSendJSON(json$)` sends to the whole room. `NetSendJSON(json$, peer)` sends to one player. `NetRecvJSON$()` returns the next JSON string, or `""` when none is waiting.

`JSONLoad` and `JSONSave` only touch the local disk. The network commands move the text.

## Commands

`NetHost` is `HostGame`. `NetConnect` is `JoinGame`.

| What you want | Command |
| --- | --- |
| Open the room | `Net.Host([port])` |
| List LAN rooms | `FindGames$([ms])` |
| Join | `Net.Join(ip$ [, port])` |
| Leave, or drop one player | `NetClose` / `NetDisconnect peer` |
| Pump the room | `NetUpdate` |
| Your name | `SetNetPlayerName s$` / `NetPlayerName$([peer])` |
| Room name | `SetNetRoom s$` / `NetRoom$()` |
| Who is waiting | `NetLobby$()` |
| Ready | `NetReady [on]` / `NetAllReady()` |
| Start | `StartGame()` / `NetStarted()` |
| Your id | `NetLocalID()` |
| How many players | `NetPeerCount()` / `NetConnected()` |
| Ping | `NetPing [peer]` / `NetRTT([peer])` |
| Chat | `NetSay msg$` |
| Read chat | `NetRecv()` / `NetMsg$()` / `NetPeer()` / `NetEvent()` |
| Any short text | `NetSend peer, msg$` / `NetBroadcast msg$` |
| Shared fact | `SetNetInfo key$, value$` / `NetInfo$(key$)` |
| JSON text | `NetSendJSON json$ [, peer]` / `NetRecvJSON$()` |
| Positions | `NetReplicate entity [, on]` / `SetNetSmooth(ms)` |
| Call a function on another machine | `NetRegister name$` / `NetCall name$ [, arg$] [, peer]` |

`NetRegister("hit")` marks a function. `NetCall("hit", "12")` runs it there with the text argument and the sender's peer id. With no session, `NetCall` runs the function locally.

`NetSendReliable` is the same as `NetSend` on UDP. ENet is the build that treats the reliable flag as delivery you can count on.

## Several connections

`HostGame` is the room above. `CreateNetworkHost` is a lower-level socket. It does not run the lobby, `FindGames$`, shared facts, or `NetReplicate`. You send and read raw text yourself. See `examples/net_host.mb` and `examples/net_physics.mb`.

```basic
host = CreateNetworkHost(27015, 32)
ev = PollNetwork(host)
If ev.Type = NET_RECEIVE Then
    Print(ev.Data)
    SendNetworkMessage(ev.PeerID, "ack", True)
EndIf
```

`PollNetwork` returns a struct:

| Field | Also called | Meaning |
| --- | --- | --- |
| `Type` | | `NET_NONE` 0, `NET_CONNECT` 1, `NET_DISCONNECT` 2, `NET_RECEIVE` 3 |
| `PeerID` | `Peer` | Who |
| `Data` | `Message`, `Msg` | Text. Empty on connect and disconnect |
| `PeerIP` | `IP` | Their address |

`PollNetwork()` with no arguments uses the active host. An idle poll has `Type` 0. Poll once a frame.

`CreateNetworkClient(ip$, port)` dials. `ConnectHost(host, ip$, port)` dials from a host you already opened. `CloseNetworkHost host` closes it. `GetNetworkData$()`, `GetNetworkPeerID()`, and `GetNetworkPeerIP$()` repeat the last poll.

`NET_NONE` is 0, `NET_CONNECT` is 1, `NET_DISCONNECT` is 2, `NET_RECEIVE` is 3.
