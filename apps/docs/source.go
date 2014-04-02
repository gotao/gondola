package docs

import (
	"bytes"
	"fmt"
	"gnd.la/app"
	"gnd.la/html"
	"gnd.la/log"
	"html/template"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	highlighters = map[string]string{
		".c":    "c",
		".cpp":  "cpp",
		".css":  "css",
		".cxx":  "cpp",
		".go":   "go",
		".h":    "c",
		".hpp":  "cpp",
		".hxx":  "cpp",
		".js":   "js",
		".md":   "markdown",
		".sh":   "sh",
		".bash": "sh",
	}
)

func sourceHandler(ctx *app.Context) {
	rel := ctx.IndexValue(0)
	p := filepath.FromSlash(rel)
	dir := packageDir(filepath.Dir(p))
	path := filepath.Join(dir, filepath.Base(p))
	log.Debugf("Loading source from %s", path)
	info, err := os.Stat(path)
	if err != nil {
		ctx.NotFound("File not found")
		return
	}
	var breadcrumbs []*breadcrumb
	for ii := 0; ii < len(rel); {
		var end int
		slash := strings.IndexByte(rel[ii:], '/')
		if slash < 0 {
			end = len(rel)
		} else {
			end = ii + slash
		}
		breadcrumbs = append(breadcrumbs, &breadcrumb{
			Title: rel[ii:end],
			Href:  ctx.MustReverse(SourceHandlerName, rel[:end]),
		})
		ii = end + 1
	}
	var tmpl string
	var title string
	var files []string
	var code template.HTML
	var lines []int
	if info.IsDir() {
		if rel != "" && rel[len(rel)-1] != '/' {
			ctx.MustRedirectReverse(true, SourceHandlerName, rel+"/")
			return
		}
		contents, err := ioutil.ReadDir(path)
		if err != nil {
			panic(err)
		}
		for _, v := range contents {
			if n := v.Name(); len(n) > 0 && n[0] != '.' {
				files = append(files, n)
			}
		}
		title = "Directory " + filepath.Base(rel)
		tmpl = "dir.html"
	} else {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		contentType := http.DetectContentType(contents)
		if !strings.HasPrefix(contentType, "text") {
			ctx.Header().Set("Content-Type", contentType)
			switch contentType {
			case "image/gif", "image/png", "image/jpeg":
			default:
				ctx.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s;", filepath.Base(rel)))
			}
			ctx.Write(contents)
			return
		}
		title = "File " + filepath.Base(rel)
		var buf bytes.Buffer
		buf.WriteString("<span id=\"line-1\">")
		last := 0
		line := 1
		for ii, v := range contents {
			if v == '\n' {
				buf.WriteString(html.Escape(string(contents[last:ii])))
				lines = append(lines, line)
				last = ii
				line++
				buf.WriteString(fmt.Sprintf("</span><span id=\"line-%d\">", line))
			}
		}
		buf.Write(contents[last:])
		buf.WriteString("</span>")
		code = template.HTML(buf.String())
		tmpl = "source.html"
	}
	data := map[string]interface{}{
		"Title":       rel,
		"Header":      title,
		"Breadcrumbs": breadcrumbs,
		"Files":       files,
		"Code":        code,
		"Lines":       lines,
		"Padding":     math.Ceil(math.Log10(float64(len(lines)+1))) + 0.1,
		"Highlighter": highlighters[filepath.Ext(rel)],
	}
	ctx.MustExecute(tmpl, data)
}
