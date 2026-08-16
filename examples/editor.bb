; In-engine ImGui: scene list, transform, material, profiler.

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Editor")
cam = CreateCamera()
SetPosition(cam, 0, 3, -8)
sun = CreateLight()
SetRotation(sun, 60, 20, 0)

cube = CreateCube()
SetPosition(cube, 0, 0, 4)
SetEntityColor(cube, 70, 160, 255)
NameEntity(cube, "cube")

sel = cube
amb# = 80
shine# = 8

While Not KeyDown(1)
    GuiBegin("Scene")
    GuiText("entities " + Str(EntityCount()) + "  fps " + Str(Int(StatsFPS())))
    GuiText("draws " + Str(StatsDraws()) + "  chunks " + Str(StatsChunks()) + "  jobs " + Str(StatsJobs()))
    GuiSeparator()
    n = EntityCount()
    For i = 1 To n
        e = EntityByIndex(i)
        label$ = EntityName$(e)
        If label$ = "" Then label$ = "ent " + Str(e)
        If GuiButton(label$) Then sel = e
    Next
    GuiEnd()

    GuiBegin("Inspector")
    GuiText("id " + Str(sel) + " " + EntityName$(sel))
    px# = GuiSlider("X", -10, 10, EntityX(sel))
    py# = GuiSlider("Y", -5, 8, EntityY(sel))
    pz# = GuiSlider("Z", -4, 16, EntityZ(sel))
    SetPosition(sel, px#, py#, pz#)
    GuiSeparator()
    amb# = GuiSlider("Ambient", 0, 255, amb#)
    shine# = GuiSlider("Shininess", 0, 64, shine#)
    EntityShininess(sel, shine#)
    EntityColor(sel, amb#, 160, 255)
    If GuiButton("Spin") Then TurnEntity(sel, 0, 15, 0)
    GuiEnd()

    TurnEntity(cube, 0, 0.3, 0)
    RenderWorld
    Flip
Wend
End
