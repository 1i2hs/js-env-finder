package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/js-env-finder/internal/command"
)

type CommandFlags struct {
	workerCount int
	paths       []string
	exclude     []string
}

func getFlags() CommandFlags {
	c := CommandFlags{
		workerCount: 0,
		paths:       []string{"."},
		exclude:     []string{},
	}

	// worker count, default count is 4
	workerCount := flag.Int("wc", 4, "specifies how much parallel worker threads to use to find env. variables. DEFAULT: 4")

	var excludeArr command.StringArray
	flag.Var(&excludeArr, "exclude", "paths to be excluded during the search")

	// parse flags
	flag.Parse()

	c.workerCount = *workerCount

	c.exclude = excludeArr

	// paths: using flag.Args
	if len(flag.Args()) != 0 {
		c.paths = flag.Args()
	}

	return c
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func isWhitespace(charByte byte) bool {
	const spaces = " \n\t" // whitespace, new line, tab
	spaceFound := false
	for i := 0; i < len(spaces); i++ {
		if spaces[i] == charByte {
			spaceFound = true
			break
		}
	}
	return spaceFound
}

func isEndToken(tokenChecker *regexp.Regexp, charByte byte) bool {
	return !tokenChecker.MatchString(string(charByte))
}

func findFromSingleFile(id int, filePaths <-chan string, results chan<- []string) {
	for filePath := range filePaths {
		fmt.Printf("Worker %d processing file: %s\n", id, filePath)
		js, err := os.Open(filePath)
		check(err)

		reader := bufio.NewReader(js)

		tokenChecker, _ := regexp.Compile("[0-9]|[a-zA-Z]|_")
		const processEnvString = "process.env."
		index := 0
		consumer := make([]string, 0)
		isConsuming := false
		temp := make([]byte, 0)

		for {
			block := make([]byte, 1)
			_, err := reader.Read(block)
			if err != nil {
				break
			}
			// match process.env.
			if index <= len(processEnvString)-1 && block[0] == processEnvString[index] {
				index++
				continue
			}

			// check if there any whitespaces between process.env. & environment variable name
			if index > len(processEnvString)-1 {
				if !isConsuming && !isWhitespace(block[0]) {
					// start of consuming
					isConsuming = true
				}

				if isConsuming && isEndToken(tokenChecker, block[0]) {
					// end of consuming
					isConsuming = false
					index = 0
					// fmt.Printf("Consumed env: %s\n", temp)
					consumer = append(consumer, string(temp))
					temp = make([]byte, 0)
					continue
				}

				if isConsuming {
					temp = append(temp, block[0])
				}
			}
		}
		// fmt.Printf("%d Worker ended: %s\n", id, filePath)
		results <- consumer
	}
}

func getFilePaths(path string, excludes []string) chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)
		fileFormatChecker, _ := regexp.Compile(`^.*\.(js|jsx|ts|tsx)$`)

		err := filepath.WalkDir(path, func(subpath string, entry os.DirEntry, err error) error {
			if err != nil {
				fmt.Printf("Error during accessing %v, maybe the path does not exist(Error: %v)\n", subpath, err)
				return nil
			}
			// TODO glob check
			// for _, ex := range excludes {
			// 	if ex == info.Name() {
			// 		fmt.Printf("Skipping the path '%v', since it is in the exclusion list\n", info.Name())
			// 		return filepath.SkipDir
			// 	}
			// }

			if !entry.IsDir() && fileFormatChecker.MatchString(entry.Name()) {
				ch <- subpath
			}

			return nil
		})
		check(err)
	}()

	return ch
}

func main() {
	cFlags := getFlags()
	WORKER_COUNT := cFlags.workerCount
	EXCLUDES := cFlags.exclude

	fmt.Printf("Number of parallel workers: %d\n", WORKER_COUNT)

	paths := cFlags.paths
	filePathChannel := make(chan string, WORKER_COUNT)
	resultChannel := make(chan []string, WORKER_COUNT)
	numFound := 0

	for w := 1; w <= WORKER_COUNT; w++ {
		go findFromSingleFile(w, filePathChannel, resultChannel)
	}

	for _, path := range paths {
		fmt.Printf("Working on a path: %s\n", path)
		pathInfo, err := os.Stat(path)
		if os.IsNotExist(err) {
			fmt.Printf("Path %s does not exist\n", path)
			continue
		}
		if !pathInfo.IsDir() {
			filePathChannel <- path
			continue
		}
		// generator pattern
		for filePath := range getFilePaths(path, EXCLUDES) {
			numFound++
			filePathChannel <- filePath
		}
	}
	close(filePathChannel)

	all := make([]string, 0)

	for j := 1; j <= numFound; j++ {
		result := <-resultChannel
		all = append(all, result...)
	}

	kvs := make(map[string]int)
	for _, env := range all {
		count, exists := kvs[env]
		if exists {
			kvs[env] = count + 1
			continue
		}
		kvs[env] = 0
	}

	finalResult := make([]string, 0)
	for key := range kvs {
		finalResult = append(finalResult, key)
	}

	sort.Strings(finalResult)
	fmt.Println(finalResult)
}
