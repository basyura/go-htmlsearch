package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

func main() {
	db, err := sql.Open("sqlite3", "articles.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// FTS5 仮想テーブルを作成
	_, err = db.Exec(`DROP TABLE IF EXISTS articles`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE VIRTUAL TABLE articles USING fts5(filename, content)`)
	if err != nil {
		panic(err)
	}

	err = filepath.WalkDir("html_files", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".html") {
			bytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := extractText(string(bytes))
			_, err = db.Exec(`INSERT INTO articles (filename, content) VALUES (?, ?)`, d.Name(), text)
			if err != nil {
				return err
			}
			fmt.Println("登録:", d.Name())
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("完了")
}

func extractText(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}
	var f func(*html.Node)
	var sb strings.Builder
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data + " ")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return sb.String()
}
