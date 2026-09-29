; GUI demo — Dear ImGui on the same G3N GLFW window as the 3D scene.
; Spin / wireframe / rename the cube. Esc quits.

Graphics3D(800, 600)
SetWindowTitle("BitShin BASIC — GUI demo")
SetCameraClsColor(18, 22, 32)
SetAmbientLight(70, 80, 100)

; Camera / light
camera = CreateCamera().Position([0, 2, -6])
camera.Point(0, 0, 0)

light = CreateLight().Rotate(50, 30, 0)

; World
ground = CreatePlane(16, 16).Position([0, -1, 0]).Color(36, 42, 52)
cube = CreateCube().Position([0, 0.2, 0]).Color(70, 160, 255)

name$ = "cube"
spin# = 1

; Loop
While Not KeyDown(1)
    dt# = DeltaTime() * 60

    GuiBegin("Cube")
    GuiText("Same GLFW window as Graphics3D")
    GuiSeparator()
    spin# = GuiSlider("Spin", 0, 5, spin#)
    wire = GuiCheckbox("Wireframe")
    name$ = GuiInputText("Name", name$)
    GuiSameLine()
    If GuiButton("Reset") Then
        spin# = 1
        cube.Position(0, 0.2, 0)
    EndIf
    GuiEnd()

    Wireframe(wire)
    cube.Turn(0.4 * dt * spin#, 1 * dt * spin#, 0)
    cube.Name(name$)

    If WantCaptureMouse() Then
        ; mouse is on the panel — skip game clicks
    EndIf

    RenderWorld
    Flip
Wend
End
