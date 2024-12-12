// simpleMd2JsConv.go
// program that converts markdown files into html files
// ./simpleMdCon /in=infile.md /out=outfile.html [/dbg]
// uses goldmark: github.com/yuin/goldmark
//
// author: prr, azul software
// date: 13 Nov 2024
// copyright prr, azul software
//

package main

import (
	"fmt"
	"log"
	"os"
	"bytes"

	md2js "goDemo/goldmark/samples/renderer"

	"github.com/yuin/goldmark"
//	"github.com/yuin/goldmark/renderer"
	util "github.com/prr123/utility/utilLib"
)

func main() {

	var buf bytes.Buffer
	
	numarg := len(os.Args)
    flags:=[]string{"dbg", "in", "out"}

    useStr := " /in=infile /out=outfile [/dbg]"
    helpStr := "markdown to html conversion program"

    if numarg > len(flags) +1 {
        fmt.Println("too many arguments in cl!")
        fmt.Println("usage: %s %s\n", os.Args[0], useStr)
        os.Exit(-1)
    }

    if numarg == 1 || (numarg > 1 && os.Args[1] == "help") {
        fmt.Printf("help: %s\n", helpStr)
        fmt.Printf("usage is: %s %s\n", os.Args[0], useStr)
        os.Exit(1)
    }

    flagMap, err := util.ParseFlags(os.Args, flags)
    if err != nil {log.Fatalf("util.ParseFlags: %v\n", err)}

    dbg:= false
    _, ok := flagMap["dbg"]
    if ok {dbg = true}

    inFil := ""
    inval, ok := flagMap["in"]
    if !ok {
		log.Fatalf("error -- no in flag provided!\n")
	} else {
        if inval.(string) == "none" {log.Fatalf("error -- no input file name provided!\n")}
        inFil = inval.(string)
    }

    outFil := ""
    outval, ok := flagMap["out"]
    if !ok {
		log.Fatalf("error -- no out flag provided!\n")
	} else {
        if outval.(string) == "none" {log.Fatalf("error -- no output file name provided!\n")}
        outFil = outval.(string)
    }

	inFilnam := "md/" + inFil + ".md"
	outFilnam := "script/" + outFil + ".js"

	if dbg {
		fmt.Printf("input:  %s\n", inFilnam)
		fmt.Printf("output: %s\n", outFilnam)
	}

	source, err := os.ReadFile(inFilnam)
	if err != nil {log.Fatalf("error -- open file: %v\n", err)}

	md2jsRen := md2js.GetRenderer(dbg)
	md := goldmark.New()
	md.SetRenderer(md2jsRen)

// func Convert(source []byte, w io.Writer, opts ...parser.ParseOption) error
//	err = goldmark.Convert(source, &buf, parser.WithContext(ctx))
	err = md.Convert(source, &buf)
	if err != nil {log.Fatalf("error -- convert: %v\n",err)}
	log.Printf("*** success converting ***\n")

	// save
	err = os.WriteFile(outFilnam, buf.Bytes(), 0666)
	if err != nil {log.Fatalf("error -- write file: %v\n")}

	log.Println("*** success ***")
}
