; Editor — ImGui scene list, transform sliders, material, and profiler stats.
; Esc quits.

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Editor")

; Camera / light
cam = CreateCamera().Position([0, 3, -8])
sun = CreateLight().Rotate(60, 20, 0)

; World
cube = CreateCube().Position([0, 0, 4]).Color(70, 160, 255).Name("cube")

sel = cube
amb# = 80
shine# = 8

; Loop
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
    sel.Position(px#, py#, pz#)
    GuiSeparator()
    amb# = GuiSlider("Ambient", 0, 255, amb#)
    shine# = GuiSlider("Shininess", 0, 64, shine#)
    sel.Shininess(shine#)
    sel.Color(amb#, 160, 255)
    If GuiButton("Spin") Then sel.Turn(0, 15, 0)
    GuiEnd()

    cube.Turn(0, 0.3, 0)
    RenderWorld
    Flip
Wend
End
