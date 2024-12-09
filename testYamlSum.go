// testYamSum
// program that test splitting input file into different sections
// testfiles:
// - yaml.md yaml header and md body
// - summary.md  summary section and md body
// - yamlsummary.md yaml header, followed by summary and md body
//

package main

import (
	"fmt"
	"log"
	"os"
    md2js "goDemo/goldmark/samples/rendererV3"
)


func main () {

	numargs := len(os.Args)

	if numargs < 2 {log.Fatalf("need a command!")}


	inpFilnam := ""
	switch os.Args[1] {
	case "meta":
		inpFilnam = "md/yaml.md"

	case "summary":
		inpFilnam = "md/summary.md"

	case "metasummary":
		inpFilnam = "md/yamlsummary.md"

	case "mdonly":
		inpFilnam = "md/mdonly.md"

	case "help":
		fmt.Printf("usage: ./testYamlSum <yaml|summary|yamlsummary|help>\n")
		os.Exit(0)
	default:
		log.Fatalf("invalid command: %s", os.Args[1])
	}

	inp, err := os.ReadFile(inpFilnam)
	if err != nil {log.Fatalf("ReadFile: %v", err)}

	fmt.Printf("Inp:\n%s\n**** end raw ****\n", inp)

	comp, err := md2js.GetMetaSum(inp)
	if err != nil {log.Fatalf("GetMetaSum: %v", err)}

	if comp.Meta != nil {
		fmt.Printf("**** meta:\n%s\n******\n",comp.Meta)
	} else {
		fmt.Printf("**** meta none ****\n")
	}
	if comp.Summary != nil {
		fmt.Printf("**** summary:\n%s\n******\n",comp.Summary)
	} else {
		fmt.Printf("**** summary none ****\n")
	}
	fmt.Printf("**** main:\n%s\n******\n",comp.Main)
}
