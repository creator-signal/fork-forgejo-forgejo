package main

import (
	"encoding/json" //nolint:depguard
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"slices"
	"strconv"
	"time"
)

func main() {
	if err := run(os.Stdin, os.Getenv("TESTJSON_FORMAT")); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(r io.Reader, mode string) error {
	switch mode {
	case "output":
		return runOutput(r)
	case "summary":
		return runSummary(r)
	default:
		return fmt.Errorf("unsupported TESTJSON_FORMAT: %q (expected output or summary)", mode)
	}
}

func runOutput(r io.Reader) error {
	failed := make(map[string]bool)

	var fail, pass, skip int
	for ev, err := range iterEvents(r) {
		if err != nil {
			return err
		}
		switch ev.Action {
		case "pass":
			if ev.Test != "" { // ignore the package-level "pass" action
				pass++
			}
		case "skip":
			skip++
		case "fail":
			if ev.Test == "" {
				if failed[ev.Package] {
					// ignore the package-level "fail" action
					continue
				}
			}
			failed[ev.Package] = true
			fail++
		}
		fmt.Print(ev.Output)
	}
	result := resultLine(pass, skip, fail)
	if fail > 0 || pass == 0 {
		// consider the absence of tests to be a failure
		return errors.New(result)
	}
	fmt.Println(result)
	return nil
}

func resultLine(pass, skip, fail int) string {
	result := "TESTJSON " + strconv.Itoa(pass) + " passed"
	if skip > 0 {
		result += ", " + strconv.Itoa(skip) + " skipped"
	}
	if fail > 0 {
		result += ", " + strconv.Itoa(fail) + " failed"
	}
	return result
}

type summary struct {
	skip, pass int
	failures   []string
}

func runSummary(r io.Reader) error {
	reduced := make(map[string]summary)
	var failedPackages []string

	var fail, pass, skip int
	for ev, err := range iterEvents(r) {
		if err != nil {
			return err
		}
		switch ev.Action {
		case "start", "run", "output", "pause", "cont", "build-output":
		case "pass":
			if ev.Test != "" { // ignore the package-level "pass" action
				pass++
				summary := reduced[ev.Package]
				summary.pass++
				reduced[ev.Package] = summary
			}
		case "skip":
			skip++
			summary := reduced[ev.Package]
			summary.skip++
			reduced[ev.Package] = summary
		case "build-fail":
			// handled below
		case "fail":
			summary := reduced[ev.Package]
			msg := ev.Test
			if msg == "" {
				if len(summary.failures) > 0 {
					// ignore the package-level "fail" action
					continue
				}
				msg = "[build failed]"
			}
			if len(summary.failures) == 0 {
				failedPackages = append(failedPackages, ev.Package)
			}
			fail++
			summary.failures = append(summary.failures, msg)
			reduced[ev.Package] = summary
		default:
			return fmt.Errorf("unexpected action: %s", ev.Action)
		}
	}

	slices.Sort(failedPackages)
	for _, path := range failedPackages {
		fmt.Println(path)
		for _, f := range reduced[path].failures {
			fmt.Println("\t" + f)
		}
	}
	result := resultLine(pass, skip, fail)
	if fail > 0 || pass == 0 {
		return errors.New(result)
	}
	fmt.Println(result)
	return nil
}

// as emitted by go test -json
type event struct {
	Time        time.Time
	Action      string
	ImportPath  string   `json:",omitempty"`
	Package     string   `json:",omitempty"`
	Test        string   `json:",omitempty"`
	Elapsed     *float64 `json:",omitempty"`
	Output      string   `json:",omitempty"`
	FailedBuild string   `json:",omitempty"`
}

func iterEvents(r io.Reader) iter.Seq2[event, error] {
	return func(yield func(event, error) bool) {
		dec := json.NewDecoder(r)
		for {
			var ev event
			if err := dec.Decode(&ev); err != nil {
				if !errors.Is(err, io.EOF) {
					yield(ev, err)
				}
				return
			}
			if !yield(ev, nil) {
				return
			}
		}
	}
}
