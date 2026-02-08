package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"encoding/hex"
	"runtime/pprof"
)


var FileList = make(map[string]*File)

type File struct {
	Hash string
	Path string
}

func OpenFile(path string) *os.File {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err.Error())
	}
	// ignore dotfiles by default?
	return file
}

func HashFile(file *os.File) []byte {
	hash := sha256.New()
	// since *os.File implements the io.Reader interface, it can
	// be used where an io.Reader is expected
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Println(err.Error())
	}
	sum := hash.Sum(nil)
	return sum
}

func walker(path string, d os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {
		// Open file
		// os.Open returns a *os.File (pointer), which is 
		// essentially a pointer to a read-only file descriptor
		file := OpenFile(path)
		defer file.Close()
		// compute hash
		hash := HashFile(file)
		cf := File{
			Hash: hex.EncodeToString(hash), 
			Path: path,
		}
		f, exists := FileList[cf.Hash]
		if exists {
			fmt.Printf("%s is a duplicate of %s", cf.Path, f.Path)
		} else {
			FileList[cf.Hash] = &cf
			fmt.Printf("%s == %s\n", cf.Path, cf.Hash)
		}
		// check for duplicates, add to list if not
		//if slices.ContainsFunc(FileList, func(f File) bool {
		//	return f.Hash == cf.Hash
		//}) {
		//	fmt.Printf("File %s is a duplicate ", cf.Path)
		//} else {
		//	FileList = append(FileList, cf)
		//}
		//fmt.Printf("%s --> %s\n", cf.Hash, cf.Path)

	}
	return nil
}

func main() {
	//cpuFile, err := os.Create("cpu.prof")
	//if err != nil {
	//	panic(err)
	//}
	//pprof.StartCPUProfile(cpuFile)
	//defer pprof.StopCPUProfile()
	filepath.WalkDir(".", walker)
}
