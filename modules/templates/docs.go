// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package templates

import (
	"html/template"
	"forgejo.org/modules/svg"
	"forgejo.org/modules/translation"
	"strings"
)

func RenderDocs(ctx *Context, key string) template.HTML {
	snippets, _ := ctx.Data["DocsSnippets"].([]string)
	ctx.Data["DocsSnippets"] = append(snippets, key)

	return template.HTML("<a class='help-icon' id='" + closeTarget(key) + "' href='#" + modalTarget(key) + "' title='" + ctx.Locale.TrString("docs.learn_more") + "'>" + string(svg.RenderHTML("octicon-info")) + "</a>")
}

func RenderDocsModals(ctx *Context) template.HTML {
	snippets, _ := ctx.Data["DocsSnippets"].([]string)

	var buf strings.Builder
	for _, snip := range snippets {
		buf.WriteString(docsModal(ctx.Locale, snip))
	}
	return template.HTML(buf.String())
}

func docsModal(locale translation.Locale, key string) string {
	return modal(key, locale.TrString("docs."+key+".title"), locale.TrString("docs."+key+".content"), locale)
}

func closeTarget(id string) string {
	return "return-docs-" + id
}

func modalTarget(id string) string {
	return "docs-" + id
}

func modal(id, title, content string, locale translation.Locale) string {
	return "<div class='ui dimmer'><a href='#_' tabindex='-1' aria-hidden></a><dialog id='" + modalTarget(id) + "'><article><header>" + title + "</header><div class='content'>" + content + "</div><a href='#" + closeTarget(id) +  "' class='tw-sr-only'>" + locale.TrString("docs.close") + "</a></article></dialog></div>"
}
