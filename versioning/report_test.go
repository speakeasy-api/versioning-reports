// versioning_test.go

package versioning

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddVersionReport(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_version_report.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	defer os.Unsetenv(ENV_VAR_PREFIX)

	ctx := context.Background()
	report := VersionReport{
		Key:          "test",
		Priority:     1,
		BumpType:     BumpMinor,
		MustGenerate: true,
		PRReport:     "Test report",
	}

	err = AddVersionReport(ctx, report)
	require.NoError(t, err)

	content, err := os.ReadFile(tempFile.Name())
	require.NoError(t, err)

	var readReport VersionReport
	err = json.Unmarshal(content, &readReport)
	require.NoError(t, err)

	assert.Equal(t, report, readReport)
}

func TestAddVersionReportWithCommitReport(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_version_report_commit.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	defer os.Unsetenv(ENV_VAR_PREFIX)

	ctx := context.Background()
	report := VersionReport{
		Key:          "test_commit",
		Priority:     1,
		BumpType:     BumpMinor,
		MustGenerate: true,
		PRReport:     "Test report",
		CommitReport: "Test commit report",
	}

	err = AddVersionReport(ctx, report)
	require.NoError(t, err)

	content, err := os.ReadFile(tempFile.Name())
	require.NoError(t, err)

	var readReport VersionReport
	err = json.Unmarshal(content, &readReport)
	require.NoError(t, err)

	assert.Equal(t, report, readReport)
}

func TestGetMergedVersionReport(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_merged_version_report.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	defer os.Unsetenv(ENV_VAR_PREFIX)

	reports := []VersionReport{
		{Key: "test1", Priority: 2, MustGenerate: true, PRReport: "Test report 1"},
		{Key: "test2", Priority: 1, MustGenerate: false, PRReport: "Test report 2"},
	}

	for _, report := range reports {
		bytes, _ := json.Marshal(report)
		tempFile.Write(append(bytes, '\n'))
	}
	tempFile.Close()

	mergedReport, err := getMergedVersionReport()
	require.NoError(t, err)

	assert.Len(t, mergedReport.Reports, 2)
	assert.Equal(t, "test1", mergedReport.Reports[0].Key)
	assert.Equal(t, "test2", mergedReport.Reports[1].Key)
	assert.Equal(t, "Test report 1\nTest report 2\n", mergedReport.GetMarkdownSection())
	assert.True(t, mergedReport.MustGenerate())
}

func TestGetMergedVersionReportWithCommitReports(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_merged_version_report_commit.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	defer os.Unsetenv(ENV_VAR_PREFIX)

	reports := []VersionReport{
		{Key: "test1", Priority: 2, MustGenerate: true, PRReport: "Test report 1", CommitReport: "Test commit report 1"},
		{Key: "test2", Priority: 1, MustGenerate: false, PRReport: "Test report 2", CommitReport: "Test commit report 2"},
	}

	for _, report := range reports {
		bytes, _ := json.Marshal(report)
		tempFile.Write(append(bytes, '\n'))
	}
	tempFile.Close()

	mergedReport, err := getMergedVersionReport()
	require.NoError(t, err)

	assert.Len(t, mergedReport.Reports, 2)
	assert.Equal(t, "test1", mergedReport.Reports[0].Key)
	assert.Equal(t, "test2", mergedReport.Reports[1].Key)
	assert.Equal(t, "Test report 1\nTest report 2\n", mergedReport.GetMarkdownSection())
	assert.Equal(t, "Test commit report 1\nTest commit report 2\n", mergedReport.GetCommitMarkdownSection())
	assert.True(t, mergedReport.MustGenerate())
}

func TestGetMergedVersionReportWithMixedCommitReports(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_merged_version_report_mixed_commit.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	defer os.Unsetenv(ENV_VAR_PREFIX)

	// test2 has no commit report (empty string)
	reports := []VersionReport{
		{Key: "test1", Priority: 2, MustGenerate: true, PRReport: "Test report 1", CommitReport: "Test commit report 1"},
		{Key: "test2", Priority: 1, MustGenerate: false, PRReport: "Test report 2", CommitReport: ""},
	}

	for _, report := range reports {
		bytes, _ := json.Marshal(report)
		tempFile.Write(append(bytes, '\n'))
	}
	tempFile.Close()

	mergedReport, err := getMergedVersionReport()
	require.NoError(t, err)

	assert.Len(t, mergedReport.Reports, 2)
	assert.Equal(t, "test1", mergedReport.Reports[0].Key)
	assert.Equal(t, "test2", mergedReport.Reports[1].Key)
	assert.Equal(t, "Test report 1\nTest report 2\n", mergedReport.GetMarkdownSection())
	assert.Equal(t, "Test commit report 1\n", mergedReport.GetCommitMarkdownSection())
	assert.True(t, mergedReport.MustGenerate())
}

func TestGetMergedVersionReportWithEmptyCommitReports(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_merged_version_report_empty_commit.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	defer os.Unsetenv(ENV_VAR_PREFIX)

	reports := []VersionReport{
		{Key: "test1", Priority: 2, MustGenerate: true, PRReport: "Test report 1", CommitReport: ""},
		{Key: "test2", Priority: 1, MustGenerate: false, PRReport: "Test report 2", CommitReport: ""},
	}

	for _, report := range reports {
		bytes, _ := json.Marshal(report)
		tempFile.Write(append(bytes, '\n'))
	}
	tempFile.Close()

	mergedReport, err := getMergedVersionReport()
	require.NoError(t, err)

	assert.Len(t, mergedReport.Reports, 2)
	assert.Equal(t, "test1", mergedReport.Reports[0].Key)
	assert.Equal(t, "test2", mergedReport.Reports[1].Key)
	assert.Equal(t, "Test report 1\nTest report 2\n", mergedReport.GetMarkdownSection())
	assert.Equal(t, "", mergedReport.GetCommitMarkdownSection())
	assert.True(t, mergedReport.MustGenerate())
}

func TestWithVersionReportCapture(t *testing.T) {
	ctx := context.Background()

	type unknown struct{}
	versionReports, _, err := WithVersionReportCapture[*unknown](ctx, func(ctx context.Context) (*unknown, error) {
		return nil, AddVersionReport(ctx, VersionReport{
			Key:          "test",
			Priority:     1,
			MustGenerate: true,
			PRReport:     "Test report",
		})
	})

	require.NoError(t, err)
	require.NotNil(t, versionReports)
	require.NotNil(t, versionReports.V1)
	assert.Len(t, versionReports.V1.Reports, 1)
	assert.Equal(t, "test", versionReports.V1.Reports[0].Key)
	assert.True(t, versionReports.V1.MustGenerate())
}

func TestWithVersionReportCaptureWithCommitReport(t *testing.T) {
	ctx := context.Background()

	type unknown struct{}
	versionReports, _, err := WithVersionReportCapture(ctx, func(ctx context.Context) (*unknown, error) {
		return nil, AddVersionReport(ctx, VersionReport{
			Key:          "test_commit",
			Priority:     1,
			MustGenerate: true,
			PRReport:     "Test report",
			CommitReport: "Test commit report",
		})
	})

	require.NoError(t, err)
	require.NotNil(t, versionReports)
	require.NotNil(t, versionReports.V1)
	assert.Len(t, versionReports.V1.Reports, 1)
	assert.Equal(t, "test_commit", versionReports.V1.Reports[0].Key)
	assert.True(t, versionReports.V1.MustGenerate())
	assert.Equal(t, "Test commit report", versionReports.V1.Reports[0].CommitReport)
	assert.Equal(t, "Test commit report\n", versionReports.V1.GetCommitMarkdownSection())
}

func TestIntegrationWithSubprocesses(t *testing.T) {
	ctx := context.Background()
	type unknown struct{}

	versionReports, _, err := WithVersionReportCapture(ctx, func(ctx context.Context) (*unknown, error) {
		// Run two subprocesses that add version reports

		for i := 0; i < 2; i++ {
			err := execSubprocess(i, "original")
			if err != nil {
				return nil, err
			}
		}
		return nil, execSubprocess(0, "overridden")
	})

	require.NoError(t, err)
	require.NotNil(t, versionReports)
	require.NotNil(t, versionReports.V1)
	assert.Len(t, versionReports.V1.Reports, 2)
	assert.Equal(t, "subprocess1", versionReports.V1.Reports[0].Key)
	assert.Equal(t, "overridden", versionReports.V1.Reports[0].PRReport)
	assert.Equal(t, "subprocess2", versionReports.V1.Reports[1].Key)
	assert.Equal(t, "original", versionReports.V1.Reports[1].PRReport)
	assert.True(t, versionReports.V1.MustGenerate())
}

func execSubprocess(i int, extra string) error {
	cmd := exec.Command("go", "run", "testdata/subprocess.go", fmt.Sprintf("%v", i+1), extra)
	cmd.Env = append(os.Environ(), ENV_VAR_PREFIX+"="+os.Getenv(ENV_VAR_PREFIX))
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// V2 Tests

func TestAddVersionReportV2Target(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_v1_report.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Set V1 location, which will derive the V2 location
	os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	defer os.Unsetenv(ENV_VAR_PREFIX)

	// Calculate V2 location and ensure cleanup
	v2Location := getV2Location()
	defer os.Remove(v2Location)

	ctx := context.Background()
	target := VersionReportV2Target{
		TargetName:      "typescript",
		PackageName:     "@vercel/sdk",
		PreviousVersion: "1.23.7",
		NewVersion:      "1.23.8",
		Operations: []VersionReportV2Operation{
			{
				Name:       "sdk.createUser()",
				Type:       OperationModified,
				IsBreaking: true,
				Changes: []VersionReportV2FieldChange{
					{Path: "request.email", Type: FieldAdded, IsBreaking: false},
					{Path: "response", Type: FieldChanged, IsBreaking: true},
				},
			},
		},
	}

	err = AddVersionReportV2Target(ctx, target)
	require.NoError(t, err)

	content, err := os.ReadFile(v2Location)
	require.NoError(t, err)

	var readTarget VersionReportV2Target
	err = json.Unmarshal(content, &readTarget)
	require.NoError(t, err)

	assert.Equal(t, target.TargetName, readTarget.TargetName)
	assert.Equal(t, target.PackageName, readTarget.PackageName)
	assert.Equal(t, target.NewVersion, readTarget.NewVersion)
	assert.Len(t, readTarget.Operations, 1)
	assert.Len(t, readTarget.Operations[0].Changes, 2)
}

func TestAddVersionReportV2Target_NoEnvVar(t *testing.T) {
	// Ensure V1 env var is not set
	os.Unsetenv(ENV_VAR_PREFIX)

	ctx := context.Background()
	target := VersionReportV2Target{
		TargetName: "typescript",
		NewVersion: "1.0.0",
	}

	// Should succeed silently when V1 env var is not set
	err := AddVersionReportV2Target(ctx, target)
	assert.NoError(t, err)
}

func TestGetVersionReportV2(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_v1_get_report.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Set V1 location, which will derive the V2 location
	os.Setenv(ENV_VAR_PREFIX, tempFile.Name())
	defer os.Unsetenv(ENV_VAR_PREFIX)

	// Calculate V2 location and ensure cleanup
	v2Location := getV2Location()
	defer os.Remove(v2Location)

	targets := []VersionReportV2Target{
		{
			TargetName:  "typescript",
			PackageName: "@vercel/sdk",
			NewVersion:  "1.23.8",
			Operations: []VersionReportV2Operation{
				{Name: "sdk.createUser()", Type: OperationModified, IsBreaking: false},
			},
		},
		{
			TargetName: "go",
			NewVersion: "1.9.2",
			Operations: []VersionReportV2Operation{
				{Name: "Sdk.Inner.ComplexOperation()", Type: OperationModified, IsBreaking: true},
			},
		},
	}

	// Write directly to the V2 location
	v2File, err := os.OpenFile(v2Location, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(t, err)
	for _, target := range targets {
		bytes, _ := json.Marshal(target)
		v2File.Write(append(bytes, '\n'))
	}
	v2File.Close()

	data, err := GetVersionReportV2()
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.Len(t, data.Targets, 2)
	assert.Equal(t, "typescript", data.Targets[0].TargetName)
	assert.Equal(t, "go", data.Targets[1].TargetName)
}

func TestGetVersionReportV2_NoEnvVar(t *testing.T) {
	// Ensure V1 env var is not set
	os.Unsetenv(ENV_VAR_PREFIX)

	data, err := GetVersionReportV2()
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestGetVersionReportV2_FileNotExist(t *testing.T) {
	// Set V1 location to a path where V2 file doesn't exist
	os.Setenv(ENV_VAR_PREFIX, "/nonexistent/path/file.json")
	defer os.Unsetenv(ENV_VAR_PREFIX)

	data, err := GetVersionReportV2()
	assert.NoError(t, err)
	assert.Nil(t, data)
}
