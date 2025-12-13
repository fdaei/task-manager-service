package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunAndCollectAggregatesEvents(t *testing.T) {
	script := `cat <<'EOF'
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"pkg/a","Test":"TestOK"}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"pkg/a","Test":"TestOK","Output":"--- PASS: TestOK (0.00s)\n"}
{"Time":"2024-01-01T00:00:00Z","Action":"pass","Package":"pkg/a","Test":"TestOK","Elapsed":0.01}
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"pkg/a","Test":"TestFail"}
{"Time":"2024-01-01T00:00:00Z","Action":"fail","Package":"pkg/a","Test":"TestFail","Elapsed":0.02}
{"Time":"2024-01-01T00:00:00Z","Action":"output","Package":"pkg/a","Output":"ok\tpkg/a\t0.03s\n"}
EOF`

	rep, err := runAndCollect([]string{"bash", "-c", script})
	if err != nil {
		t.Fatalf("runAndCollect error: %v", err)
	}

	if rep.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", rep.ExitCode)
	}

	pkg := rep.Packages["pkg/a"]
	if pkg == nil {
		t.Fatalf("package result not found")
	}
	if pkg.Status != "fail" {
		t.Fatalf("expected pkg status fail, got %s", pkg.Status)
	}
	if rep.Passed != 1 || rep.Failed != 1 {
		t.Fatalf("unexpected totals: passed=%d failed=%d", rep.Passed, rep.Failed)
	}
	if len(pkg.Tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(pkg.Tests))
	}
	if len(rep.LogLines) == 0 {
		t.Fatalf("expected log lines to be captured")
	}
}

func TestReadCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "cov.out")

	cmd := exec.Command("go", "test", "-coverprofile", path, "./internal/config")
	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(tmpDir, ".gocache"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test for coverage failed: %v\n%s", err, string(out))
	}

	percent, lines, err := readCoverage(path)
	if err != nil {
		t.Fatalf("readCoverage error: %v", err)
	}
	if percent <= 0 {
		t.Fatalf("expected positive coverage percent, got %v", percent)
	}
	if len(lines) == 0 {
		t.Fatalf("expected coverage lines")
	}

	if _, _, err := readCoverage(filepath.Join(tmpDir, "missing.out")); err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestTemplateAndWriteReport(t *testing.T) {
	rep := &report{
		Packages: map[string]*packageResult{
			"pkg/b": {
				Name:    "pkg/b",
				Status:  "pass",
				Passed:  1,
				Failed:  0,
				Skipped: 0,
				Elapsed: 1.2,
				Tests: map[string]*testCase{
					"TestBeta":  {Name: "TestBeta", Status: "pass", Elapsed: 0.5},
					"TestAlpha": {Name: "TestAlpha", Status: "fail", Elapsed: 0.3},
				},
				Output: []string{"ok"},
			},
		},
		Passed:      1,
		Failed:      1,
		Skipped:     0,
		Total:       2,
		Duration:    1500 * time.Millisecond,
		Command:     []string{"go", "test"},
		LogLines:    []string{"log1", "log2"},
		ParseErrors: []string{},
		GeneratedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		CovPercent:  80.5,
		CovStatus:   "pass",
		CovThresh:   75,
		CovProfile:  "coverage.out",
		CovDetails:  []string{"total: (statements) 80.5%"},
	}

	data := toTemplateData(rep)
	if data.Total != 2 || data.Passed != 1 || data.Failed != 1 {
		t.Fatalf("unexpected totals in template data: %+v", data)
	}
	if len(data.Packages) != 1 {
		t.Fatalf("expected one package view")
	}
	// ensure ordering put fail before pass inside test list
	tests := data.Packages[0].Tests
	if tests[0].Status != "fail" {
		t.Fatalf("expected failing test first, got %+v", tests)
	}
	if data.Duration == "" || !strings.Contains(data.Command, "go test") {
		t.Fatalf("unexpected command/duration in template data: %+v", data)
	}

	outFile := filepath.Join(t.TempDir(), "report.html")
	if err := writeReport(outFile, data); err != nil {
		t.Fatalf("writeReport error: %v", err)
	}
	bytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(bytes), "Go Test Report") {
		t.Fatalf("report content missing title")
	}
}
