package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	// Format output directory path
	outDir := strings.TrimSuffix(*epubFile, ".epub")

	zipRd, err := zip.OpenReader(*epubFile)
	check(err, "Could not read ZIP file")
	defer zipRd.Close()

	// Read CSS Data
	var cssByte []byte
	var cssData map[string]map[string]string
	for _, f := range zipRd.File {
		if !f.FileInfo().IsDir() && strings.HasSuffix(f.Name, ".css") {
			zippedFile, err := f.Open()
			check(err, "Could not open css file")
			cssByte, err = io.ReadAll(zippedFile)
			check(err, "Could not read css file")
			css := string(cssByte)
			cssData = parseCSS(css)
			defer zippedFile.Close()
		}
	}

	// Unpack images
	imgRe := regexp.MustCompile(".*\\.(jpg|jpeg|gif|png|bmp|svg)")
	for _, f := range zipRd.File {
		if !f.FileInfo().IsDir() && imgRe.MatchString(f.Name) {
			imgDir := filepath.Join(outDir, "image")

			err := os.MkdirAll(imgDir, os.ModePerm)
			check(err, "Could not create img dir")

			destPath := filepath.Join(imgDir, filepath.Base(f.Name))
			if len(destPath) > 150 {
				destPath = destPath[0:140] + "." + destPath[len(destPath)-3:]
			}
			destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			check(err, "Could not create destination image file")
			defer destFile.Close()

			imgFile, err := f.Open()
			check(err, "Could not open input image file")

			_, err = io.Copy(destFile, imgFile)
			check(err, "Could not copy image file to destination")
		}
	}

	outHtmlPath := filepath.Join(outDir, "index.html")
	outHtmlFile, err := os.Create(outHtmlPath)
	check(err, "Could not create output HTML file")
	defer outHtmlFile.Close()

	// Parse XHTML files
	outHtml := ""
	for _, f := range zipRd.File {
		if !f.FileInfo().IsDir() && strings.HasSuffix(f.Name, ".xhtml") {
			doc := etree.NewDocument()
			xHtmlFile, err := f.Open()
			check(err, "Could not open input XHTML file")

			xHtmlByte, err := io.ReadAll(xHtmlFile)
			check(err, "Could not read input XHTML file")

			err = doc.ReadFromBytes(xHtmlByte)
			check(err, "Could not read XHTML file into ElementTree")
			// Parse document
			html := doc.SelectElement("html")
			if body := html.SelectElement("body"); body != nil {
				text := elemsToSimpleHTML(body.ChildElements(), cssData)
				text = replaceNonStandardChars(text)
				text = cleanUpReopeningTags(text)
				text = cleanUpLineBreakDashes(text)
				text = convertSectionHeadings(text)
				text = cleanUpWhiteSpace(text)
				text = convertNoteNumbers(text)
				outHtml += text
				outHtml += "\n<hr>\n"
			} else {
				fmt.Println("WARNING: Body was empty!")
			}
		}
	}
	_, err = outHtmlFile.WriteString(outHtml)
	check(err, "Could not Write to output HTML file")
	outHtmlFile.Sync()
	fmt.Printf("Wrote output HTML to: %s\n", outHtmlPath)
	fmt.Println("(To view the file, open it in a web browser!)")
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
			imgStr += `style="max-width: 240px; max-height: 240px;" />`
			text += imgStr
		}

		if e.Tag == "div" && strings.HasPrefix(e.SelectAttrValue("id", ""), "_idContainer") {
			text += "<hr>\n"
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

func convertSectionHeadings(text string) string {
	headingRe := regexp.MustCompile(`<b>(([A-ZÅÄÖ0-9"?\.,\-]{1,} ?)+)</b>`)
	ms := headingRe.FindAllStringSubmatch(text, -1)
	for _, m := range ms {
		title := strings.ToUpper(string(m[1][0])) + strings.ToLower(m[1][1:])
		text = strings.ReplaceAll(text, m[0], fmt.Sprintf("<h3>%s</h3>", title))
	}

	return text
}

func check(err error, msg string) {
	if err != nil {
		fmt.Println("ERROR: " + msg)
		panic(err)
	}
}

func convertNoteNumbers(text string) string {
	re := regexp.MustCompile(`([\.,]"? )(([0-9]{1,2}[-,]?)+)`)
	ms := re.FindAllStringSubmatch(text, -1)
	for _, m := range ms {
		text = strings.ReplaceAll(text, m[0], fmt.Sprintf("%s[%s]", m[1], m[2]))
	}
	return text
}

func replaceNonStandardChars(text string) string {
	text = strings.ReplaceAll(text, `”`, `"`)
	text = strings.ReplaceAll(text, `–`, `-`)
	return text
}
