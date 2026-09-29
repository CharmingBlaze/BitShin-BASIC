; Net Host — CreateNetworkHost + PollNetwork with a spinning cube
; Esc quits after a few frames. ENet with -tags enet; else UDP.

Graphics3D(800, 600)
SetWindowTitle("BitShin BASIC — Net host")
SetBuffer(BackBuffer())

; Camera and light
cam = CreateCamera()
light = CreateLight()
SetRotation(light, 90, 0, 0)

; Scene — classic SetPosition / SetEntityColor
cube = CreateCube()
SetPosition(cube, 0, 0, 5)
SetEntityColor(cube, 70, 160, 255)

; Network host
host = CreateNetworkHost(27015, 32)
Print("CreateNetworkHost on 27015 backend=" + GetNetBackend())
Print("NET_CONNECT=", NET_CONNECT, " DISCONNECT=", NET_DISCONNECT, " RECEIVE=", NET_RECEIVE)

; Loop — poll net events, spin cube, Esc after frame 8
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
    TurnEntity(cube, 0.4, 0.8, 0)
    UpdateWorld
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
