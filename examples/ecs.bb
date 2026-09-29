; ECS Basics — Flecs entities, components, and queries (console)

EcsWorld()
Print("BitShin BASIC Flecs", EcsVersion$())

; Components
pos = EcsComponent("Position")
vel = EcsComponent("Velocity")

; Named entity with Position + Velocity
ship = EcsEntity("ship")
EcsSet(ship, pos, 1, 2, 3)
EcsSet(ship, vel, 0.5, 0, 0)
EcsName(ship, "ship")

; Second entity (string component name also works)
rock = EcsEntity("rock")
EcsSet(rock, "Position", 10, 0, 0)

; Query all Position entities
q = EcsQuery("Position")
Print("query", EcsQueryCount(q))
For i = 0 To EcsQueryCount(q) - 1
    e = EcsQueryEntity(q, i)
    Print(EcsName$(e), EcsGetX(e, pos), EcsGetY(e, pos), EcsGetZ(e, pos))
Next

Print("has vel", EcsHas(ship, vel), "count Position", EcsCount(pos))
Print("lookup", EcsLookup("ship"), "alive", EcsAlive(ship))

; One progress tick integrates Velocity into Position
EcsProgress(0.016)
Print("get x", EcsGet(ship, pos))

; Stress: many entities
For i = 1 To 400
    e = EcsEntity()
    EcsSet(e, pos, i, 0, 0)
    EcsSet(e, vel, 1, 0, 0)
Next
EcsProgress(0.016)
Print("stress Position", EcsCount(pos))
End
