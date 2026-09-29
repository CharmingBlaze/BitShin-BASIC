; Modern Language — Strict, maps, structs, Copy, states, and Try/Catch

Strict

; Typed locals and CreateMap / MapSet / MapGet
Local hp% = 3.9
Local name$ = "Ada"
Local m = CreateMap()
MapSet m, "hp", hp%
Print MapGet(m, "hp")
Print name$

; Struct handles: assignment shares; Copy snapshots
Struct Player
    Field hp
End Struct
Local hero = Player(hp%)
Local alias = hero
alias.hp = 1
Print hero.hp
Local snapshot = Copy(hero)
snapshot.hp = 9
Print hero.hp

; Typed function parameter
Function Double(n#)
    Return n# * 2
End Function
Print Double(21)

; State machine callback
Local sm = CreateStateMachine()
AddState sm, "idle", "OnIdle"
Function OnIdle()
    Print "idle"
End Function
UpdateState sm

; Try / Catch on a missing file
Try
    ReadText "no-such-file-modern.txt"
Catch err$
    Print "caught"
End Try
