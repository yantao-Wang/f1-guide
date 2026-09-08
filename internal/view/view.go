// Package view 提供服务端渲染视图层：模板与静态资源以 embed 方式打包进二进制。
//
// 模板组织方式：
//   - layout.html 定义公共骨架（导航/页脚），通过 {{block "content" .}} 注入页面主体
//   - 每个页面模板定义两个块：页面入口（如 "home"）与主体内容（"content"）
//   - Go 模板的 {{template}} 只接受常量名，且同名块后定义者覆盖先定义者，
//     因此每个页面单独构建一个模板集（layout + 该页），content 块在各集合中互不冲突
package view

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Page 是所有页面的公共渲染数据。
type Page struct {
	// Title 页面标题（出现在 <title> 与页面主标题）。
	Title string
	// Active 导航高亮项：home / stories / about / ""。
	Active string
	// Data 页面专属数据，由 content 块消费。
	Data any
}

// paragraphs 把以空行分隔的正文切分为段落（内容流水线产出的正文约定）。
func paragraphs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, "\n\n") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

var funcs = template.FuncMap{
	"paragraphs": paragraphs,
}

// pageNames 是全部页面模板名（与 templates/*.html 文件名对应）。
var pageNames = []string{
	"home",
	"drivers",
	"driver_detail",
	"tracks",
	"track_detail",
	"moments",
	"moment_detail",
	"schedule",
	"data",
	"about",
	"stub",
	"error_404",
}

// pageSets 为每个页面构建独立模板集：layout + 该页文件。
var pageSets = func() map[string]*template.Template {
	sets := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		t := template.New(name).Funcs(funcs)
		t = template.Must(t.ParseFS(templatesFS, "templates/layout.html", "templates/"+name+".html"))
		sets[name] = t
	}
	return sets
}()

// Render 渲染指定页面模板到 w。未知页面名返回错误。
func Render(w io.Writer, name string, page Page) error {
	t, ok := pageSets[name]
	if !ok {
		return fmt.Errorf("view: unknown page %q", name)
	}
	return t.ExecuteTemplate(w, name, page)
}

// Static 返回静态资源处理器，挂载于 /static/*（内部按前缀剥离，全部来自 embed）。
func Static() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // embed 静态资源在构建期已保证存在
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}
