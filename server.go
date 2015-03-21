package main

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/go-martini/martini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/martini-contrib/render"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type config struct {
	SiteUrl  string
	Port     int
	Database dbConfig //`toml:"database"`
}

type dbConfig struct {
	Host     string
	Port     int
	User     string
	Database string
}

type redirect struct {
	Uri       string
	Encrypted bool
}

var namespace = []rune("yuphjknm")

func main() {
	var conf config
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		panic(err.Error())
	}

	root := conf.SiteUrl
	connectionString := fmt.Sprintf("%s@tcp(%s:%d)/%s", conf.Database.User, conf.Database.Host, conf.Database.Port, conf.Database.Database)

	fmt.Printf("Site URL: %s\n", root)
	fmt.Printf("Connection string: %s\n", connectionString)

	rand.Seed(time.Now().UTC().UnixNano())
	m := martini.Classic()
	m.Use(martini.Static("js", martini.StaticOptions{Prefix: "js"}))
	m.Use(render.Renderer())

	m.Get("/", func(renderer render.Render) {
		//db, err := sql.Open("mysql", connectionString)
		//if err != nil {
		//panic(err.Error())
		//}
		//defer db.Close()

		renderer.HTML(200, "index", nil)
	})

	m.Get("/:link", func(params martini.Params, renderer render.Render) {
		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}
		defer db.Close()

		link := params["link"]
		redirect := GetRedirectFromLink(db, link)

		if redirect.Encrypted {
			renderer.HTML(200, "encrypted", redirect.Uri)
		} else {
			renderer.Redirect(redirect.Uri)
		}
	})

	m.Post("/", func(renderer render.Render, request *http.Request) {
		url := request.FormValue("url")
		if url == "" {
			panic("No parameter with key url provided")
		}

		fmt.Printf("POST recieved, url: %v", url)

		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}

		link := GenerateRandomLink(6)
		InsertRedirect(db, link, url, false)

		linkUrl := root + link

		renderer.HTML(200, "index", linkUrl)
	})

	m.Post("/api/link", func(r render.Render, request *http.Request) {
		url := request.FormValue("url")
		if url == "" {
			panic("No parameter with key url provided")
		}

		encrypted, _ := strconv.ParseBool(request.FormValue("encrypted"))

		fmt.Printf("POST recieved, url: %v", url)

		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}

		link := GenerateRandomLink(6)
		InsertRedirect(db, link, url, encrypted)

		linkUrl := root + link

		r.JSON(200, linkUrl)
	})

	//m.Run()
	m.RunOnAddr(fmt.Sprintf(":%d", conf.Port))
}

func GenerateRandomLink(length int) string {
	buffer := make([]rune, length)
	for i := range buffer {
		buffer[i] = namespace[rand.Intn(len(namespace))]
	}
	return string(buffer)
}

func GetRedirectLinks(db *sql.DB) (string, string) {
	stmtOut, err := db.Prepare("SELECT l.link, r.redirect_uri FROM link l JOIN redirect r ON r.link_id = l.id")
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	rows, err := stmtOut.Query()
	if err != nil {
		panic(err.Error())
	}

	var uri string
	var link string
	for rows.Next() {
		if err := rows.Scan(&link, &uri); err != nil {
			panic(err.Error())
		}
	}

	return link, uri
}

func GetRedirectFromLink(db *sql.DB, link string) redirect {
	stmtOut, err := db.Prepare("SELECT r.redirect_uri AS uri, r.encrypted AS encrypted FROM link l JOIN redirect r ON r.link_id = l.id WHERE l.link = ?")
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	var uri string
	var encrypted bool
	var redirect redirect
	err = stmtOut.QueryRow(link).Scan(&uri, &encrypted)
	if err != nil {
		panic(err.Error())
	}

	redirect.Uri = uri
	redirect.Encrypted = encrypted

	return redirect
}

func GetRedirectUriFromLink(db *sql.DB, link string) string {
	stmtOut, err := db.Prepare("SELECT r.redirect_uri FROM link l JOIN redirect r ON r.link_id = l.id WHERE l.link = ?")
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	var redirectUrl string
	err = stmtOut.QueryRow(link).Scan(&redirectUrl)
	if err != nil {
		panic(err.Error())
	}

	return redirectUrl
}

func InsertRedirect(db *sql.DB, link string, uri string, encrypted bool) bool {
	linkIns, err := db.Prepare("INSERT INTO link (link, date_created) VALUES (?, NOW())")
	if err != nil {
		panic(err.Error())
	}
	defer linkIns.Close()

	result, err := linkIns.Exec(link)
	if err != nil {
		panic(err.Error())
	}

	linkId, err := result.LastInsertId()
	if err != nil {
		panic(err.Error())
	}

	redirectIns, err := db.Prepare("INSERT INTO redirect (link_id, redirect_uri, encrypted) VALUES (?, ?, ?)")
	if err != nil {
		panic(err.Error())
	}
	defer redirectIns.Close()

	_, err = redirectIns.Exec(linkId, uri, encrypted)
	if err != nil {
		panic(err.Error())
	}

	return true
}
