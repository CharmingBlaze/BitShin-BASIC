; Flecs stress: thousands of Position+Velocity entities. Progress integrates.

EcsWorld()
pos = EcsComponent("Position")
vel = EcsComponent("Velocity")

n = 2000
For i = 1 To n
    e = EcsEntity()
    EcsSet(e, pos, Rnd(100), Rnd(10), Rnd(100))
    EcsSet(e, vel, Rnd(2) - 1, 0, Rnd(2) - 1)
Next

q = EcsQuery("Position, Velocity")
Print("ecs crowd", EcsQueryCount(q), "Flecs", EcsVersion$())

For f = 1 To 30
    EcsProgress(0.016)
Next

e0 = EcsQueryEntity(q, 0)
Print("moved x", EcsGetX(e0, pos), "count", EcsCount(pos))
End
