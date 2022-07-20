package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed templates/pybinary.go
var templateData []byte

const tmp = "tmp"

func isSymlink(mode fs.FileMode) bool {
	return mode&fs.ModeSymlink == fs.ModeSymlink
}

func compress(src string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	defer w.Close()

	addToArchive := func(path string, info fs.DirEntry, e error) error {
		// if it's a dir, we must append a trailing slash
		if info.IsDir() {
			path = path + string(os.PathSeparator)
		}

		zf, err := w.Create(path)
		if err != nil {
			return err
		}

		// if it's a file, we have to write the file data to the archive
		if !info.IsDir() && !isSymlink(info.Type()) {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			_, err = zf.Write(data)
			if err != nil {
				return err
			}
		}
		return nil
	}

	err := filepath.WalkDir(src, addToArchive)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func main() {
	projDir := os.Args[1]
	entryFile := os.Args[2]
	outFile := strings.TrimSuffix(entryFile, filepath.Ext(entryFile))
	entrypoint := filepath.Join(projDir, entryFile)
	archivePath := filepath.Join(tmp, "archive.zip")
	templatePath := filepath.Join(tmp, "template.go")

	archive, err := compress(projDir)
	if err != nil {
		log.Print(fmt.Errorf("error during compression: %w", err))
		return
	}

	err = os.Mkdir(tmp, 0777)
	if err != nil {
		log.Print(fmt.Errorf("error making temp dir: %w", err))
		return
	}

	defer func() {
		err = os.RemoveAll(tmp)
		if err != nil {
			log.Fatal(fmt.Errorf("error removing temp dir: %w", err))
		}
	}()

	err = os.WriteFile(archivePath, archive.Bytes(), 0777)
	if err != nil {
		log.Print(fmt.Errorf("error writing archive: %w", err))
		return
	}

	err = os.WriteFile(templatePath, templateData, 0777)
	if err != nil {
		log.Print(fmt.Errorf("error writing template: %w", err))
		return
	}

	entryFlag := "-X main.entrypoint=" + entrypoint
	cmd := exec.Command("go", "build", "-ldflags", entryFlag, "-o", outFile, templatePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Print(fmt.Errorf("error compiling binary: %w", err))
		return
	}
}
