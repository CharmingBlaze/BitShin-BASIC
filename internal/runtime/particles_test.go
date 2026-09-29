package runtime

import "testing"

func TestEmitterShapeNames(t *testing.T) {
	e := defaultEmitter(false)
	if e.style != 0 {
		t.Fatalf("gameplay emitter should start as a soft sprite, style %d", e.style)
	}
	applyEmitterShape(e, "Cube")
	if e.style != 3 {
		t.Fatalf("cube style %d", e.style)
	}
	applyEmitterShape(e, "streak")
	if e.style != 1 {
		t.Fatalf("streak style %d", e.style)
	}
	applyEmitterShape(e, "soft")
	if e.style != 0 {
		t.Fatalf("soft style %d", e.style)
	}
}

func TestEmitterSpawnAndTick2D(t *testing.T) {
	w := New(".")
	w.mode2D = true
	em := defaultEmitter(true)
	em.rate = 0
	em.max = 8
	em.life = 0.5
	em.speed = 10
	em.x, em.y = 100, 100
	w.emitters[1] = em
	w.spawnParticle(em)
	if len(em.parts) != 1 {
		t.Fatalf("expected 1 particle, got %d", len(em.parts))
	}
	w.tickEmitters(0.1)
	if len(em.parts) != 1 {
		t.Fatalf("particle died too soon: %d", len(em.parts))
	}
	if em.parts[0].x == 100 && em.parts[0].y == 100 {
		t.Fatal("particle did not move")
	}
	w.tickEmitters(1)
	if len(em.parts) != 0 {
		t.Fatalf("expected particle to expire, got %d", len(em.parts))
	}
}

func TestWeatherModes(t *testing.T) {
	w := New(".")
	w.setWeather("RAIN")
	if w.effectiveWeatherMode() != "rain" {
		t.Fatalf("mode %q", w.effectiveWeatherMode())
	}
	w.setWeather("nope")
	if w.effectiveWeatherMode() != "clear" {
		t.Fatalf("fallback %q", w.effectiveWeatherMode())
	}
	w.setWeatherIntensity(2)
	if w.wx.targetIntensity != 1 {
		t.Fatalf("intensity %v", w.wx.targetIntensity)
	}
}

func TestWeatherSnowFlakes(t *testing.T) {
	w := New(".")
	w.setWeatherIntensity(0.9)
	w.setWeather("snow")
	if w.effectiveWeatherMode() != "snow" {
		t.Fatalf("mode %q", w.effectiveWeatherMode())
	}
	found := false
	for _, id := range w.wx.ids {
		e := w.emitters[id]
		if e == nil || e.style != 2 || !e.recycle {
			continue
		}
		found = true
		if e.max < 280 {
			t.Fatalf("snow max too low: %d", e.max)
		}
		if e.size0 < 0.4 {
			t.Fatalf("snow flakes too small: %v", e.size0)
		}
		if e.alignY {
			t.Fatal("snow should not use rain streak align")
		}
		if len(e.parts) < 100 {
			t.Fatalf("expected seeded flakes, got %d", len(e.parts))
		}
	}
	if !found {
		t.Fatal("missing snow flake emitter")
	}
	w.setWeather("rain")
	for _, id := range w.wx.ids {
		e := w.emitters[id]
		if e != nil && e.style == 1 && e.recycle {
			t.Fatal("rain must stay original streaks (no flake recycle)")
		}
	}
}

func TestSpawnParticleRecyclesAtMax(t *testing.T) {
	w := New(".")
	w.mode2D = true
	em := defaultEmitter(true)
	em.rate = 0
	em.max = 3
	em.x, em.y = 10, 20
	w.emitters[1] = em
	for i := 0; i < 8; i++ {
		w.spawnParticle(em)
	}
	if len(em.parts) != em.max {
		t.Fatalf("expected %d particles at cap, got %d", em.max, len(em.parts))
	}
	if cap(em.parts) < em.max {
		t.Fatalf("recycle must keep backing array, cap=%d", cap(em.parts))
	}
}

func TestConeVelocity(t *testing.T) {
	vx, vy, vz := coneVelocity(0, 1, 0, 4, 0)
	if vy <= 0 {
		t.Fatalf("expected upward vel, got %v %v %v", vx, vy, vz)
	}
}
