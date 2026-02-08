// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

// extractTarGz extracts a tar.gz archive and returns the name of the mql binary
func extractTarGz(reader io.Reader, destPath string) (string, error) {
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return "", errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var binaryName string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.Wrap(err, "failed to read tar entry")
		}

		// Only extract regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Security: prevent path traversal
		name := filepath.Clean(header.Name)
		if strings.Contains(name, "..") {
			log.Warn().Str("name", header.Name).Msg("self-update: skipping suspicious path in archive")
			continue
		}

		// Only extract the mql binary
		baseName := filepath.Base(name)
		if !strings.HasPrefix(baseName, "mql") {
			log.Debug().Str("name", name).Msg("self-update: skipping non-mql file")
			continue
		}

		destFile := filepath.Join(destPath, baseName)
		log.Debug().Str("name", baseName).Str("dest", destFile).Msg("self-update: extracting file")

		// Create the file
		f, err := os.Create(destFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to create destination file")
		}

		// Copy contents
		if _, err := io.Copy(f, tarReader); err != nil {
			f.Close()
			return "", errors.Wrap(err, "failed to extract file contents")
		}
		f.Close()

		// Track the binary name (without .exe for consistency)
		if baseName == "mql" || baseName == "mql.exe" {
			binaryName = baseName
		}
	}

	if binaryName == "" {
		return "", errors.New("mql binary not found in archive")
	}

	return binaryName, nil
}

// extractZip extracts a zip archive and returns the name of the mql binary.
// Note: zip requires random access, so we need the file path, not just a reader.
func extractZip(reader io.Reader, destPath string, archivePath string) (string, error) {
	// For zip, we need to use the file path because zip requires random access
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open zip archive")
	}
	defer zipReader.Close()

	var binaryName string

	for _, file := range zipReader.File {
		// Only extract regular files
		if file.FileInfo().IsDir() {
			continue
		}

		// Security: prevent path traversal
		name := filepath.Clean(file.Name)
		if strings.Contains(name, "..") {
			log.Warn().Str("name", file.Name).Msg("self-update: skipping suspicious path in archive")
			continue
		}

		// Only extract the mql binary
		baseName := filepath.Base(name)
		if !strings.HasPrefix(baseName, "mql") {
			log.Debug().Str("name", name).Msg("self-update: skipping non-mql file")
			continue
		}

		destFile := filepath.Join(destPath, baseName)
		log.Debug().Str("name", baseName).Str("dest", destFile).Msg("self-update: extracting file")

		// Open the file in the archive
		rc, err := file.Open()
		if err != nil {
			return "", errors.Wrap(err, "failed to open file in archive")
		}

		// Create the destination file
		f, err := os.Create(destFile)
		if err != nil {
			rc.Close()
			return "", errors.Wrap(err, "failed to create destination file")
		}

		// Copy contents
		if _, err := io.Copy(f, rc); err != nil {
			f.Close()
			rc.Close()
			return "", errors.Wrap(err, "failed to extract file contents")
		}
		f.Close()
		rc.Close()

		// Track the binary name
		if baseName == "mql" || baseName == "mql.exe" {
			binaryName = baseName
		}
	}

	if binaryName == "" {
		return "", errors.New("mql binary not found in archive")
	}

	return binaryName, nil
}
