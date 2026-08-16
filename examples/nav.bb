; Detour navmesh from real mesh triangles (AABB only if a mesh has no tris) plus grid A*.

Graphics3D(800, 600)
SetWindowTitle("BitShin BASIC — Nav")

camera = CreateCamera()
light = CreateLight()
SetRotation(light, 90, 0, 0)
SetPosition(camera, 0, 12, -16)
PointEntity(camera, 0, 0, 0)

floor = CreateCube()
SetScale(floor, 10, 0.2, 10)
SetPosition(floor, 0, 0, 0)
SetEntityColor(floor, 50, 70, 60)

block = CreateCube()
SetScale(block, 1.5, 1.5, 1.5)
SetPosition(block, 0, 1, 2)
SetEntityColor(block, 160, 60, 50)

body = CreateCube()
SetScale(body, 0.4, 0.8, 0.4)
SetPosition(body, -6, 1, -6)
SetEntityColor(body, 80, 160, 255)

nav = CreateNavMesh(floor)
AddNavObstacle(block)
BakeNavMesh(nav)
agent = CreateAgent(body)
SetAgentSpeed(agent, 6)
SetAgentDestination(agent, 6, 1, 6)

grid = CreateGrid(8, 8)
SetGridWalkable(grid, 3, 3, 0)
p = FindPath(grid, 0, 0, 7, 7)
Print("grid path", PathLength(p), PathX(p, 0), PathY(p, 0))

While Not KeyDown(1)
    UpdateWorld
    RenderWorld
    Flip
Wend
End
