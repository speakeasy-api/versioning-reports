// versioning.go

package versioning

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
)

type BumpType string

const (
	BumpMajor BumpType = "major"
	BumpMinor BumpType = "minor"
	BumpPatch BumpType = "patch"
	BumpNone  BumpType = "none"
)

type VersionReport struct {
	readIndex  int
	Key          string   `json:"key"`
	Priority     int      `json:"priority"`
	BumpType     BumpType `json:"bump_type"`
	MustGenerate bool     `json:"must_generate"`
	PRReport     string   `json:"pr_report"`
}

const ENV_VAR_PREFIX = "SPEAKEASY_VERSION_REPORT_LOCATION"

var fileMutex sync.Mutex

func loadFileForWriting() (*os.File, error) {
	location := os.Getenv(ENV_VAR_PREFIX)
	if len(location) == 0 {
		return nil, fmt.Errorf("%s is not set", ENV_VAR_PREFIX)
	}
	return os.OpenFile(location, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func AddVersionReport(ctx context.Context, report VersionReport) error {
	if len(os.Getenv(ENV_VAR_PREFIX)) > 0 {
		fileMutex.Lock()
		defer fileMutex.Unlock()

		f, err := loadFileForWriting()
		if err != nil {
			return err
		}
		defer f.Close()
		if report.BumpType == "" {
			report.BumpType = BumpNone
		}

		bytes, err := json.Marshal(report)
		if err != nil {
			return err
		}

		if _, err := f.Write(append(bytes, '\n')); err != nil {
			return err
		}
	}
	return nil
}

type MergedVersionReport struct {
	Reports []VersionReport
}

func (m *MergedVersionReport) MustGenerate() bool {
	for _, report := range m.Reports {
		if report.MustGenerate {
			return true
		}
	}
	return false
}

func getMergedVersionReport() (*MergedVersionReport, error) {
	location := os.Getenv(ENV_VAR_PREFIX)
	if len(location) == 0 {
		return nil, fmt.Errorf("%s is not set", ENV_VAR_PREFIX)
	}

	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Read the entire file contents
	contents, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(contents))
	reports := make(map[string]VersionReport)

	// While there are JSON objects to decode

	for i := 0; decoder.More(); i++ {
		var report VersionReport
		if err := decoder.Decode(&report); err != nil {
			return nil, err
		}
		report.readIndex = i
		reports[report.Key] = report
	}

	// Create a slice of the latest reports
	orderedReports := make([]VersionReport, 0, len(reports))
	for _, report := range reports {
		orderedReports = append(orderedReports, report)
	}

	// Sort by priority descending, maintaining original order for equal priorities
	// If in conflict, the report read later will be considered the latest
	sort.SliceStable(orderedReports, func(i, j int) bool {
		if orderedReports[i].Priority == orderedReports[j].Priority {
			return orderedReports[i].readIndex > orderedReports[j].readIndex
		}
		return orderedReports[i].Priority > orderedReports[j].Priority
	})

	return &MergedVersionReport{Reports: orderedReports}, nil
}

func WithVersionReportCapture[T any](ctx context.Context, f func(ctx context.Context) (T, error)) (*MergedVersionReport, T, error) {
	var tempFile *os.File
	var err error
	var result T

	if len(os.Getenv(ENV_VAR_PREFIX)) == 0 {
		tempFile, err = os.CreateTemp("", "version.buf.json")
		if err != nil {
			return nil, result, err
		}
		defer os.Remove(tempFile.Name())
		os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	}

	result, err = f(ctx)
	if err != nil {
		return nil, result, err
	}

	report, err := getMergedVersionReport()
	if tempFile != nil {
		os.Unsetenv(ENV_VAR_PREFIX)
	}
	return report, result, err
}

func MustGenerate(ctx context.Context) bool {
	report, err := getMergedVersionReport()
	if err != nil || report == nil {
		return false
	}
	return report.MustGenerate()
}