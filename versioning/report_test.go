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
	commitReport := "Test commit report"
	report := VersionReport{
		Key:          "test_commit",
		Priority:     1,
		BumpType:     BumpMinor,
		MustGenerate: true,
		PRReport:     "Test report",
		CommitReport: commitReport,
	}

	err = AddVersionReport(ctx, report)
	require.NoError(t, err)

	content, err := os.ReadFile(tempFile.Name())
	require.NoError(t, err)

	var readReport VersionReport
	err = json.Unmarshal(content, &readReport)
	require.NoError(t, err)

	assert.Equal(t, report, readReport)
	assert.Equal(t, commitReport, readReport.CommitReport)
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

	commitReport1 := "Test commit report 1"
	commitReport2 := "Test commit report 2"
	reports := []VersionReport{
		{Key: "test1", Priority: 2, MustGenerate: true, PRReport: "Test report 1", CommitReport: commitReport1},
		{Key: "test2", Priority: 1, MustGenerate: false, PRReport: "Test report 2", CommitReport: commitReport2},
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

	commitReport1 := "Test commit report 1"
	// test2 has no commit report (empty string)
	reports := []VersionReport{
		{Key: "test1", Priority: 2, MustGenerate: true, PRReport: "Test report 1", CommitReport: commitReport1},
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
	report, _, err := WithVersionReportCapture[*unknown](ctx, func(ctx context.Context) (*unknown, error) {
		return nil, AddVersionReport(ctx, VersionReport{
			Key:          "test",
			Priority:     1,
			MustGenerate: true,
			PRReport:     "Test report",
		})
	})

	require.NoError(t, err)
	assert.Len(t, report.Reports, 1)
	assert.Equal(t, "test", report.Reports[0].Key)
	assert.True(t, report.MustGenerate())
}

func TestWithVersionReportCaptureWithCommitReport(t *testing.T) {
	ctx := context.Background()

	type unknown struct{}
	commitReport := "Test commit report"
	report, _, err := WithVersionReportCapture[*unknown](ctx, func(ctx context.Context) (*unknown, error) {
		return nil, AddVersionReport(ctx, VersionReport{
			Key:          "test_commit",
			Priority:     1,
			MustGenerate: true,
			PRReport:     "Test report",
			CommitReport: commitReport,
		})
	})

	require.NoError(t, err)
	assert.Len(t, report.Reports, 1)
	assert.Equal(t, "test_commit", report.Reports[0].Key)
	assert.True(t, report.MustGenerate())
	assert.Equal(t, commitReport, report.Reports[0].CommitReport)
	assert.Equal(t, "Test commit report\n", report.GetCommitMarkdownSection())
}

func TestIntegrationWithSubprocesses(t *testing.T) {
	ctx := context.Background()
	type unknown struct{}

	report, _, err := WithVersionReportCapture[*unknown](ctx, func(ctx context.Context) (*unknown, error) {
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
	assert.Len(t, report.Reports, 2)
	assert.Equal(t, "subprocess1", report.Reports[0].Key)
	assert.Equal(t, "overridden", report.Reports[0].PRReport)
	assert.Equal(t, "subprocess2", report.Reports[1].Key)
	assert.Equal(t, "original", report.Reports[1].PRReport)
	assert.True(t, report.MustGenerate())
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
