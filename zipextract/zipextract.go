package zipextract

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)
// function trying to open archive, if path correctly
func Open(dir string, target string, nameout string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isArchveFile(path) {
			OpenArchive(path, target, nameout)
		}
		return nil
	})
	if err != nil {
		log.Printf("Error when walking directory: %v", err)
	}
    return err
}

func OpenArchive(path string, target string, nameout string) []string {
	zf, err := zip.OpenReader(path)
	extract := []string{}
	if err != nil {
		log.Printf("Error when open archive: %v", err)
	}
	defer zf.Close()
	for _, file := range zf.File {
		extract = append(extract, file.Name)
		if file.Name == target {
			extractor(file, nameout)
		}
	}
	return extract
}

func isArchveFile(filename string) bool {
	extensions := []string{".zip"} 
	ext := strings.ToLower(filepath.Ext(filename))
	for _, e := range extensions {
		if ext == e {
			return true
		}
	}
	return false
}

func extractor(target *zip.File, name string) {
	src, err := target.Open()
	if err != nil {
		log.Printf("Can't open file in archive: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(name)
	if err != nil {
		log.Printf("Can't create file: %v", err)
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		log.Printf("Error when extracting file: %v", err)
	}
}

func Archivator(files []string) *bytes.Buffer {
	buf := new(bytes.Buffer)
	zipArch := zip.NewWriter(buf)
	for _, file := range files {
		fw, err := zipArch.Create(file)
		if err != nil {
			log.Printf("error when write data in archive: %v", err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error when trying to read file: %v", err)
		}
		_, err = fw.Write(data)
		if err != nil {
			log.Printf("Error when write data in archive: %v", err)
		}
	}
	err := zipArch.Close()
	if err != nil {
		log.Printf("Error when close zip archive: %v", err)
	}
	return buf
}

func AddFileToZIP(zipWriter *zip.Writer, filename string) {

	fileToZip, err := os.Open(filename)
	defer fileToZip.Close()
	if err != nil {
		log.Printf("Error when open file to add in archive: %v", err)
	} else {
		log.Printf("Successfully open file: %v", filename)
        // Reciever info about file
		info, err := fileToZip.Stat()
		if err != nil {
			log.Printf("Error when check status file to add in archive: %v", err)
		} else {
			log.Printf("Successfully receive stats: %v", info)
		}
        // Create header for file
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			log.Printf("Error when add header to add in archive: %v", err)
		} else {
			log.Printf("add the header: %v", header)
		}
        // use header by default
		header.Method = zip.Deflate
        // Create new file in archive
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			log.Printf("Error create file to add in archive: %v", err)
		} else {
			log.Printf("Successful add the add file in archive: %v", writer)
		}
		_, err = io.Copy(writer, fileToZip)
		if err != nil {
			log.Printf("Error when copy file to archive: %v", err)
		} else {
			log.Printf("successfully add file to archive: %v", err)
		}
	}
}
