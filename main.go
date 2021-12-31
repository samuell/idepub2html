package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/beevik/etree"
)

func main() {
	inFile := flag.String("infile", "", "Input file in XML format")
	flag.Parse()

	if *inFile == "" {
		fmt.Printf("No input file specified! Use the -h flag to view options\n")
		os.Exit(1)
	}

	doc := etree.NewDocument()
	err := doc.ReadFromFile(*inFile)
	if err != nil {
		panic(err)
	}
}
