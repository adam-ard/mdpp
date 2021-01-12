package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type CmdBlock struct {
	Id   int
	Dir  string
	Cmds []string
}

type ResultBlock struct {
	SrcId int
	Auto  bool
}

var cmdBlocks map[int]CmdBlock

func next(scanner *bufio.Scanner) string {
	scanner.Scan()
	return scanner.Text()
}

func assembleCmd(dir string, cmds []string) string {
	cmds = append([]string{"cd " + dir}, cmds...)
	return strings.Join(cmds, " && ")
}

func shellout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("/usr/bin/bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

func parseCmd(scanner *bufio.Scanner, line string, re *regexp.Regexp) {
	indexes := re.FindStringSubmatchIndex(line)

	blockStart := indexes[0] + 8 // index of match plus len(MDPP_CMD)

	var cmdBlock CmdBlock
	json.Unmarshal([]byte(line[blockStart:]), &cmdBlock)

	// print truncated line
	fmt.Println(line[:indexes[0]])

	cmds := []string{}
	for {
		line = next(scanner)
		fmt.Println(line)

		if line == "```" {
			break
		}

		cmds = append(cmds, line)
	}

	// store the block
	cmdBlock.Cmds = cmds
	cmdBlocks[cmdBlock.Id] = cmdBlock
}

func parseResult(scanner *bufio.Scanner, line string, re *regexp.Regexp) {
	indexes := re.FindStringSubmatchIndex(line)

	blockStart := indexes[0] + 11 // index of match plus len(MDPP_RESULT)

	var resultBlock ResultBlock
	json.Unmarshal([]byte(line[blockStart:]), &resultBlock)

	fmt.Println(line[:indexes[0]])

	assembledCmd := assembleCmd(cmdBlocks[resultBlock.SrcId].Dir, cmdBlocks[resultBlock.SrcId].Cmds)

	if resultBlock.Auto {
		err, out, errOut := shellout(assembledCmd)
		if err != nil {
			log.Printf("error: %v\n", err)
		}
		fmt.Print(out)
		fmt.Print(errOut)
	} else {
		fmt.Fprintln(os.Stderr, "Run in a terminal, and copy and paste results")
		fmt.Fprintln(os.Stderr, assembledCmd)

		// TODO: switch this to just a repl, like the one I built already (service-dependency-map)
		// fo now, I can just hit Ctrl-D to end after pasting
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\r')

		fmt.Print(text)
	}

	// skip the rest, until the end of the block
	for {
		line = next(scanner)

		if line == "```" {
			fmt.Println(line)
			break
		}
	}

}

func main() {
	cmdBlocks = map[int]CmdBlock{}

	reCmd := regexp.MustCompile(`MDPP_CMD{.*}`)
	reResult := regexp.MustCompile(`MDPP_RESULT{.*}`)

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		currLine := scanner.Text()

		switch {
		case reCmd.MatchString(currLine):
			parseCmd(scanner, currLine, reCmd)
		case reResult.MatchString(currLine):
			parseResult(scanner, currLine, reResult)
		default:
			fmt.Println(currLine)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
