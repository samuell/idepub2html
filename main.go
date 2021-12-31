package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/beevik/etree"
)

func main() {
	// Configure and parse flags
	epubFile := flag.String("epubfile", "", "EPUB file")
	flag.Parse()
	if *epubFile == "" {
		fmt.Printf("No EPUB file specified! Use the -h flag to view options\n")
		os.Exit(1)
	}

	zipRd, err := zip.OpenReader(*epubFile)
	if err != nil {
		panic(err)
	}
	defer zipRd.Close()

	// Read CSS Data
	var cssByte []byte
	var cssData map[string]map[string]string
	for _, f := range zipRd.File {
		if !f.FileInfo().IsDir() && strings.HasSuffix(f.Name, ".css") {
			zippedFile, err := f.Open()
			if err != nil {
				panic(err)
			}
			cssByte, err = io.ReadAll(zippedFile)
			if err != nil {
				panic(err)
			}
			css := string(cssByte)
			cssData = parseCSS(css)
			defer zippedFile.Close()
		}
	}

	// Parse XHTML files
	for _, f := range zipRd.File {
		if !f.FileInfo().IsDir() && strings.HasSuffix(f.Name, ".xhtml") {
			doc := etree.NewDocument()
			xHtmlFile, err := f.Open()
			if err != nil {
				panic(err)
			}
			xHtmlByte, err := io.ReadAll(xHtmlFile)
			if err != nil {
				panic(err)
			}
			err = doc.ReadFromBytes(xHtmlByte)
			if err != nil {
				panic(err)
			}
			// Parse document
			html := doc.SelectElement("html")
			if body := html.SelectElement("body"); body != nil {
				text := elemsToSimpleHTML(body.ChildElements(), cssData)
				text = cleanUpWhiteSpace(text)
				text = cleanUpReopeningTags(text)
				text = cleanUpLineBreakDashes(text)
				fmt.Println(text)
			} else {
				fmt.Println("Body was empty!")
			}
		}
	}

}

// elemsToSimpleHTML returns the consecutive text content of all children,
// being processed recursively. It preserves only very basic HTML tags such as
// for paragraphs, italic and bold text etc.
func elemsToSimpleHTML(elems []*etree.Element, cssData map[string]map[string]string) (text string) {
	for _, e := range elems {
		closingTags := []string{}

		if e.Tag == "img" {
			imgStr := "<img "
			for _, a := range e.Attr {
				if a.Key == "src" {
					imgStr += fmt.Sprintf("%s=\"%s\" ", a.Key, a.Value)
				}
			}
			imgStr += "/>"
			text += imgStr
		}

		if e.Tag == "div" && strings.HasPrefix(e.SelectAttrValue("id", ""), "_idContainer") {
			text += "<div style=\"border: 4px solid #ccc; margin: 1em; padding: 1em;\">"
			closingTags = append(closingTags, "</div>\n")
		}

		// Convert some tags to simple HTML variants
		if e.Tag == "p" {
			text += "<p>"
			closingTags = append(closingTags, "</p>")
		}
		if e.Tag == "span" {
			classStr := e.SelectAttr("class").Value
			classes := strings.Split(classStr, " ")
			for _, class := range classes {
				cssInfo := cssData[e.Tag+"."+class]
				if cssInfo["font-weight"] == "bold" {
					text += "<b>"
					closingTags = append(closingTags, "</b>")
				}
				if cssInfo["font-style"] == "italic" {
					text += "<i>"
					closingTags = append(closingTags, "</i>")
				}
			}
		}

		children := e.ChildElements()
		if children != nil {
			text += elemsToSimpleHTML(children, cssData)
		}
		text += strings.TrimSpace(e.Text()) + " "

		// Close any tags from the conversion before
		for i := len(closingTags) - 1; i >= 0; i-- {
			text += closingTags[i]
		}
	}

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

func cleanUpWhiteSpace(text string) string {
	re := regexp.MustCompile("[ ]+")
	text = re.ReplaceAllString(text, " ")
	return text
}

func cleanUpLineBreakDashes(text string) string {
	re := regexp.MustCompile("([A-Za-zåäöÅÄÖ]{2,})\\- ([A-Za-zåäöÅÄÖ]{2,})")
	ms := re.FindAllStringSubmatch(text, -1)
	for _, bits := range ms {
		text = strings.ReplaceAll(text, bits[0], bits[1]+bits[2])
	}
	return text
}

func cleanUpReopeningTags(text string) string {
	for _, s := range []string{"i", "b"} {
		text = strings.ReplaceAll(text, fmt.Sprintf("</%s><%s>", s, s), "")
	}
	return text
}
