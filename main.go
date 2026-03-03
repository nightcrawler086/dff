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
	extList     map[string]struct{}
	debugFlag   = flag.Bool("debug", false, "enable debug logging")
	machineFlag = flag.Bool("machine", false, "machine-readable output (tab-separated: hash,dupPath,origPath)")
	workersFlag = flag.Int("workers", 0, "number of worker goroutines (default: logical CPUs)")
)

type duplicateRecord struct {
	Hash      string
	Duplicate string
	Original  string
}

func debugf(format string, args ...interface{}) {
	if !*debugFlag {
		return
	}
	fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
}

// isMatchingExtension reports whether the file path has one of the wanted extensions.
func isMatchingExtension(path string, extMap map[string]struct{}) bool {
	if len(extMap) == 0 {
		// No extensions specified: match all files.
		return true
	}
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
func worker(paths <-chan string, fileMap map[string]string, duplicates *[]duplicateRecord, mu *sync.Mutex, wg *sync.WaitGroup) {
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
			if *machineFlag {
				// padded columns for aligned output: hash, duplicate-path, original-path
				fmt.Printf("%-64s\t%s\t%s\n", hash, path, firstPath)
			} else {
				*duplicates = append(*duplicates, duplicateRecord{
					Hash:      hash,
					Duplicate: path,
					Original:  firstPath,
				})
			}
		} else {
			fileMap[hash] = path
		}
		mu.Unlock()
	}
}

func main() {
	flag.Parse()

	debugf("debug logging enabled")

	// Initialize the extList map
	extList = make(map[string]struct{})

	if flag.NArg() >= 1 {
		extensions := strings.Split(flag.Arg(0), ",")
		for _, ext := range extensions {
			if ext == "" {
				continue
			}
			extList[ext] = struct{}{}
		}
	}

	fileMap := make(map[string]string)
	var (
		mu         sync.Mutex
		duplicates []duplicateRecord
	)

	paths := make(chan string, 1024)

	// Start a bounded number of workers to hash files concurrently.
	workerCount := *workersFlag
	if workerCount <= 0 {
		workerCount = runtime.NumCPU() // logical CPUs = cores * threads
	}

	if *machineFlag {
		// Print header with padded hash column so headings align with values.
		fmt.Printf("%-64s\t%s\t%s\n", "HASH", "DUPLICATE", "ORIGINAL")
	}

	var wg sync.WaitGroup
	wg.Add(workerCount)
	debugf("starting %d workers", workerCount)
	for i := 0; i < workerCount; i++ {
		go worker(paths, fileMap, &duplicates, &mu, &wg)
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

	if !*machineFlag {
		if len(duplicates) == 0 {
			fmt.Println("No duplicate files found.")
		} else {
			fmt.Println("HASH\tDUPLICATE\tORIGINAL")
			for _, d := range duplicates {
				fmt.Printf("%s\t%s\t%s\n", d.Hash, d.Duplicate, d.Original)
			}
		}
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
