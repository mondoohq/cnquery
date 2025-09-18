// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/lr"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"

	// Must include a backend implementation for commonlog
	_ "github.com/tliron/commonlog/simple"
)

const lsName = "lr-language-server"

var lsVersion = "0.1.0"

// Document represents a cached document in the language server
type Document struct {
	URI     protocol.DocumentUri
	Content string
	Version protocol.Integer
	AST     *lr.LR
	Errors  []protocol.Diagnostic
	Schema  *resources.Schema // Cached schema to avoid repeated processing
}

// LRHandler implements the LSP protocol handlers for LR files
type LRHandler struct {
	protocol.Handler
	documents map[protocol.DocumentUri]*Document
	mutex     sync.RWMutex
}

// NewLRHandler creates a new LR language server handler
func NewLRHandler() *LRHandler {
	handler := &LRHandler{
		documents: make(map[protocol.DocumentUri]*Document),
	}

	// Set up the protocol handlers
	handler.Handler = protocol.Handler{
		Initialize:                 handler.initialize,
		Initialized:                handler.initialized,
		Shutdown:                   handler.shutdown,
		SetTrace:                   handler.setTrace,
		TextDocumentDidOpen:        handler.textDocumentDidOpen,
		TextDocumentDidChange:      handler.textDocumentDidChange,
		TextDocumentDidClose:       handler.textDocumentDidClose,
		TextDocumentDocumentSymbol: handler.textDocumentDocumentSymbol,
		TextDocumentHover:          handler.textDocumentHover,
		TextDocumentDefinition:     handler.textDocumentDefinition,
		TextDocumentReferences:     handler.textDocumentReferences,
	}

	return handler
}

func init() {
	rootCmd.AddCommand(lspCmd)
}

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the LR language server",
	Long:  `Start the Language Server Protocol (LSP) server for LR files`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, _ := cmd.Flags().GetString("mode")
		debug, _ := cmd.Flags().GetBool("debug")

		switch mode {
		case "test":
			return runTestMode()
		case "server":
			return runLSPServer(debug)
		default:
			return runTestMode()
		}
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
	lspCmd.Flags().StringP("mode", "m", "test", "Mode to run: 'test' for demo, 'server' for LSP server")
	lspCmd.Flags().BoolP("debug", "d", false, "Enable debug logging")
} // runTestMode demonstrates the parsing capabilities
func runTestMode() error {
	fmt.Println("LR Language Server - Test Mode")
	fmt.Println("===============================")

	handler := NewLRHandler()

	// Test with a simple LR content
	testContent := `
// Copyright (c) Example Corp
// SPDX-License-Identifier: MIT

import "../../core/resources/core.lr"

option provider = "example.com/provider"
option go_package = "example.com/provider/resources"

// User represents a system user account
user @defaults("name uid") {
  init(username string, createHome? bool)
  // Username for this user
  name string
  // User ID number
  uid int
  // Email address
  email string
  // User's primary group
  group() group
  // All groups this user belongs to
  groups() []group
}

// Group represents a system group
private group @context("system.context") {
  // Group name
  name string
  // Group ID number  
  gid int
  // Members of this group
  members() []user
}

// Extended user information
extend user {
  // Last login time
  lastLogin() time
  // Home directory
  homeDir string
}

// Custom list type for user management
userList {
  []user
  
  // Find active users
  active() []user
  // Count of users
  count() int
}`

	doc := handler.processDocument("file:///test.lr", testContent, 1)
	if len(doc.Errors) > 0 {
		fmt.Println("Parse errors:", doc.Errors)
		return fmt.Errorf("parse errors occurred")
	}

	fmt.Printf("✓ Successfully parsed %d resources\n", len(doc.AST.Resources))
	for _, resource := range doc.AST.Resources {
		fmt.Printf("  📁 Resource: %s", resource.ID)
		if resource.IsPrivate {
			fmt.Print(" (private)")
		}
		if resource.IsExtension {
			fmt.Print(" (extension)")
		}
		fmt.Println()

		if resource.Body != nil {
			for _, field := range resource.Body.Fields {
				if field.BasicField != nil && field.BasicField.ID != "" {
					typeStr := getTypeString(field.BasicField.Type)
					fmt.Printf("    🔸 Field: %s %s\n", field.BasicField.ID, typeStr)
				}
			}
		}
	}

	// Test symbol extraction
	symbols := handler.extractSymbols(doc.AST)
	fmt.Printf("\n✓ Extracted %d symbols for LSP\n", len(symbols))

	// Test hover simulation
	hoverResult := handler.findSymbolAtPosition(doc.AST, 0, 0)
	if hoverResult != "" {
		fmt.Printf("✓ Hover at (0,0): %s\n", hoverResult)
	}

	fmt.Println("\n🚀 Ready for LSP integration!")
	fmt.Println("\nTo enable full LSP server mode:")
	fmt.Println("\tRun: lr lsp --mode=server")

	return nil
}

// runLSPServer starts the actual LSP server
func runLSPServer(debug bool) error {
	// Set up logging
	if debug {
		log.Debug().Msg("Debug logging enabled")
	}

	// Create the LSP server
	handler := NewLRHandler()

	// Create the server using GLSP
	server := server.NewServer(handler, lsName, debug)

	if debug {
		log.Debug().Msg("Starting LSP server with debug mode")
	}

	log.Info().Msg("LR LSP server starting - reading from stdin, writing to stdout")

	return server.RunStdio()
}

// processDocument parses LR content and caches the result
func (h *LRHandler) processDocument(uri protocol.DocumentUri, content string, version protocol.Integer) *Document {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	doc := &Document{
		URI:     uri,
		Content: content,
		Version: version,
		Errors:  []protocol.Diagnostic{},
	}

	// Parse the LR content
	ast, err := lr.Parse(content)
	if err != nil {
		// Convert parse error to LSP diagnostic
		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
			Message:  err.Error(),
			Source:   &[]string{"lr-parser"}[0],
		}
		doc.Errors = append(doc.Errors, diagnostic)
	} else {
		doc.AST = ast
	}

	h.documents[uri] = doc
	return doc
}

// getDocument retrieves a cached document
func (h *LRHandler) getDocument(uri protocol.DocumentUri) *Document {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.documents[uri]
}

// extractSymbols extracts document symbols from the AST
func (h *LRHandler) extractSymbols(ast *lr.LR) []string {
	var symbols []string

	for _, resource := range ast.Resources {
		symbols = append(symbols, fmt.Sprintf("Resource: %s", resource.ID))

		if resource.Body != nil {
			for _, field := range resource.Body.Fields {
				if field.BasicField != nil && field.BasicField.ID != "" {
					symbols = append(symbols, fmt.Sprintf("  Field: %s", field.BasicField.ID))
				}
			}
		}
	}

	return symbols
}

// findSymbolAtPosition finds what symbol is at a given line/character position
func (h *LRHandler) findSymbolAtPosition(ast *lr.LR, line, character int) string {
	// This is a simplified implementation
	// In a real LSP, you'd need to track source positions in the AST

	if len(ast.Resources) > line {
		resource := ast.Resources[line]
		return fmt.Sprintf("Resource: %s", resource.ID)
	}

	return ""
}

// getTypeString converts a Type to a readable string representation
func getTypeString(t lr.Type) string {
	if t.SimpleType != nil {
		return t.SimpleType.Type
	}
	if t.ListType != nil {
		return "[]" + getTypeString(t.ListType.Type)
	}
	if t.MapType != nil {
		return fmt.Sprintf("map[%s]%s", t.MapType.Key.Type, getTypeString(t.MapType.Value))
	}
	return "unknown"
}

// LSP Protocol Handlers

// initialize handles the initialize request
func (h *LRHandler) initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := protocol.ServerCapabilities{
		TextDocumentSync: &protocol.TextDocumentSyncOptions{
			OpenClose: &[]bool{true}[0],
			Change:    &[]protocol.TextDocumentSyncKind{protocol.TextDocumentSyncKindIncremental}[0],
		},
		HoverProvider:          &[]bool{true}[0],
		DocumentSymbolProvider: &[]bool{true}[0],
		DefinitionProvider:     &[]bool{true}[0],
		ReferencesProvider:     &[]bool{true}[0],
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &lsVersion,
		},
	}, nil
}

// initialized handles the initialized notification
func (h *LRHandler) initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

// shutdown handles the shutdown request
func (h *LRHandler) shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

// setTrace handles the setTrace notification
func (h *LRHandler) setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

// textDocumentDidOpen handles when a document is opened
func (h *LRHandler) textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	doc := h.processDocument(params.TextDocument.URI, params.TextDocument.Text, params.TextDocument.Version)

	// Publish diagnostics
	context.Notify("textDocument/publishDiagnostics", protocol.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: doc.Errors,
	})

	return nil
}

// textDocumentDidChange handles when a document is changed
func (h *LRHandler) textDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	// For simplicity, we'll do full document sync (take the last change which should be the full content)
	if len(params.ContentChanges) > 0 {
		change := params.ContentChanges[len(params.ContentChanges)-1]
		if textChange, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
			doc := h.processDocument(params.TextDocument.URI, textChange.Text, params.TextDocument.Version)

			// Publish diagnostics
			context.Notify("textDocument/publishDiagnostics", protocol.PublishDiagnosticsParams{
				URI:         params.TextDocument.URI,
				Diagnostics: doc.Errors,
			})
		}
	}
	return nil
}

// textDocumentDidClose handles when a document is closed
func (h *LRHandler) textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.documents, params.TextDocument.URI)
	return nil
}

// textDocumentDocumentSymbol handles document symbol requests
func (h *LRHandler) textDocumentDocumentSymbol(context *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	doc := h.getDocument(params.TextDocument.URI)
	if doc == nil || doc.AST == nil {
		return []protocol.DocumentSymbol{}, nil
	}

	var symbols []protocol.DocumentSymbol

	for i, resource := range doc.AST.Resources {
		// Create resource symbol
		resourceSymbol := protocol.DocumentSymbol{
			Name: resource.ID,
			Kind: protocol.SymbolKindClass,
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(i * 10), Character: 0}, // Simplified positioning
				End:   protocol.Position{Line: protocol.UInteger(i*10 + 5), Character: 0},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(i * 10), Character: 0},
				End:   protocol.Position{Line: protocol.UInteger(i * 10), Character: protocol.UInteger(len(resource.ID))},
			},
		}

		// Add detail for resource modifiers
		if resource.IsPrivate {
			resourceSymbol.Detail = stringPtr("private")
		}
		if resource.IsExtension {
			if resourceSymbol.Detail != nil {
				resourceSymbol.Detail = stringPtr(*resourceSymbol.Detail + " extend")
			} else {
				resourceSymbol.Detail = stringPtr("extend")
			}
		}

		// Add field symbols as children
		if resource.Body != nil {
			for j, field := range resource.Body.Fields {
				if field.BasicField != nil && field.BasicField.ID != "" {
					// Determine symbol kind based on whether it's a method or field
					var symbolKind protocol.SymbolKind
					var detail string

					if field.BasicField.Args != nil {
						// Has arguments = method
						symbolKind = protocol.SymbolKindMethod
						if len(field.BasicField.Args.List) > 0 {
							detail = fmt.Sprintf("method with %d args", len(field.BasicField.Args.List))
						} else {
							detail = "method"
						}
					} else {
						// No arguments = field
						symbolKind = protocol.SymbolKindField
						detail = "field"
					}

					fieldSymbol := protocol.DocumentSymbol{
						Name:   field.BasicField.ID,
						Kind:   symbolKind,
						Detail: &detail,
						Range: protocol.Range{
							Start: protocol.Position{Line: protocol.UInteger(i*10 + j + 1), Character: 2},
							End:   protocol.Position{Line: protocol.UInteger(i*10 + j + 1), Character: protocol.UInteger(len(field.BasicField.ID) + 2)},
						},
						SelectionRange: protocol.Range{
							Start: protocol.Position{Line: protocol.UInteger(i*10 + j + 1), Character: 2},
							End:   protocol.Position{Line: protocol.UInteger(i*10 + j + 1), Character: protocol.UInteger(len(field.BasicField.ID) + 2)},
						},
					}
					resourceSymbol.Children = append(resourceSymbol.Children, fieldSymbol)
				}
			}
		}

		symbols = append(symbols, resourceSymbol)
	}

	return symbols, nil
}

// textDocumentHover handles hover requests
func (h *LRHandler) textDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc := h.getDocument(params.TextDocument.URI)
	if doc == nil || doc.AST == nil {
		return nil, nil
	}

	line := int(params.Position.Line)
	character := int(params.Position.Character)

	// Get the word at the cursor position
	lines := strings.Split(doc.Content, "\n")
	if line >= len(lines) {
		return nil, nil
	}

	currentLine := lines[line]
	if character >= len(currentLine) {
		return nil, nil
	}

	// Extract word at cursor position
	word := extractWordAtPosition(currentLine, character)
	if word == "" {
		return nil, nil
	}

	// First, try to find if this word is a resource name
	for _, resource := range doc.AST.Resources {
		if resource.ID == word {
			return h.createResourceHover(resource, doc.AST), nil
		}
	}

	// Then, try to find if this word is a field in any resource
	for _, resource := range doc.AST.Resources {
		if resource.Body != nil {
			for _, field := range resource.Body.Fields {
				if field.BasicField != nil && field.BasicField.ID == word {
					return h.createFieldHover(field.BasicField, resource, doc.AST), nil
				}
			}
		}
	}

	// If we can't find a specific match, try to find which resource context we're in
	// by looking at the line position
	for _, resource := range doc.AST.Resources {
		if isLineInResourceContext(doc.Content, line, resource.ID) {
			return h.createResourceHover(resource, doc.AST), nil
		}
	}

	return nil, nil
}

// extractWordAtPosition extracts the word at the given character position
func extractWordAtPosition(line string, character int) string {
	if character >= len(line) {
		return ""
	}

	// Check if we're inside quotes first
	if character < len(line) && line[character] == '"' {
		// If we're at the start of a quote, move one character in
		if character+1 < len(line) {
			character++
		}
	}

	// Find word boundaries
	start := character
	end := character

	// Handle quoted strings
	if start < len(line) && line[start] == '"' {
		start++
	}

	// Move start backwards to beginning of word
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Move end forwards to end of word
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	if start == end {
		return ""
	}

	word := line[start:end]

	// Remove surrounding quotes if present
	if len(word) > 2 && word[0] == '"' && word[len(word)-1] == '"' {
		word = word[1 : len(word)-1]
	}

	return word
}

// isWordChar checks if a character is part of a word (alphanumeric, underscore, or dot)
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.'
}

// isLineInResourceContext checks if a line is within a resource definition
func isLineInResourceContext(content string, targetLine int, resourceID string) bool {
	lines := strings.Split(content, "\n")

	// Find the resource declaration line
	resourceLineStart := -1
	resourceLineEnd := -1

	for i, line := range lines {
		if strings.Contains(line, resourceID+" ") || strings.Contains(line, resourceID+` "`) {
			resourceLineStart = i

			// Find the closing brace
			braceCount := 0
			for j := i; j < len(lines); j++ {
				lineContent := lines[j]
				for _, char := range lineContent {
					if char == '{' {
						braceCount++
					} else if char == '}' {
						braceCount--
						if braceCount == 0 {
							resourceLineEnd = j
							break
						}
					}
				}
				if resourceLineEnd != -1 {
					break
				}
			}
			break
		}
	}

	return resourceLineStart != -1 && resourceLineEnd != -1 &&
		targetLine >= resourceLineStart && targetLine <= resourceLineEnd
}

// textDocumentDefinition handles go to definition requests
func (h *LRHandler) textDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	doc := h.getDocument(params.TextDocument.URI)
	if doc == nil || doc.AST == nil {
		return nil, nil
	}

	line := int(params.Position.Line)
	character := int(params.Position.Character)

	// Get the word at the cursor position
	lines := strings.Split(doc.Content, "\n")
	if line >= len(lines) {
		return nil, nil
	}

	currentLine := lines[line]
	if character >= len(currentLine) {
		return nil, nil
	}

	// Extract word at cursor position
	word := extractWordAtPosition(currentLine, character)
	if word == "" {
		return nil, nil
	}

	// Log for debugging
	log.Debug().Str("word", word).Int("line", line).Int("char", character).Str("currentLine", currentLine).Msg("Looking for definition")

	// Find definition locations
	locations := h.findDefinitionLocations(doc, word)

	log.Debug().Int("locations", len(locations)).Msg("Found definition locations")

	if len(locations) == 0 {
		return nil, nil
	}
	return locations, nil
} // textDocumentReferences handles find references requests
func (h *LRHandler) textDocumentReferences(context *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	doc := h.getDocument(params.TextDocument.URI)
	if doc == nil || doc.AST == nil {
		return nil, nil
	}

	line := int(params.Position.Line)
	character := int(params.Position.Character)

	// Get the word at the cursor position
	lines := strings.Split(doc.Content, "\n")
	if line >= len(lines) {
		return nil, nil
	}

	currentLine := lines[line]
	if character >= len(currentLine) {
		return nil, nil
	}

	// Extract word at cursor position
	word := extractWordAtPosition(currentLine, character)
	if word == "" {
		return nil, nil
	}

	// Find all references
	locations := h.findReferenceLocations(doc, word, params.Context.IncludeDeclaration)
	return locations, nil
}

// findDefinitionLocations finds where a symbol is defined
func (h *LRHandler) findDefinitionLocations(doc *Document, word string) []protocol.Location {
	var locations []protocol.Location
	lines := strings.Split(doc.Content, "\n")

	log.Debug().Str("word", word).Msg("Finding definition for word")

	// Search for resource definitions
	for _, resource := range doc.AST.Resources {
		if resource.ID == word {
			log.Debug().Str("resource", resource.ID).Msg("Found matching resource in AST")

			// Look for resource declaration patterns
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)

				// Match patterns like: "user {", "private user {", "extend user {"
				// Also handle @defaults attributes between resource name and brace
				resourcePattern := resource.ID
				if strings.Contains(trimmed, resourcePattern) {
					// Check if this line contains the resource definition
					isDefinition := false

					// Pattern 1: resource.name {
					if strings.Contains(trimmed, resourcePattern+" {") {
						isDefinition = true
					}
					// Pattern 2: resource.name @defaults(...) {
					if strings.Contains(trimmed, resourcePattern+" @") && strings.Contains(trimmed, "{") {
						isDefinition = true
					}
					// Pattern 3: private resource.name {
					if strings.Contains(trimmed, "private "+resourcePattern+" {") {
						isDefinition = true
					}
					// Pattern 4: private resource.name @defaults(...) {
					if strings.Contains(trimmed, "private "+resourcePattern+" @") && strings.Contains(trimmed, "{") {
						isDefinition = true
					}
					// Pattern 5: extend resource.name {
					if strings.Contains(trimmed, "extend "+resourcePattern+" {") {
						isDefinition = true
					}
					// Pattern 6: extend resource.name @defaults(...) {
					if strings.Contains(trimmed, "extend "+resourcePattern+" @") && strings.Contains(trimmed, "{") {
						isDefinition = true
					}

					if isDefinition {
						start := strings.Index(line, resource.ID)
						if start != -1 {
							location := protocol.Location{
								URI: doc.URI,
								Range: protocol.Range{
									Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(start)},
									End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(start + len(resource.ID))},
								},
							}
							locations = append(locations, location)
							log.Debug().Int("line", i).Int("start", start).Str("pattern", "resource definition with attributes").Msg("Found resource definition")
							goto nextResource
						}
					}
				}
			}
		nextResource:
		}
	}

	// Search for field/method definitions
	for _, resource := range doc.AST.Resources {
		if resource.Body != nil {
			for _, field := range resource.Body.Fields {
				if field.BasicField != nil && field.BasicField.ID == word {
					log.Debug().Str("field", field.BasicField.ID).Str("resource", resource.ID).Msg("Found matching field in AST") // Look for field definitions within the resource
					inResource := false
					resourceStart := -1

					for i, line := range lines {
						trimmed := strings.TrimSpace(line)

						// Check if we're entering a resource
						if strings.Contains(trimmed, resource.ID+" {") ||
							strings.Contains(trimmed, resource.ID+` "`) {
							inResource = true
							resourceStart = i
							continue
						}

						// Check if we're leaving the resource
						if inResource && trimmed == "}" {
							inResource = false
							continue
						}

						// Look for field definition within the resource
						if inResource && i > resourceStart {
							// Match field patterns: "name string", "uid int", "groups() []group"
							fieldPatterns := []string{
								field.BasicField.ID + " ",
								field.BasicField.ID + "(",
								field.BasicField.ID + "\t",
							}

							for _, pattern := range fieldPatterns {
								if strings.Contains(line, pattern) {
									start := strings.Index(line, field.BasicField.ID)
									if start != -1 {
										location := protocol.Location{
											URI: doc.URI,
											Range: protocol.Range{
												Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(start)},
												End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(start + len(field.BasicField.ID))},
											},
										}
										locations = append(locations, location)
										log.Debug().Int("line", i).Int("start", start).Str("pattern", pattern).Msg("Found field definition")
										goto nextField
									}
								}
							}
						}
					}
				nextField:
				}
			}
		}
	}

	return locations
}

// findReferenceLocations finds all references to a symbol
func (h *LRHandler) findReferenceLocations(doc *Document, word string, includeDeclaration bool) []protocol.Location {
	var locations []protocol.Location

	// Include definition if requested
	if includeDeclaration {
		locations = append(locations, h.findDefinitionLocations(doc, word)...)
	}

	// For now, we'll do a simple text search for references
	// In a more sophisticated implementation, you'd parse usage contexts
	lines := strings.Split(doc.Content, "\n")
	for i, line := range lines {
		// Find all occurrences of the word in this line
		start := 0
		for {
			index := strings.Index(line[start:], word)
			if index == -1 {
				break
			}

			index += start

			// Check if this is a whole word (not part of another identifier)
			if (index == 0 || !isWordChar(line[index-1])) &&
				(index+len(word) >= len(line) || !isWordChar(line[index+len(word)])) {

				// Skip if this is the definition and we already included it
				if !includeDeclaration {
					isDefinition := false
					for _, resource := range doc.AST.Resources {
						if resource.ID == word && strings.Contains(line, resource.ID) {
							// Check if this line contains the resource definition
							trimmed := strings.TrimSpace(line)
							resourcePattern := resource.ID

							// Check various definition patterns including @defaults
							if strings.Contains(trimmed, resourcePattern+" {") ||
								(strings.Contains(trimmed, resourcePattern+" @") && strings.Contains(trimmed, "{")) ||
								strings.Contains(trimmed, "private "+resourcePattern+" {") ||
								(strings.Contains(trimmed, "private "+resourcePattern+" @") && strings.Contains(trimmed, "{")) ||
								strings.Contains(trimmed, "extend "+resourcePattern+" {") ||
								(strings.Contains(trimmed, "extend "+resourcePattern+" @") && strings.Contains(trimmed, "{")) {
								isDefinition = true
								break
							}
						}
					}
					if !isDefinition {
						locations = append(locations, protocol.Location{
							URI: doc.URI,
							Range: protocol.Range{
								Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(index)},
								End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(index + len(word))},
							},
						})
					}
				} else {
					locations = append(locations, protocol.Location{
						URI: doc.URI,
						Range: protocol.Range{
							Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(index)},
							End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(index + len(word))},
						},
					})
				}
			}

			start = index + 1
		}
	}

	return locations
}

// createResourceHover creates hover content for a resource
func (h *LRHandler) createResourceHover(resource *lr.Resource, ast *lr.LR) *protocol.Hover {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("**Resource**: `%s`\n\n", resource.ID))

	// Include resource title and description from comments
	if resourceInfo, ok := h.getResourceInfo(resource.ID, ast); ok {
		if resourceInfo.Title != "" {
			content.WriteString(fmt.Sprintf("**%s**\n\n", resourceInfo.Title))
		}
		if resourceInfo.Desc != "" {
			content.WriteString(fmt.Sprintf("%s\n\n", resourceInfo.Desc))
		}
	}

	if resource.IsPrivate {
		content.WriteString("*Private resource*\n\n")
	}
	if resource.IsExtension {
		content.WriteString("*Extension resource*\n\n")
	}

	if resource.Body != nil && len(resource.Body.Fields) > 0 {
		content.WriteString("**Fields:**\n")
		for _, field := range resource.Body.Fields {
			if field.BasicField != nil && field.BasicField.ID != "" {
				typeStr := getTypeString(field.BasicField.Type)
				content.WriteString(fmt.Sprintf("- `%s`: %s\n", field.BasicField.ID, typeStr))
			}
		}
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: content.String(),
		},
	}
}

// createFieldHover creates hover content for a field or method
func (h *LRHandler) createFieldHover(field *lr.BasicField, resource *lr.Resource, ast *lr.LR) *protocol.Hover {
	var content strings.Builder

	// Determine if this is a field (static) or method (dynamic)
	isMethod := field.Args != nil

	if isMethod {
		content.WriteString(fmt.Sprintf("**Method**: `%s`\n\n", field.ID))

		// Show method signature with arguments
		if len(field.Args.List) > 0 {
			content.WriteString("**Arguments**:\n")
			for i, arg := range field.Args.List {
				content.WriteString(fmt.Sprintf("- `%s`: %s\n", fmt.Sprintf("arg%d", i+1), arg.Type))
			}
			content.WriteString("\n")
		} else {
			content.WriteString("**Arguments**: None\n\n")
		}
	} else {
		content.WriteString(fmt.Sprintf("**Field**: `%s`\n\n", field.ID))
	}

	// Include field title and description from comments
	if resourceInfo, ok := h.getResourceInfo(resource.ID, ast); ok {
		if fieldInfo, ok := resourceInfo.Fields[field.ID]; ok {
			if fieldInfo.Title != "" {
				content.WriteString(fmt.Sprintf("**%s**\n\n", fieldInfo.Title))
			}
			if fieldInfo.Desc != "" {
				content.WriteString(fmt.Sprintf("%s\n\n", fieldInfo.Desc))
			}
		}
	}

	typeStr := getTypeString(field.Type)
	content.WriteString(fmt.Sprintf("**Type**: %s\n\n", typeStr))

	content.WriteString(fmt.Sprintf("**Resource**: `%s`\n\n", resource.ID))

	// Add specific details based on type
	if isMethod {
		content.WriteString("**Details**: Computed method in LR resource\n")
	} else {
		content.WriteString("**Details**: Static field in LR resource\n")
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: content.String(),
		},
	}
}

// getResourceInfo safely retrieves resource information with caching to avoid AST corruption
func (h *LRHandler) getResourceInfo(resourceID string, ast *lr.LR) (*resources.ResourceInfo, bool) {
	if ast == nil {
		return nil, false
	}

	// Check if we already have a cached schema for this document
	h.mutex.RLock()
	for _, doc := range h.documents {
		if doc.AST == ast && doc.Schema != nil {
			h.mutex.RUnlock()
			if resourceInfo, ok := doc.Schema.Resources[resourceID]; ok {
				return resourceInfo, true
			}
			return nil, false
		}
	}
	h.mutex.RUnlock()

	// We need to create a schema, but we need to be careful not to modify the original AST
	// Create a minimal copy with just the options we need
	astCopy := &lr.LR{
		Comments:  ast.Comments,
		Imports:   ast.Imports,
		Options:   make(map[string]string),
		Aliases:   ast.Aliases,
		Resources: ast.Resources,
	}

	// Copy original options
	for k, v := range ast.Options {
		astCopy.Options[k] = v
	}

	// Add provider if missing
	if _, ok := astCopy.Options["provider"]; !ok {
		astCopy.Options["provider"] = "unknown"
	}

	schema, err := lr.Schema(astCopy)
	if err != nil {
		return nil, false
	}

	// Cache the schema in the document for future use
	h.mutex.Lock()
	for _, doc := range h.documents {
		if doc.AST == ast {
			doc.Schema = schema
			break
		}
	}
	h.mutex.Unlock()

	resourceInfo, ok := schema.Resources[resourceID]
	return resourceInfo, ok
}

// stringPtr returns a pointer to a string (helper function)
func stringPtr(s string) *string {
	return &s
}
