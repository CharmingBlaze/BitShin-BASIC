; Level Setup — scene chunk for LoadScene (no Flip / End)
; Loaded by examples/scenes.bb via LoadScene.

; Floor plane
floor = CreatePlane(20, 20)
SetPosition(floor, 0, 0, 6)
SetEntityColor(floor, 48, 56, 68)

; Named box — SetEntityName for lookup from the host scene
box = CreateCube()
SetPosition(box, 0, 0.5, 6)
SetEntityColor(box, 80, 200, 120)
SetEntityName(box, "level_box")
