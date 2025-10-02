// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package markup

import (
	"bufio"
	"bytes"
	"html/template"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"forgejo.org/modules/charset"
	"forgejo.org/modules/highlight"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/translation"

	participle "github.com/alecthomas/participle/v2"
	lexer "github.com/alecthomas/participle/v2/lexer"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// String is a convenience function to return a `string` pointer.
func Pointer[P any](p P) *P {
	return &p
}

// filePreviewPattern matches "http://domain/org/repo/src/commit/COMMIT/filepath#L1-L2"
var filePreviewPattern = regexp.MustCompile(`https?://((?:\S+/){3})src/commit/([0-9a-f]{4,64})/(\S+)#(L\d+(?:-L?\d+)?)`)

// FilePreviewPath is a parser struct used for parsing the components of a file preview URL path.
type FilePreviewPath struct {
	OrgRepo    OrgRepo  `parser:"@@"`
	CommitHash string   `parser:"PathSep 'src' PathSep 'commit' PathSep @CommitHash"`
	FilePath   []string `parser:"(PathSep @Path)+"`
}

func (f FilePreviewPath) File() string {
	return strings.Join(f.FilePath, "/")
}

// OrgRepo represents the organization and repository.
type OrgRepo struct {
	Parts []string `parser:"((?! PathSep 'src' PathSep 'commit') PathSep @Path)+"`
}

// IsExternal gets whether the repository is for an external organization.
func (p OrgRepo) IsExternal() bool {
	return len(p.Parts) > 2
}

// External gets the optional external portion of the `OrgRepo`.
//
// Returns a `nil` pointer for local organization repositories.
func (p OrgRepo) External() *string {
	var ret *string

	if p.IsExternal() {
		ret = Pointer(strings.Join(p.Parts[:len(p.Parts)-2], "/"))
	}

	return ret
}

// Org gets the organization.
func (p OrgRepo) Org() string {
	return p.Parts[len(p.Parts)-2]
}

// Repo gets the repository.
func (p OrgRepo) Repo() string {
	return p.Parts[len(p.Parts)-1]
}

// Path gets the full `OrgRepo` path.
func (p OrgRepo) Path() string {
	return strings.Join(p.Parts, "/")
}

// LineNumbers represents a parser struct for parsing file preview path line numbers.
type LineNumbers struct {
	Begin int  `parser:"LinePre @LineNumber+"`
	End   *int `parser:"(LineSep LinePre? @LineNumber+)?"`
}

// filePreviewPathLexer is a Participle lexer struct defining valid lexing grammar for a file preview path.
var filePreviewPathLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "CommitHash", Pattern: `[0-9a-fA-F]{4,64}`},
	{Name: "PathSep", Pattern: `/`},
	// inexact match for ASCII alphanumeric chars, special characters, and \p{L} + \p{N} for international alphanumeric codepoints
	{Name: "Path", Pattern: `[a-zA-Z\w\p{L}\p{N}_\.\-\%:]+`},
})

var lineNumbersLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "LinePre", Pattern: `L`},
	{Name: "LineSep", Pattern: `-`},
	{Name: "LineNumber", Pattern: `[0-9]`},
})

// FilePreviewPathParser is a Participle parser struct for parsing a valid file preview path.
var FilePreviewPathParser = participle.MustBuild[FilePreviewPath](
	participle.Lexer(filePreviewPathLexer),
)

// LineNumbersParser is a Participle parser struct for parsing valid file preview line numbers.
var LineNumbersParser = participle.MustBuild[LineNumbers](
	participle.Lexer(lineNumbersLexer),
)

type FilePreview struct {
	fileContent []template.HTML
	title       template.HTML
	subTitle    template.HTML
	lineOffset  int
	start       int
	end         int
	isTruncated bool
}

func NewFilePreviews(ctx *RenderContext, node *html.Node, locale translation.Locale) []*FilePreview {
	if setting.FilePreviewMaxLines == 0 {
		// Feature is disabled
		return nil
	}

	mAll := filePreviewPattern.FindAllStringSubmatchIndex(node.Data, -1)
	if mAll == nil {
		return nil
	}

	result := make([]*FilePreview, 0)

	for _, m := range mAll {
		if slices.Contains(m, -1) {
			continue
		}

		preview := newFilePreview(ctx, node, locale, m)
		if preview != nil {
			result = append(result, preview)
		}
	}

	return result
}

func newFilePreview(ctx *RenderContext, node *html.Node, locale translation.Locale, m []int) *FilePreview {
	preview := &FilePreview{}

	urlFull := node.Data[m[0]:m[1]]

	pURL, err := url.Parse(urlFull)
	// Invalid URL
	if err != nil {
		log.Error("invalid URL: %v", err)
		return nil
	}

	if pURL.Path == "" {
		return nil
	}

	path, err := FilePreviewPathParser.ParseString("", pURL.EscapedPath())
	// Invalid file preview path
	if err != nil {
		log.Error("invalid URL file preview path: %v", err)
		return nil
	}

	// Ensure that we only use links to local repositories
	if !strings.HasPrefix(urlFull, setting.AppURL) {
		return nil
	}

	ownerName := path.OrgRepo.Org()
	repoName := path.OrgRepo.Repo()

	appURL, err := url.Parse(setting.AppURL)
	if err != nil {
		log.Error("Invalid App URL: %v", err)
		return nil
	}

	appPath := strings.TrimPrefix(appURL.Path, "/")
	projPath := strings.TrimPrefix(path.OrgRepo.Path(), appPath)
	if len(strings.Split(projPath, "/")) != 2 {
		return nil
	}

	commitSha := path.CommitHash
	filePath := path.File()
	urlFullSource := urlFull
	displaySrc := "display=source"

	if Type(filePath) != "" && !strings.Contains(pURL.RawQuery, displaySrc) {
		if pURL.RawQuery != "" {
			displaySrc = "&" + displaySrc
		}
		pURL.RawQuery += displaySrc
		urlFullSource = pURL.String()
	}
	filePath, err = url.QueryUnescape(filePath)
	if err != nil {
		return nil
	}

	preview.start = m[0]
	preview.end = m[1]

	var language string
	fileBlob, err := DefaultProcessorHelper.GetRepoFileBlob(
		ctx.Ctx,
		ownerName,
		repoName,
		commitSha, filePath,
		&language,
	)
	if err != nil {
		return nil
	}

	titleBuffer := new(bytes.Buffer)

	isExternRef := ownerName != ctx.Metas["user"] || repoName != ctx.Metas["repo"]
	if isExternRef {
		link := ownerName + "/" + repoName
		exURL := pURL.Scheme + "://" + pURL.Host + "/" + link + "/"

		err = html.Render(titleBuffer, createLink(exURL, link, ""))
		if err != nil {
			log.Error("failed to render repoLink: %v", err)
		}
		titleBuffer.WriteString(" &ndash; ")
	}

	err = html.Render(titleBuffer, createLink(urlFullSource, filePath, "muted"))
	if err != nil {
		log.Error("failed to render filepathLink: %v", err)
	}

	preview.title = template.HTML(titleBuffer.String())

	commitLinkBuffer := new(bytes.Buffer)
	commitLinkText := commitSha[0:7]

	commitPath := ownerName + "/" + repoName + "/src/commit/" + path.CommitHash
	if external := path.OrgRepo.External(); external != nil {
		commitPath = *external + "/" + commitPath
	}
	commitURL := pURL.Scheme + "://" + pURL.Host + "/" + commitPath

	if isExternRef {
		commitLinkText = ownerName + "/" + repoName + "@" + commitLinkText
	}

	err = html.Render(commitLinkBuffer, createLink(commitURL, commitLinkText, "text black"))
	if err != nil {
		log.Error("failed to render commitLink: %v", err)
	}

	var startLine, endLine int

	if len(pURL.Fragment) > 0 {
		lines, err := LineNumbersParser.ParseString("", pURL.Fragment)
		if err != nil {
			log.Error("failed to parse line numbers: %v", err)
		}
		startLine = lines.Begin
		endLine = startLine
		if lines.End == nil {
			preview.subTitle = locale.Tr(
				"markup.filepreview.line", startLine,
				template.HTML(commitLinkBuffer.String()),
			)

			preview.lineOffset = startLine - 1
		} else {
			endLine = *lines.End
			preview.subTitle = locale.Tr(
				"markup.filepreview.lines", startLine, endLine,
				template.HTML(commitLinkBuffer.String()),
			)

			preview.lineOffset = startLine - 1
		}
	}

	lineCount := endLine - (startLine - 1)
	if startLine < 1 || endLine < 1 || lineCount < 1 {
		return nil
	}

	if setting.FilePreviewMaxLines > 0 && lineCount > setting.FilePreviewMaxLines {
		preview.isTruncated = true
		lineCount = setting.FilePreviewMaxLines
	}

	dataRc, err := fileBlob.DataAsync()
	if err != nil {
		return nil
	}
	defer dataRc.Close()

	reader := bufio.NewReader(dataRc)

	// skip all lines until we find our startLine
	for i := 1; i < startLine; i++ {
		_, err := reader.ReadBytes('\n')
		if err != nil {
			return nil
		}
	}

	// capture the lines we're interested in
	lineBuffer := new(bytes.Buffer)
	for i := 0; i < lineCount; i++ {
		buf, err := reader.ReadBytes('\n')
		if err == nil || err == io.EOF {
			lineBuffer.Write(buf)
		}
		if err != nil {
			break
		}
	}

	// highlight the file...
	fileContent, _, err := highlight.File(fileBlob.Name(), language, lineBuffer.Bytes())
	if err != nil {
		log.Error("highlight.File failed, fallback to plain text: %v", err)
		fileContent = highlight.PlainText(lineBuffer.Bytes())
	}
	preview.fileContent = fileContent

	return preview
}

func (p *FilePreview) CreateHTML(locale translation.Locale) *html.Node {
	table := &html.Node{
		Type: html.ElementNode,
		Data: atom.Table.String(),
		Attr: []html.Attribute{{Key: "class", Val: "file-preview"}},
	}
	tbody := &html.Node{
		Type: html.ElementNode,
		Data: atom.Tbody.String(),
	}

	status := &charset.EscapeStatus{}
	statuses := make([]*charset.EscapeStatus, len(p.fileContent))
	for i, line := range p.fileContent {
		statuses[i], p.fileContent[i] = charset.EscapeControlHTML(line, locale, charset.FileviewContext)
		status = status.Or(statuses[i])
	}

	for idx, code := range p.fileContent {
		tr := &html.Node{
			Type: html.ElementNode,
			Data: atom.Tr.String(),
		}

		lineNum := strconv.Itoa(p.lineOffset + idx + 1)

		tdLinesnum := &html.Node{
			Type: html.ElementNode,
			Data: atom.Td.String(),
			Attr: []html.Attribute{
				{Key: "class", Val: "lines-num"},
			},
		}
		spanLinesNum := &html.Node{
			Type: html.ElementNode,
			Data: atom.Span.String(),
			Attr: []html.Attribute{
				{Key: "data-line-number", Val: lineNum},
			},
		}
		tdLinesnum.AppendChild(spanLinesNum)
		tr.AppendChild(tdLinesnum)

		if status.Escaped {
			tdLinesEscape := &html.Node{
				Type: html.ElementNode,
				Data: atom.Td.String(),
				Attr: []html.Attribute{
					{Key: "class", Val: "lines-escape"},
				},
			}

			if statuses[idx].Escaped {
				btnTitle := ""
				if statuses[idx].HasInvisible {
					btnTitle += locale.TrString("repo.invisible_runes_line") + " "
				}
				if statuses[idx].HasAmbiguous {
					btnTitle += locale.TrString("repo.ambiguous_runes_line")
				}

				escapeBtn := &html.Node{
					Type: html.ElementNode,
					Data: atom.Button.String(),
					Attr: []html.Attribute{
						{Key: "class", Val: "toggle-escape-button btn interact-bg"},
						{Key: "title", Val: btnTitle},
					},
				}
				tdLinesEscape.AppendChild(escapeBtn)
			}

			tr.AppendChild(tdLinesEscape)
		}

		tdCode := &html.Node{
			Type: html.ElementNode,
			Data: atom.Td.String(),
			Attr: []html.Attribute{
				{Key: "class", Val: "lines-code chroma"},
			},
		}
		codeInner := &html.Node{
			Type: html.ElementNode,
			Data: atom.Code.String(),
			Attr: []html.Attribute{{Key: "class", Val: "code-inner"}},
		}
		codeText := &html.Node{
			Type: html.RawNode,
			Data: string(code),
		}
		codeInner.AppendChild(codeText)
		tdCode.AppendChild(codeInner)
		tr.AppendChild(tdCode)

		tbody.AppendChild(tr)
	}

	table.AppendChild(tbody)

	twrapper := &html.Node{
		Type: html.ElementNode,
		Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: "class", Val: "ui table"}},
	}
	twrapper.AppendChild(table)

	header := &html.Node{
		Type: html.ElementNode,
		Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: "class", Val: "header"}},
	}

	ptitle := &html.Node{
		Type: html.ElementNode,
		Data: atom.Div.String(),
	}
	ptitle.AppendChild(&html.Node{
		Type: html.RawNode,
		Data: string(p.title),
	})
	header.AppendChild(ptitle)

	psubtitle := &html.Node{
		Type: html.ElementNode,
		Data: atom.Span.String(),
		Attr: []html.Attribute{{Key: "class", Val: "text grey"}},
	}
	psubtitle.AppendChild(&html.Node{
		Type: html.RawNode,
		Data: string(p.subTitle),
	})
	header.AppendChild(psubtitle)

	node := &html.Node{
		Type: html.ElementNode,
		Data: atom.Div.String(),
		Attr: []html.Attribute{{Key: "class", Val: "file-preview-box"}},
	}
	node.AppendChild(header)

	if p.isTruncated {
		warning := &html.Node{
			Type: html.ElementNode,
			Data: atom.Div.String(),
			Attr: []html.Attribute{{Key: "class", Val: "ui warning message tw-text-left"}},
		}
		warning.AppendChild(&html.Node{
			Type: html.TextNode,
			Data: locale.TrString("markup.filepreview.truncated"),
		})
		node.AppendChild(warning)
	}

	node.AppendChild(twrapper)

	return node
}
