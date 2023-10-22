package storager

import (
	"archive/zip"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	maxSize = int64(100 * 1024 * 1024) // Set the maximum allowed file size (e.g., 100 MB)
)

func (s *Service) createDirIfNotExist(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
	}
	return err
}

func (s *Service) checkDirExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func deleteFolderAndCalculateSize(folderPath string) (uint64, error) {
	size, err := calculateFolderSize(folderPath)
	if err != nil {
		return 0, fmt.Errorf("error calculate folder size: %w", err)
	}
	if err := os.RemoveAll(folderPath); err != nil {
		return 0, err
	}
	return size, nil
}

func (s *Service) deleteFile(filePath string) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return
	}
	if err = os.Remove(filePath); err != nil {
		s.logger.Error("error deleting file", err)
		return
	}
	s.mu.Lock()
	s.currentUsage -= uint64(fileInfo.Size())
	s.mu.Unlock()
}

func ZipAndCalculateCRC32(sourceFolder, zipFilePath string) (uint32, error) {
	// Create a CRC32 hash
	crc32Hash := crc32.NewIEEE()

	// Create the zip file
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return 0, err
	}
	defer zipFile.Close()

	// Create a zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk the source folder to add files to the zip
	err = filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Create a new file header
		zipHeader, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Use relative path within the zip file
		relPath := strings.TrimPrefix(path, sourceFolder)
		zipHeader.Name = relPath

		// Open the source file
		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		// Create a writer in the zip file
		writer, err := zipWriter.CreateHeader(zipHeader)
		if err != nil {
			return err
		}

		// Create a CRC32 writer to calculate the checksum
		crc32Writer := io.MultiWriter(writer, crc32Hash)

		// Copy the file to the zip file and calculate CRC32
		_, err = io.Copy(crc32Writer, sourceFile)
		return err
	})

	if err != nil {
		return 0, err
	}

	// Calculate the CRC32 checksum
	checksum := crc32Hash.Sum32()
	return checksum, nil
}

func CalculateCRC32(filePath string) (uint32, error) {
	// Open the zip file
	zipFile, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer zipFile.Close()

	// Create a new CRC32 hasher
	crc32Hash := crc32.NewIEEE()

	// Create a new zip archive reader
	fileStat, err := zipFile.Stat()
	if err != nil {
		return 0, err
	}
	zipReader, err := zip.NewReader(zipFile, fileStat.Size())
	if err != nil {
		return 0, err
	}

	// Iterate through the zip entries and calculate CRC32
	for _, entry := range zipReader.File {
		// Open the entry from the zip
		entryFile, err := entry.Open()
		if err != nil {
			return 0, err
		}

		// Calculate CRC32 for the entry
		limitReader := io.LimitedReader{R: entryFile, N: maxSize}
		_, err = io.Copy(crc32Hash, &limitReader)
		entryFile.Close()
		if err != nil {
			return 0, err
		}
	}

	// Calculate and return the CRC32 checksum
	checksum := crc32Hash.Sum32()
	return checksum, nil
}

func UnpackZip(zipFile, targetDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if err = extractFile(file, targetDir); err != nil {
			return fmt.Errorf("error extracting file %s: %w", file.Name, err)
		}
	}
	return nil
}

func extractFile(file *zip.File, targetDir string) error {
	// Sanitize the file name to prevent directory traversal.
	filePath := filepath.Join(targetDir, filepath.FromSlash(file.Name))
	if !strings.HasPrefix(filePath, targetDir) {
		return fmt.Errorf("file path is outside the target directory: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return nil
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating nested directory: %w", err)
	}

	dstFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if _, err = io.Copy(dstFile, io.LimitReader(srcFile, maxSize)); err != nil {
		return err
	}
	return nil
}

func calculateFolderSize(folderPath string) (totalSize uint64, err error) {
	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// If it's a file, add its size to the total
		if !info.IsDir() {
			totalSize += uint64(info.Size())
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return totalSize, nil
}
