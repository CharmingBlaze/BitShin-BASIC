; Dear ImGui on the same G3N GLFW window. Drawn after the 3D scene on Flip.

Graphics3D(800, 600)
SetWindowTitle("GUI demo")

camera = CreateCamera()
light = CreateLight()
SetRotation(light, 90, 0, 0)

cube = CreateCube()
SetPosition(cube, 0, 0, 5)
SetEntityColor(cube, 70, 160, 255)

name$ = "cube"
spin# = 1

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
        SetPosition(cube, 0, 0, 5)
    EndIf
    GuiEnd()

    Wireframe(wire)
    TurnEntity(cube, 0.4 * dt * spin#, 1 * dt * spin#, 0)
    SetEntityName(cube, name$)

    If WantCaptureMouse() Then
        ; mouse is on the panel — skip game clicks
    EndIf

    RenderWorld
    Flip
Wend
End
