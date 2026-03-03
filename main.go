package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	extList   map[string]struct{}
	debugFlag = flag.Bool("debug", false, "enable debug logging")
)

func debugf(format string, args ...interface{}) {
	if !*debugFlag {
		return
	}
	fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
}

// isMatchingExtension reports whether the file path has one of the wanted extensions.
func isMatchingExtension(path string, extMap map[string]struct{}) bool {
	if _, ok := extMap[filepath.Ext(path)]; ok {
		return true
	}
	return false
}

// hashFile returns the SHA-256 hash of the given file's contents as a hex string.
func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	sum := h.Sum(nil)
	return hex.EncodeToString(sum), nil
}

// worker consumes file paths from the channel, hashes them, and reports duplicates.
func worker(paths <-chan string, fileMap map[string]string, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	for path := range paths {
		debugf("hashing %s", path)

		hash, err := hashFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			continue
		}

		mu.Lock()
		if firstPath, exists := fileMap[hash]; exists {
			debugf("duplicate detected: %s and %s", path, firstPath)
			fmt.Printf("File %s is a duplicate of %s\n", path, firstPath)
		} else {
			fileMap[hash] = path
		}
		mu.Unlock()
	}
}

func main() {
	flag.Parse()

	debugf("debug logging enabled")

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s ext1,ext2,ext3\n", os.Args[0])
		os.Exit(1)
	}

	// Initialize the extList map
	extList = make(map[string]struct{})

	extensions := strings.Split(flag.Arg(0), ",")
	for _, ext := range extensions {
		if ext == "" {
			continue
		}
		extList[ext] = struct{}{}
	}

	fileMap := make(map[string]string)
	var mu sync.Mutex

	paths := make(chan string, 1024)

	// Start a bounded number of workers to hash files concurrently.
	workerCount := runtime.NumCPU() * 4
	if workerCount < 1 {
		workerCount = 1
	}

	var wg sync.WaitGroup
	wg.Add(workerCount)
	debugf("starting %d workers", workerCount)
	for i := 0; i < workerCount; i++ {
		go worker(paths, fileMap, &mu, &wg)
	}

	debugf("starting directory walk")
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isMatchingExtension(path, extList) {
			debugf("queued %s", path)
			paths <- path
		}
		return nil
	})

	close(paths)
	wg.Wait()

	debugf("directory walk complete")

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
