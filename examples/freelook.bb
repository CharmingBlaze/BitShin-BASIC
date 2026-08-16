; Free-look camera (Go: CreateFreeCamera + UpdateFreeLook).
; Close the window to quit — no Escape.

Graphics3D(960, 600)
AppTitle "BitShin BASIC — Free look"
CameraClsColor 70, 110, 160
AmbientLight 60, 70, 85

cam = CreateFreeCamera()
cam.Position(0, 2.2, -8)

light = CreateLight()
light.Rotate(50, 30, 0)

ground = CreatePlane()
ground.Scale(40, 1, 40)
ground.Color(52, 78, 58)

box = CreateCube().Scale(1.2, 1.2, 1.2).Position([0, 0.6, 6]).Color($FFECc8)

SetWeather(WEATHER_CLEAR)

While Not WindowShouldClose()
    UpdateFreeLook(cam, 8)
    RenderWorld
    Text 12, 12, "WASD + mouse   close window to quit"
    Flip
Wend
End
