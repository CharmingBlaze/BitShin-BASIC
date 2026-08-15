; Keys, KeyHit, mouse, mouse look, gamepad + deadzone.

Graphics3D(800, 600)
SetWindowTitle("Input")
cam = CreateCamera()
SetPosition(cam, 0, 1.6, 0)
HidePointer()
SetCursorMode(2)
SetRawMouse(True)
SetGamepadDeadzone(0.2)

light = CreateLight()
SetRotation(light, 55, 20, 0)
ground = CreatePlane()
SetPosition(ground, 0, 0, 12)
SetEntityColor(ground, 42, 48, 58)
cube = CreateCube()
SetPosition(cube, 0, 0.5, 8)
SetEntityColor(cube, 80, 170, 255)

hits = 0

While Not WindowShouldClose()
    MouseLook(cam, 0.12, -85, 85)
    If KeyDown(KEY_W) Then MoveEntity(cam, 0, 0, 0.12)
    If KeyDown(KEY_S) Then MoveEntity(cam, 0, 0, -0.12)
    If KeyDown(KEY_A) Then MoveEntity(cam, -0.12, 0, 0)
    If KeyDown(KEY_D) Then MoveEntity(cam, 0.12, 0, 0)
    If KeyHit(KEY_SPACE) Then hits = hits + 1
    If MouseHit(1) Then TurnEntity(cube, 0, 25, 0)
    lx# = GamepadAxis(0, 0)
    ly# = GamepadAxis(0, 1)
    If lx# <> 0 Or ly# <> 0 Then
        MoveEntity(cam, lx# * 0.12, 0, -ly# * 0.12)
    EndIf
    RenderWorld
    Text(12, 12, "WASD + mouse look  Space hits=" + Str(hits))
    Text(12, 30, "Mouse " + Str(MouseX()) + "," + Str(MouseY()) + " wheel=" + Str(MouseZ()))
    Text(12, 48, "Pad present=" + Str(GamepadPresent(0)) + " deadzone=" + Str(GamepadDeadzone()))
    Flip
Wend
End
