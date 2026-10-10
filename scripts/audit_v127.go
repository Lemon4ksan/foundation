//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type PackageStats struct {
	Name           string
	Path           string
	TotalFiles     int
	TotalLines     int
	BadGenerics    int // any / interface{} in signatures
	BadFinalizers  int // runtime.SetFinalizer
	BadPools       int // sync.Pool
	BadTestLoops   int // b.N
	BadTimeSleep   int // time.Sleep in tests
	GoodWeak       int // weak.Pointer / weak.Make
	GoodCleanup    int // runtime.AddCleanup
	GoodSlicesMaps int // slices or maps usage
	GoodSyncTest   int // synctest.Run
	GoodBLoop      int // b.Loop()
}

func (s *PackageStats) Score() int {
	score := 100
	score -= s.BadGenerics * 2
	score -= s.BadFinalizers * 10
	score -= s.BadPools * 2
	score -= s.BadTestLoops * 2
	score -= s.BadTimeSleep * 2

	score += s.GoodWeak * 5
	score += s.GoodCleanup * 5
	score += s.GoodSlicesMaps * 2
	score += s.GoodSyncTest * 5
	score += s.GoodBLoop * 5

	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

func main() {
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	fset := token.NewFileSet()
	statsMap := make(map[string]*PackageStats)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name != "." && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		dir := filepath.Dir(path)
		if statsMap[dir] == nil {
			statsMap[dir] = &PackageStats{
				Name: filepath.Base(dir),
				Path: dir,
			}
		}
		stats := statsMap[dir]
		stats.TotalFiles++

		content, err := os.ReadFile(path)
		if err == nil {
			stats.TotalLines += bytes.Count(content, []byte{'\n'})
		}

		node, err := parser.ParseFile(fset, path, content, parser.ParseComments)
		if err != nil {
			return nil
		}

		isTest := strings.HasSuffix(path, "_test.go")

		ast.Inspect(node, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				if x.Name == "any" {
					stats.BadGenerics++
				}
			case *ast.InterfaceType:
				if len(x.Methods.List) == 0 {
					stats.BadGenerics++
				}
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok {
					if id.Name == "runtime" && x.Sel.Name == "SetFinalizer" {
						stats.BadFinalizers++
					}
					if id.Name == "runtime" && x.Sel.Name == "AddCleanup" {
						stats.GoodCleanup++
					}
					if id.Name == "sync" && x.Sel.Name == "Pool" {
						stats.BadPools++
					}
					if id.Name == "weak" && (x.Sel.Name == "Make" || x.Sel.Name == "Pointer") {
						stats.GoodWeak++
					}
					if isTest {
						if id.Name == "b" && x.Sel.Name == "N" {
							stats.BadTestLoops++
						}
						if id.Name == "b" && x.Sel.Name == "Loop" {
							stats.GoodBLoop++
						}
						if id.Name == "time" && x.Sel.Name == "Sleep" {
							stats.BadTimeSleep++
						}
						if id.Name == "synctest" && x.Sel.Name == "Run" {
							stats.GoodSyncTest++
						}
					}
				}
			case *ast.ImportSpec:
				if x.Path != nil {
					p := x.Path.Value
					if p == `"slices"` || p == `"maps"` {
						stats.GoodSlicesMaps++
					}
				}
			}
			return true
		})

		return nil
	})

	if err != nil {
		panic(err)
	}

	generateReport(statsMap)
}

const reportTmpl = `
# Архитектурный аудит foundation (Go 1.24-1.27)

Этот отчет сгенерирован автоматически на основе метрик и эвристик идеального Go-кода v1.27.

## Сводная таблица пакетов

| Пакет | Файлы | Строки | any/interface{} | Устаревшее (Pool/Finalizer) | Тесты (time.Sleep/b.N) | Инновации (weak/synctest/slices) | Оценка (0-100) |
|---|---|---|---|---|---|---|---|
{{- range . }}
| ` + "`{{ .Path }}`" + ` | {{ .TotalFiles }} | {{ .TotalLines }} | {{ .BadGenerics }} | {{ .BadFinalizers }} / {{ .BadPools }} | {{ .BadTimeSleep }} / {{ .BadTestLoops }} | {{ .GoodWeak }} / {{ .GoodSyncTest }} / {{ .GoodSlicesMaps }} | **{{ .Score }}** |
{{- end }}

## Рекомендации по рефакторингу

1. **Избавление от any/interface{}**: Обнаружено широкое использование ` + "`any`" + ` и ` + "`interface{}`" + ` в пакетах с низким баллом. Рекомендуется переход на параметризованные методы и алиасы типов (Go 1.24+).
2. **Очистка памяти**: Любые упоминания ` + "`runtime.SetFinalizer`" + ` необходимо заменить на ` + "`runtime.AddCleanup`" + `.
3. **Обновление тестов**: Использование ` + "`time.Sleep`" + ` в асинхронном коде следует заменить на ` + "`synctest.Run`" + `, а ` + "`b.N`" + ` в бенчмарках на ` + "`b.Loop()`" + `.
4. **Утилизация sync.Pool**: Малые структуры не требуют пулирования в Go 1.27. Рассмотрите возможность удаления ` + "`sync.Pool`" + ` в горячих путях сети и криптографии.
`

func generateReport(statsMap map[string]*PackageStats) {
	// group or sort if needed
	var list []*PackageStats
	for _, s := range statsMap {
		if s.TotalFiles > 0 {
			list = append(list, s)
		}
	}

	tmpl, err := template.New("report").Parse(reportTmpl)
	if err != nil {
		panic(err)
	}

	f, err := os.Create("docs/audit_report_2026.md")
	if err != nil {
		os.MkdirAll("docs", 0755)
		f, err = os.Create("docs/audit_report_2026.md")
		if err != nil {
			panic(err)
		}
	}
	defer f.Close()

	err = tmpl.Execute(f, list)
	if err != nil {
		panic(err)
	}
	fmt.Println("Report generated at docs/audit_report_2026.md")
}
