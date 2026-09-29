; Net physics — hinged door; Space (or a net "kick") applies impulse.
; Optional UDP host on 27015. Esc quits after a few frames.

SetWindowTitle("BitShin BASIC — Net physics")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(18, 20, 28)

; Camera
cam = CreateCamera()
cam.Position([0, 6, -14])
cam.Rotate(18, 0, 0)
CreateLight()

; Static ground
ground = CreateCube().Scale(10, 0.2, 10).Position([0, 0, 8]).Color(50, 56, 70)
CreateBodyBox(ground, 10, 0.2, 10, 0)

; Hinged door
door = CreateCube().Scale(0.08, 1.6, 0.9).Position([0, 2, 8]).Color(210, 140, 70)
doorBody = CreateBodyBox(door, 0.08, 1.6, 0.9, 2)

; 0 = world/static hinge
doorHinge = CreateHingeJoint(0, doorBody, -1, 2, 8, 0, 1, 0)

; Optional network host (UDP unless built with -tags enet)
host = CreateNetworkHost(27015, 32)
If host
    Print("Server running on port 27015  backend=" + GetNetBackend())
Else
    Print("CreateNetworkHost failed — local hinge only")
EndIf

Print "Physics backend:", GetPhysicsBackend()
Print "Hinge id:", doorHinge
Print "NET_CONNECT=", NET_CONNECT, " DISCONNECT=", NET_DISCONNECT, " RECEIVE=", NET_RECEIVE

frames = 0
While 1
    frames = frames + 1
    If host
        netEvent = PollNetwork(host)
        If netEvent.Type = NET_CONNECT Then
            Print("Player Connected: ID " + netEvent.PeerID)
        ElseIf netEvent.Type = NET_RECEIVE Then
            Print("Message from " + netEvent.PeerID + ": " + netEvent.Data)
            If netEvent.Data = "kick" Then
                ApplyImpulse(doorBody, 0, 0, 25)
            EndIf
        EndIf
    EndIf
    If KeyHit(KEY_SPACE) Then ApplyImpulse(doorBody, 0, 0, 25)
    UpdateWorld()
    RenderWorld()
    Text(16, 16, "Hinge door  |  Space / net kick  |  Esc after a few frames")
    Text(16, 36, "backend " + GetPhysicsBackend() + "  hinge " + doorHinge + "  host " + host)
    Flip()
    If frames > 8 And KeyHit(KEY_ESCAPE) Then End
Wend
End
