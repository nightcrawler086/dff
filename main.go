package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"encoding/hex"
	"slices"
)

type File struct {
	Hash string
	Path string
}

var FileList []File

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
		sum := hash.Sum(nil)
		cf := File{
			Hash: hex.EncodeToString(sum), 
			Path: path,
		}
		if slices.ContainsFunc(FileList, func(f File) bool {
			return f.Hash == cf.Hash
		}) {
			fmt.Printf("File %s is a duplicate ", cf.Path)
		} else {
			FileList = append(FileList, cf)
		}
		fmt.Printf("%s --> %s\n", cf.Hash, cf.Path)
	}
	return nil
}

func main() {
	filepath.WalkDir(".", walker)
}
