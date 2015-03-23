package main

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/martini-contrib/render"
	"math/rand"
	"net/http"
	"strconv"
	"time"
	"yuio.link/utils"
)

type config struct {
	Domain   string
	Tls      bool
	TlsCert  string
	TlsKey   string
	BindIp   string
	Port     int
	Database dbConfig
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

	log.Info("Configuration parsed...")
	log.Infof("Configuration values: %s", conf)

	siteRootUrl := utils.BuildRootUrl(conf.Domain, conf.Port, conf.Tls)
	log.Infof("Site root URL is set to %s", siteRootUrl)
	connectionString := fmt.Sprintf("%s@tcp(%s:%d)/%s", conf.Database.User, conf.Database.Host, conf.Database.Port, conf.Database.Database)

	rand.Seed(time.Now().UTC().UnixNano())
	m := martini.Classic()
	m.Use(martini.Static("js", martini.StaticOptions{Prefix: "js"}))
	m.Use(render.Renderer())

	m.Get("/", func(renderer render.Render) {
		renderer.HTML(200, "index", nil)
	})

	m.Get("/:linkName", func(params martini.Params, renderer render.Render) {
		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}
		defer db.Close()

		linkName := params["linkName"]
		redirect := GetRedirectFromLinkName(db, linkName)

		if redirect.Encrypted {
			log.Info("Link is encrypted, serving decryption page")
			renderer.HTML(200, "encrypted", redirect.Uri)
		} else {
			log.Infof("Redirecting to %s", redirect.Uri)
			renderer.Redirect(redirect.Uri)
		}
	})

	m.Post("/", func(renderer render.Render, request *http.Request) {
		uri := request.FormValue("uri")
		if uri == "" {
			panic("No parameter with key uri provided")
		}

		log.WithFields(log.Fields{
			"uri":       uri,
			"encrypted": false,
		}).Info("Creating link")

		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}

		linkName := GenerateRandomLinkName(6)
		log.WithFields(log.Fields{
			"link_name": linkName,
			"uri":       uri,
			"encrypted": false,
		}).Info("Inserting redirect link")
		InsertRedirect(db, linkName, uri, false)

		linkUrl := siteRootUrl + linkName

		renderer.HTML(200, "index", linkUrl)
	})

	m.Post("/api/redirect", func(r render.Render, request *http.Request) {
		uri := request.FormValue("uri")
		if uri == "" {
			panic("No parameter with key url provided")
		}

		encrypted, err := strconv.ParseBool(request.FormValue("encrypted"))
		if err != nil {
			panic("Invalid value for parameter \"encrypted\"")
		}

		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}

		linkName := GenerateRandomLinkName(6)
		log.WithFields(log.Fields{
			"link_name": linkName,
			"uri":       uri,
			"encrypted": encrypted,
		}).Info("Inserting redirect link")
		InsertRedirect(db, linkName, uri, encrypted)

		linkUrl := siteRootUrl + linkName

		r.JSON(200, linkUrl)
	})

	m.Post("/api/paste", func(r render.Render, request *http.Request) {})

	binding := fmt.Sprintf("%s:%d", conf.BindIp, conf.Port)
	if conf.Tls {
		http.ListenAndServeTLS(binding, conf.TlsCert, conf.TlsKey, m)
	} else {
		m.RunOnAddr(binding)
	}
}

func GenerateRandomLinkName(length int) string {
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

func GetRedirectFromLinkName(db *sql.DB, linkName string) redirect {
	stmtOut, err := db.Prepare("SELECT r.redirect_uri AS uri, r.encrypted AS encrypted FROM link l JOIN redirect r ON r.link_id = l.id WHERE l.link_name = ?")
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	var uri string
	var encrypted bool
	var redirect redirect
	err = stmtOut.QueryRow(linkName).Scan(&uri, &encrypted)
	if err != nil {
		panic(err.Error())
	}

	redirect.Uri = uri
	redirect.Encrypted = encrypted

	return redirect
}

func GetRedirectUriFromLinkName(db *sql.DB, linkName string) string {
	stmtOut, err := db.Prepare("SELECT r.redirect_uri FROM link l JOIN redirect r ON r.link_id = l.id WHERE l.link_name = ?")
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	var redirectUrl string
	err = stmtOut.QueryRow(linkName).Scan(&redirectUrl)
	if err != nil {
		panic(err.Error())
	}

	return redirectUrl
}

func InsertRedirect(db *sql.DB, linkName string, uri string, encrypted bool) bool {
	linkIns, err := db.Prepare("INSERT INTO link (link_name, date_created) VALUES (?, NOW())")
	if err != nil {
		panic(err.Error())
	}
	defer linkIns.Close()

	result, err := linkIns.Exec(linkName)
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
