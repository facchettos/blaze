package fileUtils

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

//WriteBlock (data, fileName) write the data under filename (fileName includes the path)
func WriteBlock(data []byte, fileName string) error {
	err := ioutil.WriteFile(fileName, data, 0600)
	return err
}

func findRightSubFiles(prefix string) []string {
	files, err := ioutil.ReadDir("./")
	if err != nil {
		log.Fatal(err)
	}
	res := make([]string, 0)
	for _, f := range files {
		filesPrefix := strings.Split(f.Name(), ".")
		if filesPrefix[1] == prefix {
			res = append(res, f.Name())
		}
	}
	sort.Slice(res, func(i, j int) bool {
		resi, err := strconv.Atoi(strings.Split(res[i], ".")[1])
		if err != nil {
			panic(fmt.Sprintf("panic from sorting, impossible to convert file extension to number"))
		}
		resj, err := strconv.Atoi(strings.Split(res[j], ".")[1])
		if err != nil {
			panic(fmt.Sprintf("panic from sorting, impossible to convert file extension to number"))
		}
		return resi < resj
	})
	// TODO: sort the result
	return res
}

//ReassembleFiles takes all the parts of a file and writes them back in order
func ReassembleFiles(destination string, subfiles []string, buf *bytes.Buffer) error {
	f, err := os.OpenFile(destination, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, sf := range subfiles {
		part, err := ioutil.ReadFile(sf)
		if err != nil {
			return errors.New("impossible to read part " + sf)
		}
		bytesWritten, err := buf.Write(part)
		if bytesWritten != len(part) || err != nil {
			return errors.New("problem when writing to buffer part " + sf)
		}

	}
	return nil
}
