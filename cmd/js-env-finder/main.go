package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func isWhitespace(charByte byte) bool {
	const spaces = " \n\t"
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
		// filePath := <-filePaths
		fmt.Printf("%d Worker started: %s\n", id, filePath)
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
			// line, err := reader.ReadString('\n')
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

			// check if there any whitespaces between process.ev. & environment variable name
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
		fmt.Printf("%d Worker ended: %s\n", id, filePath)
		results <- consumer
	}
}

func main() {
	filePaths := []string{"samples/index.js", "samples/index.js", "samples/index.js"}
	filePathChannel := make(chan string, 4)
	results := make(chan []string, 4)

	for w := 1; w <= 2; w++ {
		go findFromSingleFile(w, filePathChannel, results)
	}

	for i := 0; i <= len(filePaths)-1; i++ {
		filePathChannel <- filePaths[i]
	}
	// for _, filePath := range filePaths {
	// 	filePathChannel <- filePath
	// }
	// close(filePathChannel)

	all := make([]string, 0)
	// for result := range results {
	// 	fmt.Println(len(results))
	// 	fmt.Println((result))
	// 	all = append(all, result...)
	// }
	for j := 1; j <= len(filePaths); j++ {
		result := <-results
		fmt.Println(result)
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
