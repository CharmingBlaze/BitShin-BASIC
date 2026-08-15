; Extra GLFW window sharing the G3N context.
; Main view + a second camera in another window (security-cam style).
;
; Esc          quit (main window)
; H            hide / show the extra window
; F            focus the extra window
; C            recreate the extra window if you closed it
; Extra window X button closes only that window, not the program.

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — main view")

cam = CreateCamera()
SetPosition(cam, 0, 2.2, -8)

side = CreateCamera()
SetPosition(side, 9, 3.5, 2)
PointEntity(side, 0, 0.5, 4)

light = CreateLight()
SetRotation(light, 50, 30, 0)

ground = CreatePlane()
SetScale(ground, 20, 1, 20)
SetEntityColor(ground, 42, 48, 58)

cube = CreateCube()
SetPosition(cube, 0, 0.6, 4)
SetEntityColor(cube, 70, 160, 255)

ball = CreateSphere()
SetPosition(ball, 3, 0.6, 5)
SetEntityColor(ball, 255, 110, 90)

win = CreateWindow(480, 360, "Security cam")
SetWindowCamera(win, side)
SetWindowTitle(win, "Security cam")

Print "Extra window handle=" + win + "  Esc=quit  H=hide  F=focus  C=recreate"

While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    TurnEntity(cube, 0.4 * dt, 0.9 * dt, 0)
    TurnEntity(ball, 0, -0.6 * dt, 0)

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
