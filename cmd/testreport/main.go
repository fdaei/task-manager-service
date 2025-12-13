package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type testEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test,omitempty"`
	Elapsed float64 `json:"Elapsed,omitempty"`
	Output  string  `json:"Output,omitempty"`
}

type testCase struct {
	Name    string
	Status  string
	Elapsed float64
	Output  []string
}

type packageResult struct {
	Name    string
	Status  string
	Elapsed float64
	Tests   map[string]*testCase
	Output  []string
	Passed  int
	Failed  int
	Skipped int
}

type report struct {
	Packages    map[string]*packageResult
	Passed      int
	Failed      int
	Skipped     int
	Total       int
	Command     []string
	LogLines    []string
	ParseErrors []string
	Stderr      []string
	Duration    time.Duration
	GeneratedAt time.Time
	ExitCode    int
	Coverage    string
	CovPercent  float64
	CovStatus   string
	CovThresh   float64
	CovProfile  string
	CovDetails  []string
}

type templateData struct {
	Packages    []packageView
	Passed      int
	Failed      int
	Skipped     int
	Total       int
	Duration    string
	Command     string
	Logs        string
	ParseErrors []string
	Stderr      []string
	GeneratedAt string
	ExitCode    int
	Coverage    string
	CovPercent  float64
	CovStatus   string
	CovThresh   float64
	CovProfile  string
	CovDetails  []string
}

type packageView struct {
	Name     string
	Status   string
	Duration string
	Passed   int
	Failed   int
	Skipped  int
	Tests    []testView
	Output   []string
}

type testView struct {
	Name     string
	Status   string
	Duration string
	Output   []string
}

func main() {
	outputPath := flag.String("o", "test_report.html", "path to write the HTML report")
	packages := flag.String("packages", "./...", "packages to test (passed to go test)")
	coverProfile := flag.String("coverprofile", "coverage.out", "path to write coverage profile")
	threshold := flag.Float64("threshold", 75.0, "minimum coverage percentage expected")
	flag.Parse()

	pkgArgs := strings.Fields(*packages)
	if len(pkgArgs) == 0 {
		pkgArgs = []string{"./..."}
	}

	if err := os.Remove(*coverProfile); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: could not clear previous coverage profile: %v\n", err)
	}

	args := []string{
		"go", "test", "-json",
		"-covermode=atomic",
		fmt.Sprintf("-coverprofile=%s", *coverProfile),
	}
	args = append(args, pkgArgs...)

	rep, err := runAndCollect(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while running tests: %v\n", err)
	}

	rep.CovProfile = *coverProfile
	rep.CovThresh = *threshold
	if covPct, lines, covErr := readCoverage(*coverProfile); covErr != nil {
		rep.ParseErrors = append(rep.ParseErrors, fmt.Sprintf("coverage: %v", covErr))
		rep.CovStatus = "unknown"
	} else {
		rep.CovPercent = covPct
		rep.CovDetails = lines
		if covPct >= *threshold {
			rep.CovStatus = "pass"
		} else {
			rep.CovStatus = "fail"
		}
		rep.Coverage = fmt.Sprintf("%.1f%%", covPct)
	}

	rep.Total = rep.Passed + rep.Failed + rep.Skipped
	data := toTemplateData(rep)

	if err := writeReport(*outputPath, data); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write report: %v\n", err)
		os.Exit(1)
	}

	if rep.ExitCode != 0 || rep.Failed > 0 || rep.CovStatus == "fail" {
		os.Exit(1)
	}
}

func runAndCollect(cmdArgs []string) (*report, error) {
	rep := &report{
		Packages:    make(map[string]*packageResult),
		Command:     cmdArgs,
		GeneratedAt: time.Now(),
		ExitCode:    0,
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return rep, fmt.Errorf("attach stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return rep, fmt.Errorf("attach stderr: %w", err)
	}

	var stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, copyErr := io.Copy(&stderrBuf, stderr); copyErr != nil {
			rep.ParseErrors = append(rep.ParseErrors, fmt.Sprintf("stderr read error: %v", copyErr))
		}
	}()

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return rep, fmt.Errorf("start go test: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		var ev testEvent
		if unmarshalErr := json.Unmarshal([]byte(line), &ev); unmarshalErr != nil {
			rep.ParseErrors = append(rep.ParseErrors, fmt.Sprintf("could not parse line: %v", unmarshalErr))
			continue
		}
		handleEvent(rep, ev)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		rep.ParseErrors = append(rep.ParseErrors, fmt.Sprintf("read stdout: %v", scanErr))
	}

	waitErr := cmd.Wait()
	wg.Wait()
	rep.Duration = time.Since(start)

	if stderrBuf.Len() > 0 {
		rep.Stderr = append(rep.Stderr, strings.Split(strings.TrimRight(stderrBuf.String(), "\n"), "\n")...)
	}

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			rep.ExitCode = exitErr.ExitCode()
		} else {
			rep.ExitCode = 1
		}
	}

	for _, pkg := range rep.Packages {
		if pkg.Status == "" {
			switch {
			case pkg.Failed > 0:
				pkg.Status = "fail"
			case pkg.Skipped > 0 && pkg.Passed == 0:
				pkg.Status = "skip"
			default:
				pkg.Status = "pass"
			}
		}
	}

	return rep, waitErr
}

func handleEvent(rep *report, ev testEvent) {
	if ev.Action == "output" {
		line := strings.TrimSuffix(ev.Output, "\n")
		rep.LogLines = append(rep.LogLines, line)
	}

	if ev.Package == "" {
		return
	}

	pkg := ensurePackage(rep, ev.Package)

	switch ev.Action {
	case "run":
		if ev.Test != "" {
			_ = ensureTest(pkg, ev.Test)
		}
	case "output":
		line := strings.TrimSuffix(ev.Output, "\n")
		if ev.Test != "" {
			tc := ensureTest(pkg, ev.Test)
			tc.Output = append(tc.Output, line)
		} else {
			pkg.Output = append(pkg.Output, line)
		}
	case "pass", "fail", "skip":
		if ev.Test != "" {
			tc := ensureTest(pkg, ev.Test)
			tc.Status = ev.Action
			if ev.Elapsed > 0 {
				tc.Elapsed = ev.Elapsed
			}
			switch ev.Action {
			case "pass":
				rep.Passed++
				pkg.Passed++
			case "fail":
				rep.Failed++
				pkg.Failed++
				pkg.Status = "fail"
			case "skip":
				rep.Skipped++
				pkg.Skipped++
			}
		} else {
			if ev.Elapsed > 0 {
				pkg.Elapsed = ev.Elapsed
			}
			pkg.Status = ev.Action
		}
	}
}

func ensurePackage(rep *report, name string) *packageResult {
	if rep.Packages[name] == nil {
		rep.Packages[name] = &packageResult{
			Name:   name,
			Tests:  make(map[string]*testCase),
			Status: "",
		}
	}
	return rep.Packages[name]
}

func ensureTest(pkg *packageResult, name string) *testCase {
	if pkg.Tests[name] == nil {
		pkg.Tests[name] = &testCase{Name: name}
	}
	return pkg.Tests[name]
}

func toTemplateData(rep *report) templateData {
	pkgs := make([]packageView, 0, len(rep.Packages))
	for _, pkg := range rep.Packages {
		tests := make([]testView, 0, len(pkg.Tests))
		for _, tc := range pkg.Tests {
			tests = append(tests, testView{
				Name:     tc.Name,
				Status:   tc.Status,
				Duration: formatElapsed(tc.Elapsed),
				Output:   tc.Output,
			})
		}
		sort.Slice(tests, func(i, j int) bool {
			if tests[i].Status != tests[j].Status {
				return statusOrder(tests[i].Status) < statusOrder(tests[j].Status)
			}
			return tests[i].Name < tests[j].Name
		})
		pkgs = append(pkgs, packageView{
			Name:     pkg.Name,
			Status:   pkg.Status,
			Duration: formatElapsed(pkg.Elapsed),
			Passed:   pkg.Passed,
			Failed:   pkg.Failed,
			Skipped:  pkg.Skipped,
			Tests:    tests,
			Output:   pkg.Output,
		})
	}

	sort.Slice(pkgs, func(i, j int) bool {
		if pkgs[i].Status != pkgs[j].Status {
			return statusOrder(pkgs[i].Status) < statusOrder(pkgs[j].Status)
		}
		return pkgs[i].Name < pkgs[j].Name
	})

	return templateData{
		Packages:    pkgs,
		Passed:      rep.Passed,
		Failed:      rep.Failed,
		Skipped:     rep.Skipped,
		Total:       rep.Total,
		Duration:    rep.Duration.Round(time.Millisecond).String(),
		Command:     strings.Join(rep.Command, " "),
		Logs:        strings.Join(rep.LogLines, "\n"),
		ParseErrors: rep.ParseErrors,
		Stderr:      rep.Stderr,
		GeneratedAt: rep.GeneratedAt.Format(time.RFC1123),
		ExitCode:    rep.ExitCode,
		Coverage:    rep.Coverage,
		CovPercent:  rep.CovPercent,
		CovStatus:   rep.CovStatus,
		CovThresh:   rep.CovThresh,
		CovProfile:  rep.CovProfile,
		CovDetails:  rep.CovDetails,
	}
}

func statusOrder(status string) int {
	switch status {
	case "fail":
		return 0
	case "skip":
		return 1
	case "pass":
		return 2
	default:
		return 3
	}
}

func formatElapsed(elapsed float64) string {
	if elapsed <= 0 {
		return ""
	}
	return fmt.Sprintf("%.3fs", elapsed)
}

func writeReport(path string, data templateData) error {
	tpl := template.Must(template.New("page").Parse(pageTemplate))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func readCoverage(profile string) (float64, []string, error) {
	if profile == "" {
		return 0, nil, fmt.Errorf("coverage profile path is empty")
	}
	if _, err := os.Stat(profile); err != nil {
		return 0, nil, fmt.Errorf("coverage profile not found: %w", err)
	}
	cmd := exec.Command("go", "tool", "cover", fmt.Sprintf("-func=%s", profile))
	out, err := cmd.CombinedOutput()
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if err != nil {
		return 0, lines, fmt.Errorf("go tool cover: %w: %s", err, string(out))
	}

	var (
		total float64
		found bool
	)

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "total:" && len(fields) >= 3 {
			percStr := strings.TrimSuffix(fields[len(fields)-1], "%")
			val, parseErr := strconv.ParseFloat(percStr, 64)
			if parseErr != nil {
				return 0, lines, fmt.Errorf("parse coverage percent from %q: %w", line, parseErr)
			}
			total = val
			found = true
			break
		}
	}

	if !found {
		return 0, lines, fmt.Errorf("could not locate total coverage line")
	}

	return total, lines, nil
}

const pageTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Go Test Report</title>
  <style>
    :root {
      --pass: #0f9d58;
      --fail: #d93025;
      --skip: #f9ab00;
      --bg: #0b1021;
      --panel: #121932;
      --text: #e5e7f0;
      --muted: #94a3b8;
      --border: #22304f;
      --accent: #4f46e5;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Inter", "Segoe UI", system-ui, -apple-system, sans-serif;
      background: radial-gradient(circle at 20% 20%, #1b2342 0, #0b1021 50%);
      color: var(--text);
      padding: 24px;
    }
    h1 {
      margin: 0 0 16px;
      font-weight: 700;
      letter-spacing: -0.5px;
    }
    .meta { color: var(--muted); margin-bottom: 16px; }
    .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-bottom: 20px; }
    .card {
      padding: 14px;
      border: 1px solid var(--border);
      border-radius: 12px;
      background: linear-gradient(135deg, rgba(79,70,229,0.08), rgba(79,70,229,0.02));
    }
    .card .label { color: var(--muted); font-size: 13px; }
    .card .value { font-size: 22px; font-weight: 700; margin-top: 4px; }
    .subtext { font-size: 12px; color: var(--muted); margin-top: 6px; display: block; }
    .pill { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 12px; font-weight: 600; }
    .pill.pass { background: rgba(15,157,88,0.15); color: var(--pass); }
    .pill.fail { background: rgba(217,48,37,0.15); color: var(--fail); }
    .pill.skip { background: rgba(249,171,0,0.15); color: var(--skip); }
    details { border: 1px solid var(--border); border-radius: 12px; margin-bottom: 12px; background: var(--panel); }
    summary { padding: 12px 14px; cursor: pointer; display: flex; align-items: center; gap: 8px; font-weight: 600; }
    summary:hover { background: rgba(79,70,229,0.08); }
    .pkg-name { flex: 1; }
    .badge { padding: 4px 8px; border-radius: 8px; font-size: 12px; font-weight: 600; border: 1px solid var(--border); color: var(--muted); }
    .badge.pass { border-color: rgba(15,157,88,0.4); color: var(--pass); }
    .badge.fail { border-color: rgba(217,48,37,0.4); color: var(--fail); }
    .badge.skip { border-color: rgba(249,171,0,0.4); color: var(--skip); }
    .duration { color: var(--muted); font-size: 12px; }
    .test { border-top: 1px solid var(--border); }
    .test summary { font-weight: 500; }
    pre {
      margin: 0;
      padding: 12px 14px;
      background: #0d1328;
      color: #e2e8f0;
      overflow: auto;
      border-top: 1px solid var(--border);
      border-radius: 0 0 12px 12px;
    }
    .logs, .errors { margin-top: 18px; }
    .section-title { font-size: 15px; color: var(--muted); margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.04em; }
    .alert { border: 1px solid var(--border); border-radius: 10px; padding: 12px; background: rgba(217,48,37,0.08); color: var(--text); }
    code { font-family: "JetBrains Mono", "SFMono-Regular", Menlo, monospace; }
  </style>
</head>
<body>
  <h1>Go Test Report</h1>
  <div class="meta">Generated at {{.GeneratedAt}} &middot; Duration {{.Duration}} &middot; Exit code {{.ExitCode}} &middot; Command: <code>{{.Command}}</code></div>
  <div class="cards">
    <div class="card">
      <div class="label">Total tests</div>
      <div class="value">{{.Total}}</div>
    </div>
    <div class="card">
      <div class="label">Passed</div>
      <div class="value"><span class="pill pass">{{.Passed}}</span></div>
    </div>
    <div class="card">
      <div class="label">Failed</div>
      <div class="value"><span class="pill fail">{{.Failed}}</span></div>
    </div>
    <div class="card">
      <div class="label">Skipped</div>
      <div class="value"><span class="pill skip">{{.Skipped}}</span></div>
    </div>
    <div class="card">
      <div class="label">Coverage</div>
      <div class="value">
        {{if .Coverage}}
          {{if eq .CovStatus "fail"}}<span class="pill fail">{{.Coverage}}</span>{{else if eq .CovStatus "pass"}}<span class="pill pass">{{.Coverage}}</span>{{else}}<span class="pill skip">{{.Coverage}}</span>{{end}}
        {{else}}
          <span class="pill skip">N/A</span>
        {{end}}
        <span class="subtext">Threshold {{printf "%.1f%%" .CovThresh}}</span>
      </div>
    </div>
  </div>

  {{range .Packages}}
    <details open>
      <summary>
        <span class="pkg-name">{{.Name}}</span>
        <span class="badge {{.Status}}">{{.Status}}</span>
        {{if .Duration}}<span class="badge">{{.Duration}}</span>{{end}}
        <span class="badge pass">+{{.Passed}}</span>
        <span class="badge fail">-{{.Failed}}</span>
        {{if .Skipped}}<span class="badge skip">{{.Skipped}} skipped</span>{{end}}
      </summary>
      {{range .Tests}}
        <details class="test" {{if eq .Status "fail"}}open{{end}}>
          <summary>
            <span class="pkg-name">{{.Name}}</span>
            <span class="badge {{.Status}}">{{.Status}}</span>
            {{if .Duration}}<span class="duration">{{.Duration}}</span>{{end}}
          </summary>
          {{if .Output}}
          <pre>{{range .Output}}{{.}}{{"\n"}}{{end}}</pre>
          {{end}}
        </details>
      {{end}}
      {{if .Output}}
        <div class="section-title" style="padding: 0 14px;">Package output</div>
        <pre>{{range .Output}}{{.}}{{"\n"}}{{end}}</pre>
      {{end}}
    </details>
  {{end}}

  {{if .CovDetails}}
  <div class="logs">
    <div class="section-title">Coverage ({{.CovProfile}})</div>
    <pre>{{range .CovDetails}}{{.}}{{"\n"}}{{end}}</pre>
  </div>
  {{end}}

  <div class="logs">
    <div class="section-title">Test output</div>
    <pre>{{.Logs}}</pre>
  </div>

  {{if .Stderr}}
  <div class="errors">
    <div class="section-title">Stderr</div>
    <pre>{{range .Stderr}}{{.}}{{"\n"}}{{end}}</pre>
  </div>
  {{end}}

  {{if .ParseErrors}}
  <div class="errors">
    <div class="section-title">Parse issues</div>
    <div class="alert">
      {{range .ParseErrors}}• {{.}}<br>{{end}}
    </div>
  </div>
  {{end}}
</body>
</html>
`
