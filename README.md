# Script to convert EPUB files to HTML

## Usage

- Unpack the epub file with unzip:
  ```
  unzip -d <outdir> <epubfile.epub>
  ```

- Move into xhtml folder
  ```
  cd <outdir>/OEBPS/
  ```

- Convert one of the XHTMl files to cleaned up HTML
  ```
  epub2html -infile somefile.xhtml > somefile.cleaned.html
  ```
