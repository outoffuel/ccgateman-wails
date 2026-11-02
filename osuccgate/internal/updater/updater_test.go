package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTarGzBinary(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test.tar.gz")
	destPath := filepath.Join(tempDir, "extracted")

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("#!/bin/sh\necho test\n")
	hdr := &tar.Header{
		Name: "bin/osuccgate",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tw.Close failed: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gw.Close failed: %v", err)
	}

	if err := os.WriteFile(archivePath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := extractTarGzBinary(archivePath, "osuccgate", destPath); err != nil {
		t.Fatalf("extractTarGzBinary failed: %v", err)
	}

	extractedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(extractedContent, content) {
		t.Fatalf("expected content %q, got %q", content, extractedContent)
	}
}

func TestCleanupOldExecutable(t *testing.T) {
	// パニックなく実行できることを確認
	CleanupOldExecutable()
}

