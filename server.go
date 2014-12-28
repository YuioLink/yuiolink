package main

import (
	"database/sql"
	"fmt"
	"github.com/go-martini/martini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/martini-contrib/render"
	"math/rand"
	"net/http"
	"time"
)

var namespace = []rune("yuphjknm")

func main() {
	const root = "http://localhost:3000/"
	const connectionString = "root@tcp(10.211.55.8:3306)/link"

	rand.Seed(time.Now().UTC().UnixNano())
	m := martini.Classic()
	m.Use(render.Renderer())

	m.Get("/", func(renderer render.Render) {
		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}
		defer db.Close()

		renderer.HTML(200, "index", nil)
	})

	m.Get("/:link", func(params martini.Params, renderer render.Render) {
		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}
		defer db.Close()

		link := params["link"]
		redirectUri := GetRedirectUriFromLink(db, link)

		renderer.Redirect(redirectUri)
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
		InsertRedirectLink(db, link, url)

		linkUrl := root + link

		renderer.HTML(200, "index", linkUrl)
	})

	m.Run()
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

func InsertRedirect(db *sql.DB, link string, uri string) bool {
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

	redirectIns, err := db.Prepare("INSERT INTO redirect (link_id, redirect_uri) VALUES (?, ?)")
	if err != nil {
		panic(err.Error())
	}
	defer redirectIns.Close()

	_, err = redirectIns.Exec(linkId, uri)
	if err != nil {
		panic(err.Error())
	}

	return true
}
