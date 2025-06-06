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

func init() {
	Run = mainRun
}

func mainRun() *echo.Echo {

	e := echo.New()
	config, err := config.NewConfig()
	if err != nil {
		e.Logger.Fatal(err)
		return nil
	}

	fmt.Println(config)

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

	go e.Logger.Fatal(e.Start(":" + strconv.Itoa(config.Port)))

	return e
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
      font-size:80%;
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
  <div id="container" style="display: flex; height: 80vh; overflow: hidden;">
    <!-- 左: 検索結果リスト -->
    <div id="sidebar" style="width: 25%; overflow-y: auto; padding-right: 10px;">
      {{range .Results}}
        <div class="result">
          <div class="title">
            <a href="{{$.BaseUrl}}{{.Filename}}" target="resultFrame">{{.Title}}</a>
          </div>
          <div class="content">{{.Snippet}}</div>
        </div>
      {{else}}
        {{if .Query}}<p>該当なし</p>{{end}}
      {{end}}
    </div>

    <!-- ドラッグバー -->
    <div id="divider" style="width: 5px; cursor: ew-resize; background-color: #ccc;"></div>

    <!-- 右: iframe 表示 -->
    <div style="flex-grow: 1; height: 100%;">
      <iframe name="resultFrame" style="width: 100%; height: 100%; border: none;"></iframe>
    </div>
  </div>

  <script>
    const divider = document.getElementById('divider');
    const sidebar = document.getElementById('sidebar');
    const container = document.getElementById('container');

    divider.addEventListener('mousedown', function (e) {
      e.preventDefault();
      document.addEventListener('mousemove', resize);
      document.addEventListener('mouseup', stopResize);
    });

    function resize(e) {
      const containerRect = container.getBoundingClientRect();
      const newWidth = e.clientX - containerRect.left;
      if (newWidth > 100 && newWidth < containerRect.width - 100) {
        sidebar.style.width = newWidth + 'px';
      }
    }

    function stopResize() {
      document.removeEventListener('mousemove', resize);
      document.removeEventListener('mouseup', stopResize);
    }
  </script>
</body>
</html>
`
