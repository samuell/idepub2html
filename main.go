package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

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

	html := doc.SelectElement("html")
	if body := html.SelectElement("body"); body != nil {
		text := elemsToText(body.ChildElements())
		fmt.Println(text)
	} else {
		fmt.Println("Body was empty!")
	}
}

func elemsToText(elems []*etree.Element) (text string) {
	for _, e := range elems {
		if e.Tag == "p" {
			text += "<p>"
		}
		children := e.ChildElements()
		if children != nil {
			text += elemsToText(children)
		}
		text += strings.TrimSpace(e.Text()) + " "
		if e.Tag == "p" {
			text += "</p>"
		}
	}
	re := regexp.MustCompile("[ ]+")
	text = re.ReplaceAllString(text, " ")
	return text
}
