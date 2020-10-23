package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/mdplain"
	lsp "github.com/sourcegraph/go-lsp"
)

func CompletionList(candidates lang.Candidates, caps lsp.TextDocumentClientCapabilities) lsp.CompletionList {
	snippetSupport := caps.Completion.CompletionItem.SnippetSupport
	list := lsp.CompletionList{}

	if candidates == nil {
		return list
	}

	list.IsIncomplete = !candidates.IsComplete()
	list.Items = make([]lsp.CompletionItem, len(candidates))
	for i, c := range candidates {
		list.Items[i] = CompletionItem(c, snippetSupport)
	}

	return list
}

func CompletionItem(candidate lang.Candidate, snippetSupport bool) lsp.CompletionItem {
	doc := candidate.Description.Value
	if candidate.Description.Kind == lang.MarkdownKind {
		// TODO: markdown handling
		doc = mdplain.Clean(doc)
	}

	var kind lsp.CompletionItemKind
	switch candidate.Kind {
	case lang.AttributeCandidateKind:
		kind = lsp.CIKField
	case lang.BlockCandidateKind:
		kind = lsp.CIKClass
	case lang.LabelCandidateKind:
		kind = lsp.CIKConstant
	}

	te, format := textEdit(candidate.TextEdit, snippetSupport)

	return lsp.CompletionItem{
		Label:            candidate.Label,
		Kind:             kind,
		InsertTextFormat: format,
		Detail:           candidate.Detail,
		Documentation:    doc,
		TextEdit:         te,
	}
}

func textEdit(te lang.TextEdit, snippetSupport bool) (*lsp.TextEdit, lsp.InsertTextFormat) {
	if snippetSupport {
		return &lsp.TextEdit{
			NewText: te.Snippet,
			Range:   HCLRangeToLSP(te.Range),
		}, lsp.ITFSnippet
	}

	return &lsp.TextEdit{
			NewText: te.RawText,
			Range:   HCLRangeToLSP(te.Range),
		}, lsp.ITFPlainText
}
