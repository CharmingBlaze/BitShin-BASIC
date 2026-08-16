; Scene chunk — setup only, no Flip / End.
; Loaded by examples/scenes.bb via LoadScene.

floor = CreatePlane(20, 20)
SetPosition(floor, 0, 0, 6)
SetEntityColor(floor, 48, 56, 68)

box = CreateCube()
SetPosition(box, 0, 0.5, 6)
SetEntityColor(box, 80, 200, 120)
SetEntityName(box, "level_box")
