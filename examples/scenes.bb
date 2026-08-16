; Menu / level pattern: ClearWorld then LoadScene a setup .bb
; Space loads the level chunk. Esc quits.

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — scenes")
SetCameraClsColor(22, 28, 40)
SetAmbientLight(80, 90, 110)

cam = CreateCamera()
SetPosition(cam, 0, 1.8, -6)
PointEntity(cam, 0, 0.5, 0)
light = CreateLight()
SetRotation(light, 40, 20, 0)

marker = CreateCube()
SetPosition(marker, 0, 0.5, 0)
SetEntityColor(marker, 80, 170, 255)

phase = 0

While Not KeyDown(KEY_ESCAPE)
    If phase = 0 Then
        TurnEntity(marker, 0, 0.6, 0)
        RenderWorld
        Text(16, 16, "BitShin BASIC — menu")
        Text(16, 36, "Space = LoadScene level_setup.bb")
        Text(16, 56, "Esc = quit")
        If KeyHit(KEY_SPACE) Then
            ClearWorld()
            cam = CreateCamera()
            SetPosition(cam, 0, 1.8, -4)
            PointEntity(cam, 0, 0.5, 6)
            light = CreateLight()
            SetRotation(light, 40, 20, 0)
            SetAmbientLight(80, 90, 110)
            LoadScene("level_setup.bb")
            phase = 1
        EndIf
    Else
        UpdateWorld
        RenderWorld
        Text(16, 16, "Level loaded  |  Esc quit")
    EndIf
    Flip
Wend
End
