package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed archive.zip
var projZip []byte

// taken from ldflags
var entrypoint string

var tmp = "tmp"

func extract(zr *zip.Reader, dest string) {
	for _, f := range zr.File {
		name := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(name, 0777)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			fh, err := f.Open()
			if err != nil {
				log.Fatal(err)
			}
			defer fh.Close()

			data, err := io.ReadAll(fh)
			if err != nil {
				log.Fatal(err)
			}

			err = os.WriteFile(name, data, 0777)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func run(entry string, dir string) {
	out, _ := exec.Command(
		filepath.Join(dir, "venv", "bin", "activate"),
		filepath.Join(dir, entry),
	).CombinedOutput()
	fmt.Print(string(out))

	cmd := exec.Command("python3", filepath.Join(dir, entry))
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	out, _ = exec.Command("deactivate").CombinedOutput()
	fmt.Print(string(out))
}

func main() {
	z, err := zip.NewReader(bytes.NewReader(projZip), int64(len(projZip)))
	if err != nil {
		log.Fatal(fmt.Errorf("error reading archive: %w", err))
	}
	extract(z, tmp)

	defer func() {
		err = os.RemoveAll(tmp)
		if err != nil {
			log.Fatal(fmt.Errorf("error removing temp dir: %w", err))
		}
	}()

	run(entrypoint, tmp)
}
