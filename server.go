package main

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/flosch/pongo2"
	"github.com/go-martini/martini"
	_ "github.com/go-sql-driver/mysql"
	//"github.com/martini-contrib/render"
	"encoding/json"
	"github.com/yuiolink/yuiolink/utils"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type config struct {
	Protocol string
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
	Password string
	Database string
}

type redirect struct {
	Uri       string
	Encrypted bool
}

var namespace = []rune("yuphjknm")

func renderHtml(templateName string, context pongo2.Context, writer http.ResponseWriter) {
	var template = pongo2.Must(pongo2.FromCache(templateName))
	template.ExecuteWriter(context, writer)
}

func renderJson(v interface{}, writer http.ResponseWriter) {
	result, err := json.Marshal(v) // TODO: Implement configurable ident

	if err != nil {
		panic(err.Error()) // TODO: Write json error with status code
	}

	writer.Write(result)
}

func main() {
	var conf config
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		panic(err.Error())
	}

	log.Info("Configuration parsed...")
	log.Infof("Configuration values: %s", conf)

	siteRootUrl := utils.BuildRootUrl(conf.Protocol, conf.Domain, conf.Port, conf.Tls)
	log.Infof("Site root URL is set to %s", siteRootUrl)
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", conf.Database.User, conf.Database.Password, conf.Database.Host, conf.Database.Port, conf.Database.Database)

	rand.Seed(time.Now().UTC().UnixNano())
	m := martini.Classic()
	m.Use(martini.Static("js", martini.StaticOptions{Prefix: "js"}))

	m.Get("/", func(response http.ResponseWriter) {
		renderHtml("templates/index.tmpl", nil, response)
	})

	m.Get("/paste", func(response http.ResponseWriter) {
		renderHtml("templates/paste.tmpl", nil, response)
	})

	m.Get("/:linkName", func(params martini.Params, response http.ResponseWriter, request *http.Request) {
		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}
		defer db.Close()

		linkName := params["linkName"]
		redirect := GetRedirectFromLinkName(db, linkName)

		if redirect.Encrypted {
			log.Info("Link is encrypted, serving decryption page")
			renderHtml("templates/encrypted.tmpl", pongo2.Context{"uri": redirect.Uri}, response)
		} else {
			log.Infof("Redirecting to %s", redirect.Uri)
			http.Redirect(response, request, redirect.Uri, 303)
		}
	})

	m.Post("/", func(request *http.Request, response http.ResponseWriter) {
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

		linkName := GenerateUniqueLinkName(db, 6)
		log.WithFields(log.Fields{
			"link_name": linkName,
			"uri":       uri,
			"encrypted": false,
		}).Info("Inserting redirect link")
		InsertRedirect(db, linkName, uri, false)

		linkUrl := siteRootUrl + linkName
		renderHtml("templates/index.tmpl", pongo2.Context{"link": linkUrl}, response)
	})

	m.Post("/paste", func(request *http.Request, response http.ResponseWriter) {
		content := request.FormValue("content")
		if content == "" {
			panic("No parameter with key content provided")
		}

		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}

		linkName := GenerateUniqueLinkName(db, 6)
		log.WithFields(log.Fields{
			"link_name": linkName,
		}).Info("Inserting paste link")
		InsertPaste(db, linkName, content, false)

		linkUrl := siteRootUrl + linkName

		renderHtml("templates/paste.tmpl", pongo2.Context{"link": linkUrl}, response)
	})
	m.Post("/api/redirect", func(request *http.Request, response http.ResponseWriter) {
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

		linkName := GenerateUniqueLinkName(db, 6)
		log.WithFields(log.Fields{
			"link_name": linkName,
			"uri":       uri,
			"encrypted": encrypted,
		}).Info("Inserting redirect link")
		InsertRedirect(db, linkName, uri, encrypted)

		linkUrl := siteRootUrl + linkName

		renderJson(linkUrl, response)
	})

	m.Post("/api/paste", func(request *http.Request, response http.ResponseWriter) {
		content := request.FormValue("content")
		if content == "" {
			panic("No parameter with key content provided")
		}

		encrypted, err := strconv.ParseBool(request.FormValue("encrypted"))
		if err != nil {
			panic("Invalid value for parameter \"encrypted\"")
		}

		db, err := sql.Open("mysql", connectionString)
		if err != nil {
			panic(err.Error())
		}

		linkName := GenerateUniqueLinkName(db, 6)
		log.WithFields(log.Fields{
			"link_name": linkName,
			"encrypted": encrypted,
		}).Info("Inserting paste link")
		InsertPaste(db, linkName, content, encrypted)

		linkUrl := siteRootUrl + linkName

		renderJson(linkUrl, response)
	})

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

func GenerateUniqueLinkName(db *sql.DB, length int) string {
	var linkName string
	for true {
		linkName = GenerateRandomLinkName(length)
		if !LinkNameExists(db, linkName) {
			break
		}
	}
	return linkName
}
