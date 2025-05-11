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

func extractTextAndTitle(htmlContent string) (string, string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", ""
	}
	var title string
	var bodyContent string

	var findTitleAndBody func(*html.Node)
	findTitleAndBody = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = n.FirstChild.Data
		}
		if n.Type == html.ElementNode && n.Data == "body" {
			var sb strings.Builder
			var extractBodyText func(*html.Node)
			extractBodyText = func(n *html.Node) {
				if n.Type == html.TextNode {
					sb.WriteString(n.Data + " ")
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					extractBodyText(c)
				}
			}
			extractBodyText(n)
			bodyContent = strings.TrimSpace(sb.String())
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitleAndBody(c)
		}
	}
	findTitleAndBody(doc)

	return bodyContent, strings.TrimSpace(title)
}

func main() {
	db, err := sql.Open("sqlite3", "articles.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 通常テーブルを使う（FTS5 なし）
	_, _ = db.Exec(`DROP TABLE IF EXISTS articles`)
	_, err = db.Exec(`
		CREATE TABLE articles (
			filename TEXT,
			title TEXT,
			content TEXT
		)
	`)
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
			content, title := extractTextAndTitle(string(bytes))
			_, err = db.Exec(`INSERT INTO articles (filename, title, content) VALUES (?, ?, ?)`,
				d.Name(), title, content)
			if err != nil {
				return err
			}
			fmt.Printf("登録: %s（タイトル: %s）\n", d.Name(), title)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("完了")
}
