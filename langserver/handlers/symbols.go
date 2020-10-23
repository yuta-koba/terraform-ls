package handlers

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl-lang/lang"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/sourcegraph/go-lsp"
)

func (h *logHandler) TextDocumentSymbol(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.SymbolInformation, error) {
	var symbols []lsp.SymbolInformation

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return symbols, err
	}

	df, err := lsctx.DecoderFinder(ctx)
	if err != nil {
		return symbols, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return symbols, err
	}

	// TODO: block until it's available <-df.ParserLoadingDone()
	// requires https://github.com/hashicorp/terraform-ls/issues/8
	// textDocument/documentSymbol fires early alongside textDocument/didOpen
	// the protocol does not retry the request, so it's best to give the parser
	// a moment
	if err := Waiter(func() (bool, error) {
		return df.IsCoreSchemaLoaded(file.Dir())
	}).Waitf("core schema is not available yet for %s", file.Dir()); err != nil {
		return symbols, err
	}

	d, err := df.DecoderForDir(file.Dir())
	if err != nil {
		return symbols, fmt.Errorf("finding compatible decoder failed: %w", err)
	}

	sbs, err := d.Symbols()
	if err != nil {
		return symbols, err
	}
	for _, s := range sbs {
		var kind lsp.SymbolKind
		switch s.Kind() {
		case lang.BlockSymbolKind:
			kind = lsp.SKClass
		case lang.AttributeSymbolKind:
			kind = lsp.SKField
		}

		symbols = append(symbols, lsp.SymbolInformation{
			Name: s.Name(),
			Kind: kind,
			Location: lsp.Location{
				Range: ilsp.HCLRangeToLSP(s.Range()),
				URI:   params.TextDocument.URI,
			},
			// TODO: children
		})
	}

	return symbols, nil

}
