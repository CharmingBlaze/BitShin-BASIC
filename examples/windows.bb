; Windows — main view plus a second camera in an extra GLFW window.
; Esc quits (main). H hide/show, F focus, C recreate the extra window.
; Extra window X closes only that window, not the program.

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — main view")

; Cameras
cam = CreateCamera().Position([0, 2.2, -8])
side = CreateCamera().Position([9, 3.5, 2])
side.Point(0, 0.5, 4)

; Light / world
light = CreateLight().Rotate(50, 30, 0)
ground = CreatePlane().Scale(20, 1, 20).Color(42, 48, 58)
cube = CreateCube().Position([0, 0.6, 4]).Color(70, 160, 255)
ball = CreateSphere().Position([3, 0.6, 5]).Color(255, 110, 90)

; Extra window
win = CreateWindow(480, 360, "Security cam")
SetWindowCamera(win, side)
SetWindowTitle(win, "Security cam")

Print "Extra window handle=" + win + "  Esc=quit  H=hide  F=focus  C=recreate"

; Loop
While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    cube.Turn(0.4 * dt, 0.9 * dt, 0)
    ball.Turn(0, -0.6 * dt, 0)

    If KeyHit(KEY_H) Then
        If WindowClosed(win) Then
            ; already gone
        Else
            hide = 1 - hide
            If hide Then HideWindow(win) Else ShowWindow(win)
        EndIf
    EndIf
    If KeyHit(KEY_F) Then ActivateWindow(win)
    If KeyHit(KEY_C) Then
        If WindowClosed(win) Or win = 0 Then
            win = CreateWindow(480, 360, "Security cam")
            SetWindowCamera(win, side)
            hide = 0
        EndIf
    EndIf

    UpdateWorld
    RenderWorld
    Text(12, 12, "Main  |  Esc quit  H hide  F focus  C recreate")
    Text(12, 32, "Extra closed=" + WindowClosed(win) + "  size=" + GetWindowWidth(win) + "x" + GetWindowHeight(win))
    Flip
Wend
End
