package main

import (
	"strings"
	"testing"
)

func TestParseFrameLog(t *testing.T) {
	l := parseFrameLog(strings.NewReader("gpu AMD Radeon 780M Graphics\ngl 3.3.0 Core\n16.5 high 1 0\nnot a frame\n20 medium 0.75 2\n"))
	if l.gpu != "AMD Radeon 780M Graphics" || l.gl != "3.3.0 Core" {
		t.Fatalf("got %q, %q", l.gpu, l.gl)
	}
	if len(l.frames) != 2 || l.frames[1] != (benchFrame{20, "medium", 0.75, 2}) {
		t.Fatalf("got %+v", l.frames)
	}
}

func frames(ms ...float64) []benchFrame {
	out := make([]benchFrame, len(ms))
	for i, m := range ms {
		out[i] = benchFrame{ms: m, tier: "high", scale: 1}
	}
	return out
}

func TestSummarizeLeavesOutTheWarmup(t *testing.T) {
	// 1000 ms of a load stall, then steady frames.
	s := summarize(frames(1000, 10, 10, 10, 10), 1)
	if s.worst != 10 || s.frames != 4 || s.avg != 10 || s.hitches != 0 {
		t.Fatalf("got %+v", s)
	}
}

func TestSummarizeKeepsAVerySlowRunWhole(t *testing.T) {
	s := summarize(frames(500, 500), 5)
	if s.frames != 2 || s.avg != 500 || s.hitches != 2 {
		t.Fatalf("got %+v", s)
	}
}

func TestSummarizeTiersAndErrors(t *testing.T) {
	f := []benchFrame{{10, "high", 1, 0}, {50, "medium", 0.7, 3}}
	s := summarize(f, 0)
	if s.startTier != "high" || s.endTier != "medium" || s.endScale != 0.7 || s.glErrors != 3 || s.hitches != 1 {
		t.Fatalf("got %+v", s)
	}
	if got := summarize(nil, 5); got.frames != 0 {
		t.Fatalf("no frames should be empty, got %+v", got)
	}
}

func TestPercentile(t *testing.T) {
	v := []float64{1, 2, 3, 4, 5}
	if percentile(v, 0.5) != 3 || percentile(v, 0) != 1 || percentile(v, 1) != 5 {
		t.Fatal("wrong percentile")
	}
	if percentile(nil, 0.5) != 0 {
		t.Fatal("empty should be 0")
	}
}

func TestBestBenchTier(t *testing.T) {
	r := func(name string, p95 float64) benchResult {
		return benchResult{name: name, summary: benchSummary{frames: 100, p95: p95}}
	}
	if n, how := bestBenchTier([]benchResult{r("low", 9), r("medium", 15), r("high", 25), r("auto", 1)}); n != "medium" || how != "steady" {
		t.Fatalf("got %s, %s", n, how)
	}
	if n, how := bestBenchTier([]benchResult{r("low", 28), r("medium", 50)}); n != "low" || how != "playable, not steady" {
		t.Fatalf("got %s, %s", n, how)
	}
	if n, _ := bestBenchTier([]benchResult{r("low", 90)}); n != "" {
		t.Fatalf("got %s", n)
	}
	failed := r("high", 5)
	failed.failure = "crashed"
	if n, _ := bestBenchTier([]benchResult{r("low", 10), failed}); n != "low" {
		t.Fatalf("a failed tier should not win, got %s", n)
	}
}

func TestBenchReportShowsFailures(t *testing.T) {
	report := benchReport(benchLog{gpu: "Test GPU"}, []benchResult{
		{name: "low", summary: benchSummary{frames: 10, median: 8, p95: 9}},
		{name: "high", failure: "shader error\nexit status 1"},
	}, 30)
	for _, want := range []string{"Test GPU", "low", "high    FAILED: shader error", "Best tier: low (steady)", "exit status 1"} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q:\n%s", want, report)
		}
	}
}

func TestBenchSceneHasItsPlaceholders(t *testing.T) {
	for _, p := range []string{"@SETUP@", "@NAME@", "@SECONDS@"} {
		if !strings.Contains(benchScene, p) {
			t.Errorf("scene lacks %s", p)
		}
	}
}
