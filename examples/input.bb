; Input — keys, KeyHit, mouse look, and gamepad with deadzone.
; WASD + mouse look, Space counts hits, left-click spins the cube. Close window to quit.

Graphics3D(800, 600)
SetWindowTitle("BitShin BASIC — Input")

; Camera
cam = CreateCamera().Position([0, 1.6, 0])
HidePointer()
SetCursorMode(2)
SetRawMouse(True)
SetGamepadDeadzone(0.2)

; Light / world
light = CreateLight().Rotate(55, 20, 0)
ground = CreatePlane().Position([0, 0, 12]).Color(42, 48, 58)
cube = CreateCube().Position([0, 0.5, 8]).Color(80, 170, 255)

hits = 0

; Loop
While Not WindowShouldClose()
    MouseLook(cam, 0.12, -85, 85)
    If KeyDown(KEY_W) Then cam.Move(0, 0, 0.12)
    If KeyDown(KEY_S) Then cam.Move(0, 0, -0.12)
    If KeyDown(KEY_A) Then cam.Move(-0.12, 0, 0)
    If KeyDown(KEY_D) Then cam.Move(0.12, 0, 0)
    If KeyHit(KEY_SPACE) Then hits = hits + 1
    If MouseHit(1) Then cube.Turn(0, 25, 0)
    lx# = GamepadAxis(0, 0)
    ly# = GamepadAxis(0, 1)
    If lx# <> 0 Or ly# <> 0 Then
        cam.Move(lx# * 0.12, 0, -ly# * 0.12)
    EndIf
    RenderWorld
    Text(12, 12, "WASD + mouse look  Space hits=" + Str(hits))
    Text(12, 30, "Mouse " + Str(MouseX()) + "," + Str(MouseY()) + " wheel=" + Str(MouseZ()))
    Text(12, 48, "Pad present=" + Str(GamepadPresent(0)) + " deadzone=" + Str(GamepadDeadzone()))
    Flip
Wend
End
