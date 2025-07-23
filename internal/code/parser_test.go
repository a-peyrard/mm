package code

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
)

func TestGenericParser_ParseFile_Python(t *testing.T) {
	type args struct {
		filePath   string
		sourceCode string
	}
	tests := []struct {
		name    string
		args    args
		want    []Chunk
		wantErr bool
	}{
		{
			name: "it should parse Python function and class correctly",
			args: args{
				filePath: "test.py",
				sourceCode: `
def calculate_tax(income):
   if income > 50000:
       return income * 0.3
   else:
       return income * 0.2

class TaxCalculator:
   def __init__(self):
       self.rate = 0.2
   
   def calculate(self, amount):
       return amount * self.rate

TAX_RATE = 0.2
`,
			},
			want: []Chunk{
				{
					Id:      "test.py_calculate_tax_2",
					Content: "def calculate_tax(income):\n   if income > 50000:\n       return income * 0.3\n   else:\n       return income * 0.2",
					Metadata: ChunkMetadata{
						FilePath:     "test.py",
						FunctionName: "calculate_tax",
						StartLine:    2,
						EndLine:      6,
						Language:     "python",
						ChunkType:    "functions",
					},
				},
				{
					Id:      "test.py___init___9",
					Content: "def __init__(self):\n       self.rate = 0.2",
					Metadata: ChunkMetadata{
						FilePath:     "test.py",
						FunctionName: "__init__",
						ClassName:    "TaxCalculator",
						StartLine:    9,
						EndLine:      10,
						Language:     "python",
						ChunkType:    "methods",
					},
				},
				{
					Id:      "test.py_calculate_12",
					Content: "def calculate(self, amount):\n       return amount * self.rate",
					Metadata: ChunkMetadata{
						FilePath:     "test.py",
						FunctionName: "calculate",
						ClassName:    "TaxCalculator",
						StartLine:    12,
						EndLine:      13,
						Language:     "python",
						ChunkType:    "methods",
					},
				},
				{
					Id:      "test.py_TaxCalculator_8",
					Content: "class TaxCalculator:\n    def __init__(self):\n        self.rate = 0.2\n    \n    def calculate(self, amount):\n        return amount * self.rate",
					Metadata: ChunkMetadata{
						FilePath:  "test.py",
						ClassName: "TaxCalculator",
						StartLine: 8,
						EndLine:   13,
						Language:  "python",
						ChunkType: "classes",
					},
				},
				{
					Id:      "test.py_TAX_RATE_15",
					Content: "TAX_RATE = 0.2",
					Metadata: ChunkMetadata{
						FilePath:     "test.py",
						FunctionName: "TAX_RATE",
						StartLine:    15,
						EndLine:      15,
						Language:     "python",
						ChunkType:    "variables",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "it should handle empty Python file",
			args: args{
				filePath:   "empty.py",
				sourceCode: "",
			},
			want:    []Chunk{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN
			parser := NewGenericParser()

			// WHEN
			got, err := parser.ParseFile(tt.args.filePath, []byte(tt.args.sourceCode))

			// THEN
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assertChunksEqual(t, tt.want, got)
		})
	}
}

//func TestGenericParser_ParseFile_Go(t *testing.T) {
//	type args struct {
//		filePath   string
//		sourceCode string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    []Chunk
//		wantErr bool
//	}{
//		{
//			name: "it should parse Go function and type correctly",
//			args: args{
//				filePath: "test.go",
//				sourceCode: `
//package main
//
//import "fmt"
//
//type Calculator struct {
//   rate float64
//}
//
//func (c *Calculator) Calculate(amount float64) float64 {
//   return amount * c.rate
//}
//
//func main() {
//   calc := &Calculator{rate: 0.2}
//   result := calc.Calculate(1000)
//   fmt.Printf("Result: %f\n", result)
//}
//
//const TAX_RATE = 0.2
//var GlobalVar = "test"
//`,
//			},
//			want: []Chunk{
//				{
//					ID:      "test.go_Calculate_10",
//					Content: "func (c *Calculator) Calculate(amount float64) float64 {\n    return amount * c.rate\n}",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.go",
//						FunctionName: "Calculate",
//						StartLine:    10,
//						EndLine:      12,
//						Language:     "go",
//						ChunkType:    "functions",
//					},
//				},
//				{
//					ID:      "test.go_main_14",
//					Content: "func main() {\n    calc := &Calculator{rate: 0.2}\n    result := calc.Calculate(1000)\n    fmt.Printf(\"Result: %f\\n\", result)\n}",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.go",
//						FunctionName: "main",
//						StartLine:    14,
//						EndLine:      18,
//						Language:     "go",
//						ChunkType:    "functions",
//					},
//				},
//				{
//					ID:      "test.go_Calculator_6",
//					Content: "type Calculator struct {\n    rate float64\n}",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.go",
//						FunctionName: "Calculator",
//						StartLine:    6,
//						EndLine:      8,
//						Language:     "go",
//						ChunkType:    "types",
//					},
//				},
//				{
//					ID:      "test.go_TAX_RATE_20",
//					Content: "const TAX_RATE = 0.2",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.go",
//						FunctionName: "TAX_RATE",
//						StartLine:    20,
//						EndLine:      20,
//						Language:     "go",
//						ChunkType:    "constants",
//					},
//				},
//				{
//					ID:      "test.go_GlobalVar_21",
//					Content: "var GlobalVar = \"test\"",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.go",
//						FunctionName: "GlobalVar",
//						StartLine:    21,
//						EndLine:      21,
//						Language:     "go",
//						ChunkType:    "variables",
//					},
//				},
//			},
//			wantErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// GIVEN
//			parser := NewGenericParser()
//
//			// WHEN
//			got, err := parser.ParseFile(tt.args.filePath, []byte(tt.args.sourceCode))
//
//			// THEN
//			if tt.wantErr {
//				assert.Error(t, err)
//				return
//			}
//
//			assert.NoError(t, err)
//			assertChunksEqual(t, tt.want, got)
//		})
//	}
//}
//
//func TestGenericParser_ParseFile_JavaScript(t *testing.T) {
//	type args struct {
//		filePath   string
//		sourceCode string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    []Chunk
//		wantErr bool
//	}{
//		{
//			name: "it should parse JavaScript function and class correctly",
//			args: args{
//				filePath: "test.js",
//				sourceCode: `
//function calculateTax(income) {
//   if (income > 50000) {
//       return income * 0.3;
//   } else {
//       return income * 0.2;
//   }
//}
//
//class TaxCalculator {
//   constructor() {
//       this.rate = 0.2;
//   }
//
//   calculate(amount) {
//       return amount * this.rate;
//   }
//}
//
//const TAX_RATE = 0.2;
//`,
//			},
//			want: []Chunk{
//				{
//					ID:      "test.js_calculateTax_2",
//					Content: "function calculateTax(income) {\n    if (income > 50000) {\n        return income * 0.3;\n    } else {\n        return income * 0.2;\n    }\n}",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.js",
//						FunctionName: "calculateTax",
//						StartLine:    2,
//						EndLine:      8,
//						Language:     "javascript",
//						ChunkType:    "functions",
//					},
//				},
//				{
//					ID:      "test.js_TaxCalculator_10",
//					Content: "class TaxCalculator {\n    constructor() {\n        this.rate = 0.2;\n    }\n    \n    calculate(amount) {\n        return amount * this.rate;\n    }\n}",
//					Metadata: ChunkMetadata{
//						FilePath:  "test.js",
//						ClassName: "TaxCalculator",
//						StartLine: 10,
//						EndLine:   18,
//						Language:  "javascript",
//						ChunkType: "classes",
//					},
//				},
//				{
//					ID:      "test.js_TAX_RATE_20",
//					Content: "const TAX_RATE = 0.2;",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.js",
//						FunctionName: "TAX_RATE",
//						StartLine:    20,
//						EndLine:      20,
//						Language:     "javascript",
//						ChunkType:    "variables",
//					},
//				},
//			},
//			wantErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// GIVEN
//			parser := NewGenericParser()
//
//			// WHEN
//			got, err := parser.ParseFile(tt.args.filePath, []byte(tt.args.sourceCode))
//
//			// THEN
//			if tt.wantErr {
//				assert.Error(t, err)
//				return
//			}
//
//			assert.NoError(t, err)
//			assertChunksEqual(t, tt.want, got)
//		})
//	}
//}
//
//func TestGenericParser_ParseFile_Rust(t *testing.T) {
//	type args struct {
//		filePath   string
//		sourceCode string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    []Chunk
//		wantErr bool
//	}{
//		{
//			name: "it should parse Rust struct and function correctly",
//			args: args{
//				filePath: "test.rs",
//				sourceCode: `
//struct Calculator {
//   rate: f64,
//}
//
//impl Calculator {
//   fn new(rate: f64) -> Self {
//       Calculator { rate }
//   }
//
//   fn calculate(&self, amount: f64) -> f64 {
//       amount * self.rate
//   }
//}
//
//fn main() {
//   let calc = Calculator::new(0.2);
//   let result = calc.calculate(1000.0);
//   println!("Result: {}", result);
//}
//
//const TAX_RATE: f64 = 0.2;
//`,
//			},
//			want: []Chunk{
//				{
//					ID:      "test.rs_new_7",
//					Content: "fn new(rate: f64) -> Self {\n        Calculator { rate }\n    }",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.rs",
//						FunctionName: "new",
//						StartLine:    7,
//						EndLine:      9,
//						Language:     "rust",
//						ChunkType:    "functions",
//					},
//				},
//				{
//					ID:      "test.rs_calculate_11",
//					Content: "fn calculate(&self, amount: f64) -> f64 {\n        amount * self.rate\n    }",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.rs",
//						FunctionName: "calculate",
//						StartLine:    11,
//						EndLine:      13,
//						Language:     "rust",
//						ChunkType:    "functions",
//					},
//				},
//				{
//					ID:      "test.rs_main_16",
//					Content: "fn main() {\n    let calc = Calculator::new(0.2);\n    let result = calc.calculate(1000.0);\n    println!(\"Result: {}\", result);\n}",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.rs",
//						FunctionName: "main",
//						StartLine:    16,
//						EndLine:      20,
//						Language:     "rust",
//						ChunkType:    "functions",
//					},
//				},
//				{
//					ID:      "test.rs_Calculator_2",
//					Content: "struct Calculator {\n    rate: f64,\n}",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.rs",
//						FunctionName: "Calculator",
//						StartLine:    2,
//						EndLine:      4,
//						Language:     "rust",
//						ChunkType:    "structs",
//					},
//				},
//				{
//					ID:      "test.rs_impls_6",
//					Content: "impl Calculator {\n    fn new(rate: f64) -> Self {\n        Calculator { rate }\n    }\n    \n    fn calculate(&self, amount: f64) -> f64 {\n        amount * self.rate\n    }\n}",
//					Metadata: ChunkMetadata{
//						FilePath:  "test.rs",
//						StartLine: 6,
//						EndLine:   14,
//						Language:  "rust",
//						ChunkType: "impls",
//					},
//				},
//				{
//					ID:      "test.rs_TAX_RATE_22",
//					Content: "const TAX_RATE: f64 = 0.2;",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.rs",
//						FunctionName: "TAX_RATE",
//						StartLine:    22,
//						EndLine:      22,
//						Language:     "rust",
//						ChunkType:    "constants",
//					},
//				},
//			},
//			wantErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// GIVEN
//			parser := NewGenericParser()
//
//			// WHEN
//			got, err := parser.ParseFile(tt.args.filePath, []byte(tt.args.sourceCode))
//
//			// THEN
//			if tt.wantErr {
//				assert.Error(t, err)
//				return
//			}
//
//			assert.NoError(t, err)
//			assertChunksEqual(t, tt.want, got)
//		})
//	}
//}
//
//func TestGenericParser_ParseFile_TypeScript(t *testing.T) {
//	type args struct {
//		filePath   string
//		sourceCode string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    []Chunk
//		wantErr bool
//	}{
//		{
//			name: "it should parse TypeScript interface and function correctly",
//			args: args{
//				filePath: "test.ts",
//				sourceCode: `
//interface Calculator {
//   rate: number;
//   calculate(amount: number): number;
//}
//
//function createCalculator(rate: number): Calculator {
//   return {
//       rate,
//       calculate: (amount: number) => amount * rate
//   };
//}
//
//type TaxRate = number;
//`,
//			},
//			want: []Chunk{
//				{
//					ID:      "test.ts_Calculator_2",
//					Content: "interface Calculator {\n    rate: number;\n    calculate(amount: number): number;\n}",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.ts",
//						FunctionName: "Calculator",
//						StartLine:    2,
//						EndLine:      5,
//						Language:     "typescript",
//						ChunkType:    "interfaces",
//					},
//				},
//				{
//					ID:      "test.ts_createCalculator_7",
//					Content: "function createCalculator(rate: number): Calculator {\n    return {\n        rate,\n        calculate: (amount: number) => amount * rate\n    };\n}",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.ts",
//						FunctionName: "createCalculator",
//						StartLine:    7,
//						EndLine:      12,
//						Language:     "typescript",
//						ChunkType:    "functions",
//					},
//				},
//				{
//					ID:      "test.ts_TaxRate_14",
//					Content: "type TaxRate = number;",
//					Metadata: ChunkMetadata{
//						FilePath:     "test.ts",
//						FunctionName: "TaxRate",
//						StartLine:    14,
//						EndLine:      14,
//						Language:     "typescript",
//						ChunkType:    "types",
//					},
//				},
//			},
//			wantErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// GIVEN
//			parser := NewGenericParser()
//
//			// WHEN
//			got, err := parser.ParseFile(tt.args.filePath, []byte(tt.args.sourceCode))
//
//			// THEN
//			if tt.wantErr {
//				assert.Error(t, err)
//				return
//			}
//
//			assert.NoError(t, err)
//			assertChunksEqual(t, tt.want, got)
//		})
//	}
//}

func TestGenericParser_ParseFile_UnsupportedFiles(t *testing.T) {
	type args struct {
		filePath   string
		sourceCode string
	}
	tests := []struct {
		name    string
		args    args
		want    []Chunk
		wantErr bool
	}{
		{
			name: "it should return error for unsupported file type",
			args: args{
				filePath:   "test.txt",
				sourceCode: "some text content",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "it should return error for file without extension",
			args: args{
				filePath:   "test",
				sourceCode: "some content",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN
			parser := NewGenericParser()

			// WHEN
			got, err := parser.ParseFile(tt.args.filePath, []byte(tt.args.sourceCode))

			// THEN
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assertChunksEqual(t, tt.want, got)
		})
	}
}

func TestGenericParser_detectLanguage(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "it should detect python file",
			args: args{filePath: "example/test.py"},
			want: "python",
		},
		{
			name: "it should detect go file",
			args: args{filePath: "example/test.go"},
			want: "go",
		},
		{
			name: "it should detect javascript file",
			args: args{filePath: "example/test.js"},
			want: "javascript",
		},
		{
			name: "it should detect typescript file",
			args: args{filePath: "example/test.ts"},
			want: "typescript",
		},
		{
			name: "it should detect tsx file",
			args: args{filePath: "example/test.tsx"},
			want: "typescript",
		},
		{
			name: "it should detect rust file",
			args: args{filePath: "example/test.rs"},
			want: "rust",
		},
		{
			name: "it should return empty string for unsupported file",
			args: args{filePath: "example/test.txt"},
			want: "",
		},
		{
			name: "it should return empty string for file without extension",
			args: args{filePath: "example/test"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewGenericParser()
			got, found := p.detectLanguage(tt.args.filePath)
			if tt.want == "" && found {
				t.Errorf("detectLanguage() = %v, but want 'not found'", got.LanguageName)
			} else if tt.want != "" && got.LanguageName != tt.want {
				t.Errorf("detectLanguage() = %v, want %v", got.LanguageName, tt.want)
			}
		})
	}
}

func normalizeWhitespace(s string) string {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")

	return s
}

func assertChunksEqual(t *testing.T, expected []Chunk, actual []Chunk) {
	require.Len(t, actual, len(expected), "Number of chunks should match")

	for i, expectedChunk := range expected {
		actualChunk := actual[i]

		assert.Equal(t, expectedChunk.Id, actualChunk.Id)
		assert.Equal(t, expectedChunk.Metadata, actualChunk.Metadata)

		expectedContent := normalizeWhitespace(expectedChunk.Content)
		actualContent := normalizeWhitespace(actualChunk.Content)
		assert.Equal(t, expectedContent, actualContent)
	}
}
