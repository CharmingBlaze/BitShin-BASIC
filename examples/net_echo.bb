; Multiplayer stub — HostNet / ConnectNet / NetSend / NetRecv / NetClose
; ENet when built with -tags enet; otherwise UDP

SetWindowTitle("BitShin BASIC — Net echo")
Print("Starting host on 1973...")
HostNet(1973)
Print("Backend:", GetNetBackend())

peer = ConnectNet("127.0.0.1", 1973)
NetSend(peer, "hello from BitShin BASIC", 1)

ticks = 0
While ticks < 30
    ticks = ticks + 1
    If NetRecv() Then
        Print("event", GetNetEvent(), "peer", GetNetPeer(), "msg", GetNetMsg())
    EndIf
Wend
NetClose()
End
