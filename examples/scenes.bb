; Menu / level pattern: ClearWorld then LoadScene a setup .bb
; Space loads the level chunk. Esc quits.

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — scenes")

cam = CreateCamera()
SetPosition(cam, 0, 1.6, 0)
light = CreateLight()
SetRotation(light, 40, 20, 0)

phase = 0

While Not KeyDown(KEY_ESCAPE)
    If phase = 0 Then
        RenderWorld
        Text(16, 16, "Menu  |  Space = LoadScene level_setup.bb")
        If KeyHit(KEY_SPACE) Then
            ClearWorld()
            cam = CreateCamera()
            SetPosition(cam, 0, 1.6, -2)
            light = CreateLight()
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
