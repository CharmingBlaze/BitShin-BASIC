; Nav — Detour navmesh from mesh triangles, plus a small grid A* path.
; Blue agent walks around the red obstacle. Esc quits.

Graphics3D(800, 600)
SetWindowTitle("BitShin BASIC — Nav")

; Camera / light
camera = CreateCamera().Position([0, 12, -16])
camera.Point(0, 0, 0)
light = CreateLight().Rotate(90, 0, 0)

; World
floor = CreateCube().Scale(10, 0.2, 10).Position([0, 0, 0]).Color(50, 70, 60)
block = CreateCube().Scale(1.5, 1.5, 1.5).Position([0, 1, 2]).Color(160, 60, 50)
body = CreateCube().Scale(0.4, 0.8, 0.4).Position([-6, 1, -6]).Color(80, 160, 255)

; Navmesh
nav = CreateNavMesh(floor)
AddNavObstacle(block)
BakeNavMesh(nav)
agent = CreateAgent(body)
SetAgentSpeed(agent, 6)
SetAgentDestination(agent, 6, 1, 6)

; Grid A*
grid = CreateGrid(8, 8)
SetGridWalkable(grid, 3, 3, 0)
p = FindPath(grid, 0, 0, 7, 7)
Print("grid path", PathLength(p), PathX(p, 0), PathY(p, 0))

; Loop
While Not KeyDown(1)
    UpdateWorld
    RenderWorld
    Flip
Wend
End
