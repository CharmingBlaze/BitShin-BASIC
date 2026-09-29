; Stream — walk a streamed prop grid (chunks load around the player)
; WASD moves. Esc quits.

Graphics3D(960, 600)
SetWindowTitle("BitShin BASIC — Stream")
SetAmbientLight(50, 55, 65)

; Camera and light
cam = CreateCamera()
sun = CreateLight()
SetRotation(sun, 70, 20, 0)

; Player sphere the stream follows
player = CreateSphere(8)
SetEntityColor(player, 255, 200, 60)
SetPosition(player, 0, 1, 0)

CreateWorldStream(20, 2)
SetStreamFollow(player)

; Loop — WASD move, CameraFollow, stream HUD
While Not KeyDown(1)
    dt# = DeltaTime()
    If KeyDown(KEY_W) Then SetPosition(player, EntityX(player), 1, EntityZ(player) + 12 * dt)
    If KeyDown(KEY_S) Then SetPosition(player, EntityX(player), 1, EntityZ(player) - 12 * dt)
    If KeyDown(KEY_A) Then SetPosition(player, EntityX(player) - 12 * dt, 1, EntityZ(player))
    If KeyDown(KEY_D) Then SetPosition(player, EntityX(player) + 12 * dt, 1, EntityZ(player))
    CameraFollow(cam, player, 16, 8, 10, 0, 18)
    Text(12, 12, "chunks=" + Str(StreamChunkCount()) + " jobs=" + Str(JobCount()))
    RenderWorld
    Flip
Wend
End
