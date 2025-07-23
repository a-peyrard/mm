package code

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	golang "github.com/tree-sitter/tree-sitter-go/bindings/go"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type ChunkMetadata struct {
	FilePath     string `json:"file_path"`
	FunctionName string `json:"function_name,omitempty"`
	ClassName    string `json:"class_name,omitempty"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	Language     string `json:"language"`
	ChunkType    string `json:"chunk_type"` // "function", "class", "variable", "import", etc.
}

type Chunk struct {
	Id       string        `json:"id"`
	Content  string        `json:"content"`
	Metadata ChunkMetadata `json:"metadata"`
}

type LanguageConfig struct {
	Language     *sitter.Language
	Queries      map[string]string
	FileExt      string
	LanguageName string
}

// GenericParser handles parsing of multiple languages
type GenericParser struct {
	languages map[string]LanguageConfig
}

// NewGenericParser creates a new parser with language configurations
func NewGenericParser() *GenericParser {
	parser := &GenericParser{
		languages: make(map[string]LanguageConfig),
	}

	// Configure supported languages
	parser.configureLanguages()
	return parser
}

func (p *GenericParser) configureLanguages() {
	// Python configuration
	p.languages["python"] = LanguageConfig{
		Language:     sitter.NewLanguage(python.Language()),
		FileExt:      ".py",
		LanguageName: "python",
		Queries: map[string]string{
			"functions": `
				(function_definition
					name: (identifier) @function.name
					parameters: (parameters) @function.params
					body: (block) @function.body
				) @function.definition
			`,
			"classes": `
				(class_definition
					name: (identifier) @class.name
					body: (block) @class.body
				) @class.definition
			`,
			"variables": `
				(assignment
					left: (identifier) @variable.name
					right: (_) @variable.value
				) @variable.assignment
			`,
			"imports": `
				(import_statement) @import
				(import_from_statement) @import
			`,
		},
	}

	// Go configuration
	p.languages["go"] = LanguageConfig{
		Language:     sitter.NewLanguage(golang.Language()),
		FileExt:      ".go",
		LanguageName: "go",
		Queries: map[string]string{
			"functions": `
				(function_declaration
					name: (identifier) @function.name
					parameters: (parameter_list) @function.params
					body: (block) @function.body
				) @function.definition
				(method_declaration
					name: (identifier) @method.name
					parameters: (parameter_list) @method.params
					body: (block) @method.body
				) @method.definition
			`,
			"types": `
				(type_declaration
					(type_spec
						name: (type_identifier) @type.name
						type: (_) @type.definition
					)
				) @type.declaration
			`,
			"variables": `
				(var_declaration
					(var_spec
						name: (identifier) @variable.name
						type: (_)? @variable.type
						value: (_)? @variable.value
					)
				) @variable.declaration
			`,
			"constants": `
				(const_declaration
					(const_spec
						name: (identifier) @constant.name
						type: (_)? @constant.type
						value: (_) @constant.value
					)
				) @constant.declaration
			`,
		},
	}

	// JavaScript configuration
	p.languages["javascript"] = LanguageConfig{
		Language:     sitter.NewLanguage(javascript.Language()),
		FileExt:      ".js",
		LanguageName: "javascript",
		Queries: map[string]string{
			"functions": `
				(function_declaration
					name: (identifier) @function.name
					parameters: (formal_parameters) @function.params
					body: (statement_block) @function.body
				) @function.definition
				(arrow_function
					parameters: (formal_parameters) @function.params
					body: (_) @function.body
				) @function.definition
			`,
			"classes": `
				(class_declaration
					name: (identifier) @class.name
					body: (class_body) @class.body
				) @class.definition
			`,
			"variables": `
				(variable_declaration
					(variable_declarator
						name: (identifier) @variable.name
						value: (_)? @variable.value
					)
				) @variable.declaration
			`,
		},
	}

	// TypeScript configuration
	p.languages["typescript"] = LanguageConfig{
		Language:     sitter.NewLanguage(typescript.LanguageTypescript()),
		FileExt:      ".ts",
		LanguageName: "typescript",
		Queries: map[string]string{
			"functions": `
				(function_declaration
					name: (identifier) @function.name
					parameters: (formal_parameters) @function.params
					body: (statement_block) @function.body
				) @function.definition
				(arrow_function
					parameters: (formal_parameters) @function.params
					body: (_) @function.body
				) @function.definition
			`,
			"classes": `
				(class_declaration
					name: (identifier) @class.name
					body: (class_body) @class.body
				) @class.definition
			`,
			"interfaces": `
				(interface_declaration
					name: (type_identifier) @interface.name
					body: (object_type) @interface.body
				) @interface.definition
			`,
			"types": `
				(type_alias_declaration
					name: (type_identifier) @type.name
					value: (_) @type.definition
				) @type.declaration
			`,
			"variables": `
				(variable_declaration
					(variable_declarator
						name: (identifier) @variable.name
						value: (_)? @variable.value
					)
				) @variable.declaration
			`,
		},
	}

	// Rust configuration
	p.languages["rust"] = LanguageConfig{
		Language:     sitter.NewLanguage(rust.Language()),
		FileExt:      ".rs",
		LanguageName: "rust",
		Queries: map[string]string{
			"functions": `
				(function_item
					name: (identifier) @function.name
					parameters: (parameters) @function.params
					body: (block) @function.body
				) @function.definition
			`,
			"structs": `
				(struct_item
					name: (type_identifier) @struct.name
					body: (field_declaration_list) @struct.body
				) @struct.definition
			`,
			"enums": `
				(enum_item
					name: (type_identifier) @enum.name
					body: (enum_variant_list) @enum.body
				) @enum.definition
			`,
			"impls": `
				(impl_item
					type: (type_identifier) @impl.type
					body: (declaration_list) @impl.body
				) @impl.definition
			`,
			"traits": `
				(trait_item
					name: (type_identifier) @trait.name
					body: (declaration_list) @trait.body
				) @trait.definition
			`,
			"constants": `
				(const_item
					name: (identifier) @constant.name
					type: (_) @constant.type
					value: (_) @constant.value
				) @constant.definition
			`,
			"statics": `
				(static_item
					name: (identifier) @static.name
					type: (_) @static.type
					value: (_) @static.value
				) @static.definition
			`,
		},
	}

	// Also add TypeScript JSX support
	p.languages["tsx"] = LanguageConfig{
		Language:     sitter.NewLanguage(typescript.LanguageTSX()),
		FileExt:      ".tsx",
		LanguageName: "typescript",
		Queries:      p.languages["typescript"].Queries, // Reuse TypeScript queries
	}
}

// ParseFile parses a source file and returns chunks
func (p *GenericParser) ParseFile(filePath string, sourceCode []byte) ([]Chunk, error) {
	config, found := p.detectLanguage(filePath)
	if !found {
		return nil, fmt.Errorf("unsupported file type: %s", filePath)
	}

	parser := sitter.NewParser()
	err := parser.SetLanguage(config.Language)
	if err != nil {
		return nil, err
	}

	var tree *sitter.Tree
	defer func() {
		if r := recover(); r != nil {
			// if Parse method panics, we'll handle it gracefully
			tree = nil
		}
	}()

	callback := func(offset int, position sitter.Point) []byte {
		if offset >= len(sourceCode) {
			return []byte{}
		}
		return sourceCode[offset:]
	}
	tree = parser.ParseWithOptions(callback, nil, nil) // Pass nil for options
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file: %s", filePath)
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	if rootNode == nil {
		return nil, fmt.Errorf("failed to get root node for file: %s", filePath)
	}

	chunks := make([]Chunk, 0)

	// Extract different types of definitions
	for queryType, queryString := range config.Queries {
		typeChunks, err := p.extractChunksWithQuery(
			rootNode,
			queryString,
			sourceCode,
			filePath,
			config,
			queryType,
		)
		if err != nil {
			continue // Skip failed queries
		}
		chunks = append(chunks, typeChunks...)
	}

	return chunks, nil
}

func (p *GenericParser) extractChunksWithQuery(
	node *sitter.Node,
	queryString string,
	sourceCode []byte,
	filePath string,
	config *LanguageConfig,
	chunkType string,
) ([]Chunk, error) {
	query, err := sitter.NewQuery(config.Language, queryString)
	if err != nil {
		return nil, err
	}
	defer query.Close()

	// Execute query
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	// Execute the query on the node
	matches := cursor.Matches(query, node, sourceCode)

	var chunks []Chunk

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		chunk := p.processMatch(match, sourceCode, filePath, config.LanguageName, chunkType)
		if chunk != nil {
			chunks = append(chunks, *chunk)
		}
	}

	return chunks, nil
}

func (p *GenericParser) processMatch(
	match *sitter.QueryMatch,
	sourceCode []byte,
	filePath string,
	language string,
	chunkType string,
) *Chunk {
	var mainNode *sitter.Node
	var name string
	var className string

	// Extract information from captures
	for _, capture := range match.Captures {
		content := capture.Node.Utf8Text(sourceCode)

		switch {
		case strings.Contains(capture.Node.Kind(), "definition"):
			mainNode = &capture.Node
		case capture.Node.Kind() == "assignment":
			mainNode = &capture.Node
		case capture.Node.Kind() == "identifier":
			name = content
		case strings.Contains(capture.Node.Kind(), "class"):
			if strings.Contains(capture.Node.Kind(), "name") {
				className = content
			}
		}
	}

	if mainNode == nil {
		return nil
	}

	// Get the content of the matched node
	content := mainNode.Utf8Text(sourceCode)

	// Calculate line numbers
	startLine := int(mainNode.StartPosition().Row) + 1
	endLine := int(mainNode.EndPosition().Row) + 1

	// Generate unique ID
	id := fmt.Sprintf("%s_%s_%d", filePath, name, startLine)
	if name == "" {
		id = fmt.Sprintf("%s_%s_%d", filePath, chunkType, startLine)
	}

	if chunkType == "functions" && isMethod(mainNode, sourceCode) {
		className = extractParentIdentifier(mainNode, sourceCode)
		chunkType = "methods"
	}
	if chunkType == "classes" {
		className = name
		name = ""
	}

	// Create chunk
	chunk := &Chunk{
		Id:      id,
		Content: content,
		Metadata: ChunkMetadata{
			FilePath:     filePath,
			FunctionName: name,
			ClassName:    className,
			StartLine:    startLine,
			EndLine:      endLine,
			Language:     language,
			ChunkType:    chunkType,
		},
	}

	return chunk
}

//func extractParentIdentifier(node *sitter.Node, sourceCode []byte) string {
//	for parent := node.Parent(); parent != nil; parent = parent.NextSibling() {
//		if parent.Kind() == "identifier" {
//			return parent.Utf8Text(sourceCode)
//		}
//	}
//	for parent := node.Parent(); parent != nil; parent = parent.PrevSibling() {
//		if parent.Kind() == "identifier" {
//			return parent.Utf8Text(sourceCode)
//		}
//	}
//
//	return ""
//}

func (p *GenericParser) detectLanguage(filePath string) (config *LanguageConfig, found bool) {
	for _, config := range p.languages {
		if strings.HasSuffix(filePath, config.FileExt) {
			return &config, true
		}
	}
	return nil, false
}

func extractParentIdentifier(node *sitter.Node, sourceCode []byte) string {
	// Traverse up the AST to find a class definition
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Kind() == "class_definition" {
			// Found a class definition, now find its name
			for i := 0; i < int(parent.ChildCount()); i++ {
				child := parent.Child(uint(i))
				if child.Kind() == "identifier" {
					return child.Utf8Text(sourceCode)
				}
			}
		}
	}
	return ""
}

func isMethod(node *sitter.Node, sourceCode []byte) bool {
	// Check if this function is inside a class definition
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Kind() == "class_definition" {
			return true
		}
	}
	return false
}
