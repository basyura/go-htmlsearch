package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"htmlsearch/config"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

type Result struct {
	Filename string
	Title    string
	Snippet  string
}

func main() {
	if err := doMain(); err != nil {
		fmt.Println(err)
	}
}

func doMain() error {

	config, err := config.NewConfig()
	if err != nil {
		return err
	}

	fmt.Println(config)

	e := echo.New()

	db, err := sql.Open("sqlite3", config.DbFile)
	if err != nil {
		e.Logger.Fatal("DB open error:", err)
	}
	defer db.Close()

	t := template.Must(template.New("search").Parse(htmlTemplate))

	e.GET("/", func(c echo.Context) error {
		rawQuery := c.QueryParam("q")
		var results []Result

		if rawQuery != "" {
			words := strings.Fields(rawQuery)
			if len(words) == 0 {
				return t.Execute(c.Response(), map[string]interface{}{
					"Results": nil,
					"Query":   rawQuery,
				})
			}

			var whereParts []string
			var args []interface{}
			for _, word := range words {
				whereParts = append(whereParts, "content LIKE ?")
				args = append(args, "%"+word+"%")
			}
			whereClause := strings.Join(whereParts, " AND ")
			argsWithPreview := append([]interface{}{words[0]}, args...)

			fmt.Println(args)

			query := `
				SELECT filename, title, substr(content, instr(content, ?) - 10, 500)
				FROM articles
				WHERE ` + whereClause + `
			`

			rows, err := db.Query(query, argsWithPreview...)
			if err != nil {
				c.Logger().Error("DB query error:", err)
				return c.String(http.StatusInternalServerError, err.Error())
			}
			defer rows.Close()

			for rows.Next() {
				var r Result
				if err := rows.Scan(&r.Filename, &r.Title, &r.Snippet); err != nil {
					c.Logger().Error("Row scan error:", err)
					return c.String(http.StatusInternalServerError, err.Error())
				}
				results = append(results, r)
			}
		}

		return t.Execute(c.Response(), map[string]any{
			"Results": results,
			"Query":   rawQuery,
			"BaseUrl": config.BaseUrl,
		})
	})

	e.Logger.Fatal(e.Start(":" + strconv.Itoa(config.Port)))

	return nil
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <title>部分一致検索</title>
  <style>
  * {
      line-height:130%;
  }
  .result {
      margin-bottom:20px;
  }
  .title {
      font-weight:bold;
  }
  .content {
  }
  </style>
</head>
<body>
  <h1>Jira 検索</h1>
  <form method="GET" action="/">
    <input type="text" name="q" value="{{.Query}}" style="width:300px;">
    <input type="submit" value="検索">
  </form>
  <hr>
  {{range .Results}}
    <div class="result">
      <div class="title">
        <a href="{{$.BaseUrl}}{{.Filename}}" target="_blank">{{.Title}}</a>
      </div>
      <div class="content">{{.Snippet}}</div>
    </div>
  {{else}}
    {{if .Query}}<p>該当なし</p>{{end}}
  {{end}}
</body>
</html>
`
