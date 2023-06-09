package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"io"
	"net/http"
	"os"
)

const (
	ExitCodeOK   = 0
	ExitCodeFail = 1
)

type CLI struct {
	outStream, errStream io.Writer
}

var (
	db  *sqlx.DB
	err error
)

type CrawlerSite struct {
	Domain               string `db:"domain"`
	URL                  string `db:"url"`
	Block                string `db:"block"`
	ArticleLinkFromBlock string `db:"article_link_from_block"`
	Title                string `db:"title"`
	ArticleUpdatedAt     string `db:"article_updated_at"`
}

func getEnv(key string, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}

func connectDB() (*sqlx.DB, error) {
	config := mysql.NewConfig()
	config.Net = "tcp"
	config.Addr = getEnv("DB_HOST", "127.0.0.1") + ":" + getEnv("DB_PORT", "3306")
	config.User = getEnv("DB_USER", "davy_elton")
	config.Passwd = getEnv("DB_PASSWORD", "password")
	config.DBName = getEnv("DB_NAME", "schaben_local")
	config.ParseTime = true
	dsn := config.FormatDSN()

	return sqlx.Open("mysql", dsn)
}

func NewCLI(outStream, errStream io.Writer) *CLI {
	return &CLI{outStream: outStream, errStream: errStream}
}

func main() {
	cmd := NewCLI(os.Stdout, os.Stderr)
	os.Exit(cmd.execute())
}

func (c *CLI) execute() int {
	db, err = connectDB()
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
		}
	}(db)

	var crawlerSite []CrawlerSite
	query := "SELECT `cs`.`domain`, `cs`.`url`, `css`.`block`, `css`.`article_link_from_block`, `css`.`title`, `css`.`article_updated_at` FROM `crawler_site` as `cs` " +
		"JOIN `crawler_site_setting` as `css` ON (`cs`.`id` = `css`.`crawler_site_id`) "
	if err := db.Select(&crawlerSite, query); err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	resp, err := http.Get(crawlerSite[0].URL)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			_, _ = fmt.Fprintln(c.errStream, err.Error())
		}
	}(resp.Body)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		_, _ = fmt.Fprintln(c.errStream, err.Error())
		return ExitCodeFail
	}

	doc.Find(crawlerSite[0].Block).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		s.Find(crawlerSite[0].ArticleLinkFromBlock).EachWithBreak(func(i int, s *goquery.Selection) bool {
			aURL, exists := s.Attr("href")
			if exists != true {
				fmt.Println("not found href.")
				return false
			}

			fmt.Println(aURL)

			return true
		})

		return true
	})

	return ExitCodeOK
}
