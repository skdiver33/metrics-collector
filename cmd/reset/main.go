package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"html/template"
	"os"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

type StructScel struct {
	StructName         string
	OrigFields         map[string]string
	OrigFieldsStruct   map[string]string
	SliesFields        map[string]string
	MapFields          []string
	PointerBTFields    map[string]string
	PointerStructFieds map[string]string
}

func NewStructScel() *StructScel {
	return &StructScel{
		OrigFields:         make(map[string]string),
		OrigFieldsStruct:   make(map[string]string),
		SliesFields:        make(map[string]string),
		MapFields:          make([]string, 0),
		PointerBTFields:    make(map[string]string),
		PointerStructFieds: make(map[string]string),
	}
}

var templateStr = `
func(str *{{.StructName}}) Reset() {
	{{if .OrigFields}} {{ range $k,$v:=.OrigFields}}str.{{$k}}=zero{{$v}}
	{{end}}{{end}}
	{{if .MapFields}} {{ range $k,$v:=.MapFields}}clear(str.{{$v}})
	{{end}}{{end}}
	{{if .SliesFields}} {{ range $k,$v:=.SliesFields}}str.{{$k}} = str.{{$k}}[:0]
	{{end}}{{end}}
	{{if .PointerBTFields}} {{ range $k,$v:=.PointerBTFields}}
	if str.{{$k}} != nil {
	*str.{{$k}} = ""
	}
	{{end}}{{end}}
	{{if .PointerStructFieds}} {{ range $k,$v:=.PointerStructFieds}}
	if resetter, ok := str.{{$k}}.(interface{ Reset() }); ok && str.{{$k}} != nil {
        resetter.Reset()
		str.{{$k}} = nil
    }
	{{end}}{{end}}
	{{if .OrigFieldsStruct}} {{ range $k,$v:=.OrigFieldsStruct}}
	if resetter, ok := str.{{$k}}.(interface{ Reset() }); ok {
        resetter.Reset()
    }
	{{end}}{{end}}
} 
`

var tmpl = template.Must(template.New("parseStruct").Parse(templateStr))
var baseTypes = "int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64  byte rune float32 float64 complex64 complex128 bool string"

var StructAnalizer = &analysis.Analyzer{
	Name: "structAnalyzer",
	Doc:  "find all struct",
	Run:  run,
}
var processedFile = make([]string, 0)

func prepareResetFile(pkgPath string, pkgName string) error {
	baseStr := fmt.Sprintf(`
		package %s
var(
	zeroint int
	zeroint8 int8
	zeroint16 int16
	zeeroint32 int32
	zeroint64 int64
	zerouint uint
	zerouint8 uint8
	zerouint16 uint16
	zerouint32 uint32
	zerouint64 uint64
	zerobyte byte
	zerorune rune
	zerofloat32 float32
	zerofloat64 float64
	zerocomplex64 complex64
	zerocomplex128 complex128
	zerobool bool
	zerostring string
)`, pkgName)
	fmtBytes, err := format.Source([]byte(baseStr))
	if err != nil {
		return err
	}
	if _, err := os.Stat(pkgPath + "/reset.gen.go"); errors.Is(err, os.ErrNotExist) {
		err = os.WriteFile(fmt.Sprintf("%s/reset.gen.go", pkgPath), fmtBytes, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeResetMethod(writeStruct *StructScel, pkgPath string) error {
	f, err := os.OpenFile(pkgPath+"/reset.gen.go", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, writeStruct)
	if err != nil {
		return err
	}
	bufFmt, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	if _, err = f.Write(bufFmt); err != nil {
		return err
	}
	return nil
}

func run(pass *analysis.Pass) (interface{}, error) {
	packageName := pass.Pkg.Name()

	for _, fast := range pass.Files {
		fileName := pass.Fset.File(fast.Pos()).Name()

		if slices.Contains(processedFile, fileName) {
			continue
		}
		processedFile = append(processedFile, fileName)
		fmt.Println(strings.Cut(fileName, packageName))
		pkg, _, _ := strings.Cut(fileName, packageName)
		pkgPath := pkg + packageName

		ast.Inspect(fast, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.File:
				for _, decl := range x.Decls {
					var genDecl *ast.GenDecl
					var ok bool
					if genDecl, ok = decl.(*ast.GenDecl); !ok {
						continue
					}
					createReset := false
					if genDecl.Doc == nil {
						continue
					}
					for _, comment := range genDecl.Doc.List {
						if comment.Text == "// generate:reset" {
							createReset = true
							prepareResetFile(pkgPath, packageName)
						}
					}
					if !createReset {
						continue
					}
					if genDecl.Tok != token.TYPE {
						continue
					}
					for _, specs := range genDecl.Specs {
						typeSpec := specs.(*ast.TypeSpec)
						var structTypeSpec *ast.StructType
						var ok bool
						if structTypeSpec, ok = typeSpec.Type.(*ast.StructType); !ok {
							continue
						}
						parseStruct := NewStructScel()
						parseStruct.StructName = typeSpec.Name.Name
						for _, field := range structTypeSpec.Fields.List {
							if len(field.Names) > 1 {
								continue
							}
							fieldName := field.Names[0].Name
							switch fieldType := field.Type.(type) {
							case *ast.Ident:
								if strings.Contains(baseTypes, fieldType.Name) {
									parseStruct.OrigFields[fieldName] = fieldType.Name
								} else {
									parseStruct.OrigFieldsStruct[fieldName] = fieldType.Name
								}
							case *ast.SelectorExpr:
								if pkg, ok := fieldType.X.(*ast.Ident); ok {
									parseStruct.OrigFieldsStruct[fieldName] = pkg.Name + "." + fieldType.Sel.Name
								}
							case *ast.MapType:
								parseStruct.MapFields = append(parseStruct.MapFields, fieldName)
							case *ast.ArrayType:
								elemType := ""
								if et, ok := fieldType.Elt.(*ast.Ident); ok {
									elemType = et.Name
								}
								parseStruct.SliesFields[fieldName] = elemType
							case *ast.StarExpr:
								pointType := ""
								if pt, ok := fieldType.X.(*ast.Ident); ok {
									pointType = pt.Name
								}
								if strings.Contains(baseTypes, pointType) {
									parseStruct.PointerBTFields[fieldName] = pointType
								} else {
									parseStruct.PointerStructFieds[fieldName] = pointType
								}
							}

						}
						writeResetMethod(parseStruct, pkgPath)
					}
				}
				return false
			}
			return true
		})
	}
	return nil, nil
}

func main() {
	singlechecker.Main(StructAnalizer)
}
