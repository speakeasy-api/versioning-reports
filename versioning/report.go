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
	BumpMajor      BumpType = "major"
	BumpMinor      BumpType = "minor"
	BumpPatch      BumpType = "patch"
	BumpGraduate   BumpType = "graduate"
	BumpPrerelease BumpType = "prerelease"
	BumpCustom     BumpType = "custom"
	BumpNone       BumpType = "none"
)

type VersionReport struct {
	readIndex    int
	Key          string   `json:"key"`
	Priority     int      `json:"priority"`
	BumpType     BumpType `json:"bump_type"`
	NewVersion   string   `json:"new_version"`
	MustGenerate bool     `json:"must_generate"`
	PRReport     string   `json:"pr_report"`
	CommitReport string   `json:"commit_report"`
}

// VersionReportV2Data is the top-level container for V2 changelog data.
// It stores structured change information for all targets in a single generation run.
// This enables more flexible changelog rendering with support for multiple targets
// and future features like commit templates.
type VersionReportV2Data struct {
	Targets []VersionReportV2Target `json:"targets"`
}

// VersionReportV2Target represents the changes made to a single SDK target.
type VersionReportV2Target struct {
	TargetName      string                     `json:"target_name"`                // e.g., "typescript", "go", "python"
	PackageName     string                     `json:"package_name,omitempty"`     // e.g., "@vercel/sdk", "github.com/vercel/sdk-go"
	PreviousVersion string                     `json:"previous_version,omitempty"` // e.g., "1.23.7"
	NewVersion      string                     `json:"new_version"`                // e.g., "1.23.8"
	GeneratedAt     string                     `json:"generated_at,omitempty"`     // ISO8601 timestamp
	Operations      []VersionReportV2Operation `json:"operations"`                 // List of changed operations
}

// VersionReportV2OperationType indicates what kind of change happened to an operation.
type VersionReportV2OperationType string

const (
	OperationAdded      VersionReportV2OperationType = "added"
	OperationRemoved    VersionReportV2OperationType = "removed"
	OperationModified   VersionReportV2OperationType = "modified"
	OperationDeprecated VersionReportV2OperationType = "deprecated"
)

// VersionReportV2Operation represents changes to a single SDK operation/method.
type VersionReportV2Operation struct {
	Name       string                       `json:"name"`        // e.g., "sdk.createUser()", "Sdk.Inner.ComplexOperation()"
	Type       VersionReportV2OperationType `json:"type"`        // added, removed, modified, deprecated
	IsBreaking bool                         `json:"is_breaking"` // true if any child change is breaking
	Changes    []VersionReportV2FieldChange `json:"changes"`     // list of field-level changes (empty for added/removed operations)
}

// VersionReportV2FieldChangeType indicates what kind of change happened to a field.
type VersionReportV2FieldChangeType string

const (
	FieldAdded   VersionReportV2FieldChangeType = "added"
	FieldRemoved VersionReportV2FieldChangeType = "removed"
	FieldChanged VersionReportV2FieldChangeType = "changed"
)

// VersionReportV2FieldChange represents a single field-level change within an operation.
type VersionReportV2FieldChange struct {
	Path       string                         `json:"path"`        // e.g., "request.email", "response.data.items"
	Type       VersionReportV2FieldChangeType `json:"type"`        // added, removed, changed
	IsBreaking bool                           `json:"is_breaking"` // true if this specific change is breaking
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

func (m *MergedVersionReport) GetMarkdownSection() string {
	inner := ""
	for _, report := range m.Reports {
		if len(report.PRReport) > 0 {
			inner += report.PRReport + "\n"
		}
	}
	return inner
}

func (m *MergedVersionReport) GetCommitMarkdownSection() string {
	inner := ""
	for _, report := range m.Reports {
		if len(report.CommitReport) > 0 {
			inner += report.CommitReport + "\n"
		}
	}
	return inner

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

// VersionReportCapture holds both V1 and V2 version reports.
type VersionReportCapture struct {
	V1 *MergedVersionReport
	V2 *VersionReportV2Data
}

func WithVersionReportCapture[T any](ctx context.Context, f func(ctx context.Context) (T, error)) (*VersionReportCapture, T, error) {
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

		// Also clean up the V2 file that will be created
		v2Location := getV2Location()
		if len(v2Location) > 0 {
			defer os.Remove(v2Location)
		}
	}

	result, err = f(ctx)
	if err != nil {
		return nil, result, err
	}

	report, err := getMergedVersionReport()
	reportV2, errV2 := GetVersionReportV2()

	if tempFile != nil {
		os.Unsetenv(ENV_VAR_PREFIX)
	}

	// Prioritize the function error, then V1 report error, then V2 report error
	if err != nil {
		return nil, result, err
	}
	if errV2 != nil {
		return nil, result, errV2
	}

	return &VersionReportCapture{V1: report, V2: reportV2}, result, nil
}

func MustGenerate(ctx context.Context) bool {
	report, err := getMergedVersionReport()
	if err != nil || report == nil {
		return false
	}
	return report.MustGenerate()
}

// V2 Report Functions
// These functions provide structured changelog storage and rendering,
// running alongside V1 for backwards compatibility.

var v2FileMutex sync.Mutex

// getV2Location derives the V2 report file location from the V1 location.
// If the V1 location is "/path/to/version.json", the V2 location will be "/path/to/version.v2.json".
// Returns empty string if the V1 environment variable is not set.
func getV2Location() string {
	v1Location := os.Getenv(ENV_VAR_PREFIX)
	if len(v1Location) == 0 {
		return ""
	}

	// Append .v2 before the file extension
	// e.g., "/path/to/version.json" -> "/path/to/version.v2.json"
	if len(v1Location) > 5 && v1Location[len(v1Location)-5:] == ".json" {
		return v1Location[:len(v1Location)-5] + ".v2.json"
	}
	// Fallback: just append .v2 to the end
	return v1Location + ".v2"
}

// AddVersionReportV2Target appends a single target's changelog data to the V2 report file.
// Multiple calls with different targets will accumulate in the same file.
// Returns nil if the V1 environment variable is not set (graceful degradation).
func AddVersionReportV2Target(ctx context.Context, target VersionReportV2Target) error {
	location := getV2Location()
	if len(location) == 0 {
		// V1 not configured, silently skip (backwards compatible)
		return nil
	}

	v2FileMutex.Lock()
	defer v2FileMutex.Unlock()

	f, err := os.OpenFile(location, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open V2 report file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("failed to marshal V2 target: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write V2 target: %w", err)
	}

	return nil
}

// GetVersionReportV2 reads all V2 target reports from the file and returns them
// as a VersionReportV2Data struct. Returns nil if the file doesn't exist or
// the V1 environment variable is not set.
func GetVersionReportV2() (*VersionReportV2Data, error) {
	location := getV2Location()
	if len(location) == 0 {
		return nil, nil // V1 not configured
	}

	v2FileMutex.Lock()
	defer v2FileMutex.Unlock()

	contents, err := os.ReadFile(location)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, not an error
		}
		return nil, fmt.Errorf("failed to read V2 report file: %w", err)
	}

	if len(contents) == 0 {
		return nil, nil // Empty file
	}

	decoder := json.NewDecoder(bytes.NewReader(contents))
	targets := make([]VersionReportV2Target, 0)

	for decoder.More() {
		var target VersionReportV2Target
		if err := decoder.Decode(&target); err != nil {
			return nil, fmt.Errorf("failed to decode V2 target: %w", err)
		}
		targets = append(targets, target)
	}

	if len(targets) == 0 {
		return nil, nil
	}

	return &VersionReportV2Data{Targets: targets}, nil
}
