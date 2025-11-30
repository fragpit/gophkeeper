package buildinfo

import (
	"io"
	"os"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	_ = r.Close()
	os.Stdout = old
	return string(out)
}

func TestPrintBuildInfo_AllEmpty(t *testing.T) {
	out := captureStdout(t, func() {
		PrintBuildInfo("", "", "")
	})

	expected := "Build version: N/A\nBuild date: \nBuild commit: \n"
	if out != expected {
		t.Fatalf("unexpected output\nwant:\n%q\nhave:\n%q", expected, out)
	}
}

func TestPrintBuildInfo_AllNonEmpty(t *testing.T) {
	out := captureStdout(t, func() {
		PrintBuildInfo("1.2.3", "2025-01-01", "abcdef")
	})

	// According to current implementation: version stays, date and commit become "N/A"
	expected := "Build version: 1.2.3\nBuild date: N/A\nBuild commit: N/A\n"
	if out != expected {
		t.Fatalf("unexpected output\nwant:\n%q\nhave:\n%q", expected, out)
	}
}

func TestPrintBuildInfo_Mixed(t *testing.T) {
	out := captureStdout(t, func() {
		PrintBuildInfo("", "2025-01-01", "")
	})

	// version empty -> N/A; date non-empty -> becomes N/A; commit empty -> stays empty
	expected := "Build version: N/A\nBuild date: N/A\nBuild commit: \n"
	if out != expected {
		t.Fatalf("unexpected output\nwant:\n%q\nhave:\n%q", expected, out)
	}
}
