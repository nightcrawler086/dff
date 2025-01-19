package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func walker(path string, d os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {
		file, err := os.Open(path)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer file.Close()
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(strings.ToUpper(fmt.Sprintf("%x", hash.Sum(nil))), path)
	}
	return nil
}

func main() {
	filepath.WalkDir(".", walker)
}
