; Flecs 4.1.6 — entities, components, query. Progress also runs on Flip.

EcsWorld()
Print("BitShin BASIC Flecs", EcsVersion$())

pos = EcsComponent("Position")
vel = EcsComponent("Velocity")

ship = EcsEntity("ship")
EcsSet(ship, pos, 1, 2, 3)
EcsSet(ship, vel, 0.5, 0, 0)
EcsName(ship, "ship")

rock = EcsEntity("rock")
EcsSet(rock, "Position", 10, 0, 0)

q = EcsQuery("Position")
Print("query", EcsQueryCount(q))
For i = 0 To EcsQueryCount(q) - 1
    e = EcsQueryEntity(q, i)
    Print(EcsName$(e), EcsGetX(e, pos), EcsGetY(e, pos), EcsGetZ(e, pos))
Next

Print("has vel", EcsHas(ship, vel), "count Position", EcsCount(pos))
Print("lookup", EcsLookup("ship"), "alive", EcsAlive(ship))

EcsProgress(0.016)
Print("get x", EcsGet(ship, pos))

For i = 1 To 400
    e = EcsEntity()
    EcsSet(e, pos, i, 0, 0)
    EcsSet(e, vel, 1, 0, 0)
Next
EcsProgress(0.016)
Print("stress Position", EcsCount(pos))
End
