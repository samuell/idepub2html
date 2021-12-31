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
	// Define flags
	xHtmlFile := flag.String("xhtmlfile", "", "Input file in XML format")
	cssFile := flag.String("cssfile", "", "Input file in XML format")
	flag.Parse()

	if *xHtmlFile == "" {
		fmt.Printf("No input file specified! Use the -h flag to view options\n")
		os.Exit(1)
	}
	if *cssFile == "" {
		fmt.Printf("No CSS file specified! Use the -h flag to view options\n")
		os.Exit(1)
	}

	// Read CSS File
	cssByte, err := os.ReadFile(*cssFile)
	if err != nil {
		panic(err)
	}
	css := string(cssByte)
	cssData := parseCSS(css)

	// Read xHtmlFile
	doc := etree.NewDocument()
	err = doc.ReadFromFile(*xHtmlFile)
	if err != nil {
		panic(err)
	}

	// Parse document
	html := doc.SelectElement("html")
	if body := html.SelectElement("body"); body != nil {
		text := elemsToSimpleHTML(body.ChildElements())
		fmt.Println(text)
	} else {
		fmt.Println("Body was empty!")
	}
}

// elemsToSimpleHTML returns the consecutive text content of all children,
// being processed recursively. It preserves only very basic HTML tags such as
// for paragraphs, italic and bold text etc.
func elemsToSimpleHTML(elems []*etree.Element) (text string) {
	for _, e := range elems {
		if e.Tag == "p" {
			text += "<p>"
		}
		children := e.ChildElements()
		if children != nil {
			text += elemsToSimpleHTML(children)
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

func parseCSS(css string) map[string]map[string]string {
	cssData := map[string]map[string]string{}
	ruleRe := regexp.MustCompile("(?s)([@A-Za-z0-9-\\.]+) {([^}]+)}")

	matches := ruleRe.FindAllString(css, -1)
	for _, m := range matches {
		parts := ruleRe.FindStringSubmatch(m)
		name := parts[1]

		propData := map[string]string{}
		content := parts[2]
		properties := strings.Split(content, "\n")
		for _, prop := range properties {
			prop = strings.TrimSpace(prop)
			prop = strings.TrimRight(prop, ";")
			kv := strings.Split(prop, ":")
			if len(kv) > 1 {
				propData[kv[0]] = kv[1]
			}
		}
		cssData[name] = propData
	}
	return cssData
}
