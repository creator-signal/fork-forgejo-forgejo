// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package markup

import (
	"bufio"
	"bytes"
	"errors"
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

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// filePreviewPattern matches "http://domain/org/repo/src/commit/COMMIT/filepath#L1-L2"
var filePreviewPattern = regexp.MustCompile(`https?://((?:\S+/){3})src/commit/([0-9a-f]{4,64})/(\S+)#(L\d+(?:-L?\d+)?)`)

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

	// Ensure that we only use links to local repositories
	if !strings.HasPrefix(urlFull, setting.AppURL) {
		return nil
	}

	projPath := strings.TrimPrefix(strings.TrimSuffix(node.Data[m[0]:m[3]], "/"), setting.AppURL)

	commitSha := node.Data[m[4]:m[5]]
	filePath := node.Data[m[6]:m[7]]
	hash := node.Data[m[8]:m[9]]
	urlFullSource := urlFull
	if strings.HasSuffix(filePath, "?display=source") {
		filePath = strings.TrimSuffix(filePath, "?display=source")
	} else if Type(filePath) != "" {
		urlFullSource = node.Data[m[0]:m[6]] + filePath + "?display=source#" + hash
	}
	filePath, err := url.QueryUnescape(filePath)
	if err != nil {
		return nil
	}

	preview.start = m[0]
	preview.end = m[1]

	projPathSegments := strings.Split(projPath, "/")
	if len(projPathSegments) != 2 {
		return nil
	}

	ownerName := projPathSegments[len(projPathSegments)-2]
	repoName := projPathSegments[len(projPathSegments)-1]

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
		err = html.Render(titleBuffer, createLink(node.Data[m[0]:m[3]], ownerName+"/"+repoName, ""))
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
	if isExternRef {
		commitLinkText = ownerName + "/" + repoName + "@" + commitLinkText
	}

	err = html.Render(commitLinkBuffer, createLink(node.Data[m[0]:m[5]], commitLinkText, "text black"))
	if err != nil {
		log.Error("failed to render commitLink: %v", err)
	}

	lineSpecs := strings.Split(hash, "-")

	dataRc, err := fileBlob.DataAsync()
	if err != nil {
		return nil
	}
	defer dataRc.Close()

	reader := bufio.NewReader(dataRc)
	lineBuffer := new(bytes.Buffer)

	var startLine, endLine int
	startLine, _ = strconv.Atoi(strings.TrimPrefix(lineSpecs[0], "L"))

	if len(lineSpecs) == 1 {
		endLine = startLine
	} else {
		endLine, _ = strconv.Atoi(strings.TrimPrefix(lineSpecs[1], "L"))
	}

	// simple inspection
	if startLine < 1 || endLine < 1 || startLine > endLine {
		return nil
	}
	
	rawFileLine := 0
	linesRead := 0
	for {
		buf, err := reader.ReadBytes('\n')
		
		if err != nil && !errors.Is(err, io.EOF) {
			return nil
		}

		rawFileLine++

		if rawFileLine >= startLine && rawFileLine <= endLine {
			lineBuffer.Write(buf)
			linesRead++
		}
		
		if errors.Is(err, io.EOF) {
			break
		}
		
		// Row limit check
		if setting.FilePreviewMaxLines > 0 && linesRead >= setting.FilePreviewMaxLines {
			if rawFileLine < endLine {
				preview.isTruncated = true
			}
			break
		}

		if rawFileLine >= endLine {
			break
		}
	}

	// After the cycle is completed, the boundary conditions are uniformly processed
	if endLine > rawFileLine {
		endLine = rawFileLine
	}
	if startLine > rawFileLine {
		startLine = 1
		endLine = rawFileLine
	}

	if startLine == endLine {
		preview.subTitle = locale.Tr(
			"markup.filepreview.line", startLine,
			template.HTML(commitLinkBuffer.String()),
		)
	} else {
		preview.subTitle = locale.Tr(
			"markup.filepreview.lines", startLine, endLine,
			template.HTML(commitLinkBuffer.String()),
		)
	}
	
	preview.lineOffset = startLine - 1	

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