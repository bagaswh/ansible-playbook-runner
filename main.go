package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type PlaybookList []string

func readFile(file *os.File, fileStat fs.FileInfo) []byte {
	buf := make([]byte, fileStat.Size())
	for {
		_, err := file.Read(buf)
		if err == io.EOF {
			break
		}
	}
	return buf
}

func main() {
	// Define flag
	var playbookListFile string
	var includePlaybooks string
	var excludePlaybooks string
	var fromPlaybook string
	var toPlaybook string
	var ansiblePlaybookArgs string

	flag.StringVarP(&playbookListFile, "playbook-list-file", "f", "", "The playbook list file")
	flag.StringVarP(&includePlaybooks, "include-playbooks", "p", "", "Specify which playbook files to include")
	flag.StringVarP(&excludePlaybooks, "exclude-playbooks", "x", "", "Specify which playbook files to exclude")
	flag.StringVar(&fromPlaybook, "from", "", "Specify which playbook to start running from")
	flag.StringVar(&toPlaybook, "to", "", "Specify which playbook to end running")
	flag.StringVar(&ansiblePlaybookArgs, "pargs", "a", "ansible-playbook cli args")

	flag.Parse()

	// argsAfterFlags := []string{}
	argCount := len(os.Args)
	fmt.Printf("argCount=%d NFlag=%d\n", argCount, flag.NFlag())
	// fmt.Println(os.Args[flag.NFlag()+2:])

	if playbookListFile == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	playbookListFileHandle, err := os.OpenFile(playbookListFile, os.O_RDONLY, 0600)
	if err != nil {
		log.Fatalf("failed opening playbook list file: %v\n", err)
	}
	defer playbookListFileHandle.Close()

	stat, _ := playbookListFileHandle.Stat()
	if !stat.Mode().IsRegular() {
		log.Fatalf("playbook list file provided is not a regular file\n")
	}

	mib := (2 << 20)
	maxSize := int64(1 * mib)
	if stat.Size() > maxSize {
		log.Fatalf("file size exceeds maximum (by default it's set to 1 MiB)\n")
	}

	playbookListFileContent := readFile(playbookListFileHandle, stat)
	var playbookList PlaybookList
	err = yaml.Unmarshal([]byte(playbookListFileContent), &playbookList)
	if err != nil {
		log.Fatalf("error unmarshalling yaml: %v\n", err)
	}

	var filteredPlaybooks []string
	if includePlaybooks != "" {
		filteredPlaybooks = strings.Split(includePlaybooks, ",")
	} else {
		filteredPlaybooks = playbookList
	}

	if excludePlaybooks != "" {
		excludeList := strings.Split(excludePlaybooks, ",")
		var temp []string
		for _, playbook := range filteredPlaybooks {
			if !contains(excludeList, playbook) {
				temp = append(temp, playbook)
			}
		}
		filteredPlaybooks = temp
	}

	fromPlaybookIndex := 0
	if fromPlaybook != "" {
		fromPlaybookIndex = slices.Index[[]string](filteredPlaybooks, fromPlaybook)
	}
	if fromPlaybookIndex == -1 {
		log.Fatalf("cannot find playbook in --from '%s'\n", fromPlaybook)
	}

	toPlaybookIndex := len(filteredPlaybooks)
	if toPlaybook != "" {
		toPlaybookIndex = slices.Index[[]string](filteredPlaybooks, toPlaybook)
	}
	if toPlaybookIndex == -1 {
		log.Fatalf("cannot find playbook in --to '%s'\n", toPlaybook)
	}

	filteredPlaybooks = filteredPlaybooks[fromPlaybookIndex:toPlaybookIndex]
	fmt.Println("Playbooks to be run:", filteredPlaybooks)

	for _, playbook := range filteredPlaybooks {
		cmd := exec.Command("ansible-playbook", playbook)
		restArgs := strings.Split(ansiblePlaybookArgs, " ")
		if ansiblePlaybookArgs != "" {
			cmd.Args = append(cmd.Args, restArgs...)
		}
		fmt.Printf("%s\n", cmd.String())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			fmt.Fprintf(os.Stderr, "error running playbook %s: %v\n", playbook, err)
			os.Exit(cmd.ProcessState.ExitCode())
		}
	}
}

func contains(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}
