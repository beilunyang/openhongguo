package main

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode"
)

// group 是方法表的一节。
type group struct {
	title   string
	methods []string // Go 的方法名或 sdk 的包级函数名
}

// groups 决定方法在文档里的分节和顺序。新增方法时要加进这里，否则测试失败。
var groups = []group{
	{"发现与搜索", []string{"Discover", "DiscoverPager", "Search", "SearchPager", "Suggest", "HotSearch", "SearchCues", "NewReleases"}},
	{"短剧与分集", []string{"GetSeries", "GetSeriesBatch", "ListEpisodes", "ListDanmaku", "Seasons", "SeriesRecommend", "GetAlbum"}},
	{"榜单与分类", []string{"ShortRank", "ShortFilterPanel", "ShortFilter", "RankRows", "RankGroups", "Ranking", "Categories", "CategoryLanding"}},
	{"演员", []string{"Person", "PersonWorks"}},
	{"播放与设备", []string{"ResolvePlayback", "ContentKey", "ResetDevice"}},
}

// renamed 是其他语言里名字不同的方法；空字符串表示其他语言没有这个方法。
var renamed = map[string]string{
	"DiscoverPager": "discoverAll",
	"SearchPager":   "searchAll",
	"ContentKey":    "",
}

// otherNote 说明其他语言里没有的方法在哪里。
var otherNote = map[string]string{
	"ContentKey": "并入 `resolvePlayback` 的结果",
}

// methodTables 生成分节的方法表。goNames 为真时同时列出 Go 和其他语言的
// 写法（根目录 README），否则只列其他语言的写法（各语言的 README）。
func methodTables(a *api, goNames bool) (string, error) {
	var b strings.Builder
	for i, g := range groups {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "### %s\n\n", g.title)
		if goNames {
			b.WriteString("| Go | 其他语言 | 作用 |\n|---|---|---|\n")
		} else {
			b.WriteString("| 方法 | 作用 |\n|---|---|\n")
		}
		for _, name := range g.methods {
			f, ok := a.funcs[name]
			if !ok {
				return "", fmt.Errorf("method %s not found in sdk", name)
			}
			desc := cell(a.synopsis(name, f.Doc))
			other := bindingCall(name, f.Decl)
			switch {
			case goNames:
				o := "`" + other + "`"
				if other == "" {
					o = otherNote[name]
				}
				fmt.Fprintf(&b, "| `%s` | %s | %s |\n", name, o, desc)
			case other != "":
				fmt.Fprintf(&b, "| `%s` | %s |\n", other, desc)
			}
		}
	}
	return b.String(), nil
}

// bindingName 是 Go 方法 name 在其他语言里的名字，没有时为空。
func bindingName(name string) string {
	if n, ok := renamed[name]; ok {
		return n
	}
	r := []rune(name)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// bindingCall 是其他语言的调用写法，如 search(query, options)：参数取自
// Go 的签名，去掉 ctx 和不用的参数。
func bindingCall(name string, decl *ast.FuncDecl) string {
	n := bindingName(name)
	if n == "" {
		return ""
	}
	var params []string
	for _, field := range decl.Type.Params.List {
		if sel, ok := field.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Context" {
			continue
		}
		for _, id := range field.Names {
			if id.Name != "_" {
				params = append(params, paramName(id.Name))
			}
		}
	}
	return n + "(" + strings.Join(params, ", ") + ")"
}

// paramName 把 Go 的参数名改成其他语言的写法：seriesID → seriesId，
// opts → options，q → query。
func paramName(s string) string {
	switch s {
	case "opts":
		return "options"
	case "q":
		return "query"
	}
	s = strings.ReplaceAll(s, "IDs", "Ids")
	return strings.ReplaceAll(s, "ID", "Id")
}
