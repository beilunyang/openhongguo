// Package schema 读取 sdk 和 sdk/model 的源码，整理成各语言类型生成器共用的
// 描述：结构体及其字段、命名的基本类型（枚举）及其常量，连同 Go 注释。
//
// 字段的 JSON 名按 encoding/json 的规则取：带 omitempty 的字段是可选的，
// 标为 "-" 的字段不出现。没有 JSON 标签的查询参数结构体，字段名改为首字母
// 小写的驼峰（Go 解码时不区分大小写），并且全部可选。
package schema

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

// QueryTypes 是 sdk 包里作为参数传入的结构体。
var QueryTypes = []string{
	"DiscoverOptions", "SearchOptions", "EpisodeOptions", "DanmakuOptions", "LandingOptions",
	"NewReleaseQuery", "ShortFilterQuery", "ShortRankQuery",
}

// Kind 是类型引用的种类。
type Kind int

const (
	String Kind = iota
	Bool
	Int   // int、int32 等不超过 32 位的整数，以及 Go 的 int
	Int64 // int64、uint64 等
	Float
	Time  // time.Time，JSON 里是 RFC 3339 字符串
	Named // model 里的结构体或枚举
	List
	Map // 键总是字符串
)

// Ref 引用一个类型。
type Ref struct {
	Kind Kind
	Name string // Named 的类型名
	Elem *Ref   // List 和 Map 的元素
	// Pointer 标记 Go 里的指针：JSON 里可能是 null 或缺省，其余类型缺省时取零值。
	Pointer bool
}

// Field 是结构体的一个字段。
type Field struct {
	GoName   string
	JSONName string
	Optional bool
	Type     Ref
	Doc      string
}

// Const 是枚举的一个常量。
type Const struct {
	GoName string
	Value  string // Go 源码里的字面量
	Note   string // 行尾注释，没有时取上方的注释
}

// Type 是一个结构体或枚举。
type Type struct {
	Name string
	Doc  string
	// Struct 为真时是结构体：Embeds 是嵌入的结构体，Fields 是自己的字段。
	Struct bool
	Embeds []string
	Fields []Field
	// 枚举（命名的基本类型）的底层类型和常量。
	Underlying Ref
	Consts     []Const
}

// Schema 是生成器的输入。
type Schema struct {
	Model   []Type // sdk/model 的全部导出类型，按名字排序
	Queries []Type // QueryTypes，按 QueryTypes 的顺序
	byName  map[string]Type
}

// Lookup 按名字查找 model 里的类型。
func (s *Schema) Lookup(name string) (Type, bool) {
	t, ok := s.byName[name]
	return t, ok
}

// AllFields 返回结构体连同嵌入结构体的全部字段，嵌入的字段在前。
func (s *Schema) AllFields(t Type) []Field {
	var out []Field
	for _, e := range t.Embeds {
		if et, ok := s.byName[e]; ok {
			out = append(out, s.AllFields(et)...)
		}
	}
	return append(out, t.Fields...)
}

// Load 读取仓库根目录 root 下的 sdk 和 sdk/model。
func Load(root string) (*Schema, error) {
	modelDir := filepath.Join(root, "sdk", "model")
	model, err := parseDir(modelDir)
	if err != nil {
		return nil, err
	}
	sdkDecls, err := parseDir(filepath.Join(root, "sdk"))
	if err != nil {
		return nil, err
	}
	consts, err := modelConsts(modelDir)
	if err != nil {
		return nil, err
	}

	s := &Schema{byName: map[string]Type{}}
	names := make([]string, 0, len(model))
	for n := range model {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		t := convert(model[n], false)
		t.Consts = consts[n]
		s.Model = append(s.Model, t)
		s.byName[n] = t
	}
	for _, n := range QueryTypes {
		d, ok := sdkDecls[n]
		if !ok {
			return nil, fmt.Errorf("%s not found in sdk", n)
		}
		s.Queries = append(s.Queries, convert(d, true))
	}
	return s, nil
}

type decl struct {
	name string
	doc  string
	spec *ast.TypeSpec
}

func parseDir(dir string) (map[string]decl, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	out := map[string]decl{}
	for _, p := range pkgs {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				g, ok := d.(*ast.GenDecl)
				if !ok || g.Tok != token.TYPE {
					continue
				}
				for _, sp := range g.Specs {
					ts := sp.(*ast.TypeSpec)
					if !ts.Name.IsExported() || ts.TypeParams != nil {
						continue
					}
					doc := ts.Doc
					if doc == nil {
						doc = g.Doc
					}
					out[ts.Name.Name] = decl{name: ts.Name.Name, doc: doc.Text(), spec: ts}
				}
			}
		}
	}
	return out, nil
}

// modelConsts 收集每个枚举类型的常量，保持源码顺序。
func modelConsts(dir string) (map[string][]Const, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	// 按文件名排序，保证输出稳定。
	var files []*ast.File
	var fnames []string
	byName := map[string]*ast.File{}
	for _, p := range pkgs {
		for n, f := range p.Files {
			fnames = append(fnames, n)
			byName[n] = f
		}
	}
	sort.Strings(fnames)
	for _, n := range fnames {
		files = append(files, byName[n])
	}

	out := map[string][]Const{}
	for _, f := range files {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, sp := range g.Specs {
				vs := sp.(*ast.ValueSpec)
				id, ok := vs.Type.(*ast.Ident)
				if !ok || len(vs.Values) != 1 {
					continue
				}
				lit, ok := vs.Values[0].(*ast.BasicLit)
				if !ok {
					continue
				}
				note := strings.TrimSpace(vs.Comment.Text())
				if note == "" {
					note = strings.TrimSpace(strings.ReplaceAll(vs.Doc.Text(), "\n", " "))
				}
				out[id.Name] = append(out[id.Name], Const{GoName: vs.Names[0].Name, Value: lit.Value, Note: note})
			}
		}
	}
	return out, nil
}

func convert(d decl, query bool) Type {
	t := Type{Name: d.name, Doc: strings.TrimSpace(d.doc)}
	st, ok := d.spec.Type.(*ast.StructType)
	if !ok {
		t.Underlying = ref(d.spec.Type)
		return t
	}
	t.Struct = true
	for _, fl := range st.Fields.List {
		if len(fl.Names) == 0 { // 嵌入的结构体
			t.Embeds = append(t.Embeds, typeName(fl.Type))
			continue
		}
		for _, n := range fl.Names {
			if !n.IsExported() {
				continue
			}
			key, optional, skip := fieldKey(n.Name, fl.Tag, query)
			if skip {
				continue
			}
			doc := fl.Doc.Text()
			if doc == "" {
				doc = fl.Comment.Text()
			}
			t.Fields = append(t.Fields, Field{
				GoName: n.Name, JSONName: key, Optional: optional,
				Type: ref(fl.Type), Doc: strings.TrimSpace(doc),
			})
		}
	}
	return t
}

func fieldKey(name string, tag *ast.BasicLit, query bool) (key string, optional, skip bool) {
	if query || tag == nil {
		r := []rune(name)
		r[0] = unicode.ToLower(r[0])
		return string(r), true, false
	}
	jt := reflect.StructTag(strings.Trim(tag.Value, "`")).Get("json")
	if jt == "-" {
		return "", false, true
	}
	parts := strings.Split(jt, ",")
	key = parts[0]
	if key == "" {
		key = name
	}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			optional = true
		}
	}
	return key, optional, false
}

func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return typeName(t.X)
	}
	return "unknown"
}

func ref(e ast.Expr) Ref {
	switch t := e.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string":
			return Ref{Kind: String}
		case "bool":
			return Ref{Kind: Bool}
		case "int", "int8", "int16", "int32", "uint8", "uint16":
			return Ref{Kind: Int}
		case "int64", "uint", "uint32", "uint64":
			return Ref{Kind: Int64}
		case "float32", "float64":
			return Ref{Kind: Float}
		}
		return Ref{Kind: Named, Name: t.Name}
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok && x.Name == "time" && t.Sel.Name == "Time" {
			return Ref{Kind: Time}
		}
		return Ref{Kind: Named, Name: t.Sel.Name}
	case *ast.StarExpr:
		r := ref(t.X)
		r.Pointer = true
		return r
	case *ast.ArrayType:
		elem := ref(t.Elt)
		return Ref{Kind: List, Elem: &elem}
	case *ast.MapType:
		elem := ref(t.Value)
		return Ref{Kind: Map, Elem: &elem}
	}
	panic(fmt.Sprintf("schema: unsupported type %T", e))
}

// Words 把 Go 的驼峰名字拆成单词，连续的大写字母算一个缩写词：
// "CoverURL" → ["Cover", "URL"]，"PPTRecommend" → ["PPT", "Recommend"]。
func Words(name string) []string {
	r := []rune(name)
	var out []string
	start := 0
	for i := 1; i < len(r); i++ {
		lower := unicode.IsLower(r[i-1]) || unicode.IsDigit(r[i-1])
		if unicode.IsUpper(r[i]) && (lower || (i+1 < len(r) && unicode.IsLower(r[i+1]))) {
			out = append(out, string(r[start:i]))
			start = i
		}
	}
	return append(out, string(r[start:]))
}

// EnumPrefix 返回枚举常量名共同的单词前缀，如 "Content"（ContentShortSeries、
// ContentMovie……）。去掉前缀后不能剩下空名字或以数字开头的名字。
func EnumPrefix(consts []Const) string {
	if len(consts) < 2 {
		return ""
	}
	prefix := Words(consts[0].GoName)
	for _, c := range consts[1:] {
		w := Words(c.GoName)
		n := 0
		for n < len(prefix) && n < len(w)-1 && prefix[n] == w[n] {
			n++
		}
		prefix = prefix[:n]
	}
	return strings.Join(prefix, "")
}
