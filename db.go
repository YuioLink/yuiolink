package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

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

func LinkNameExists(db *sql.DB, linkName string) bool {
	stmtOut, err := db.Prepare("SELECT EXISTS (SELECT 1 FROM link WHERE link_name = ?)")
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	var result bool
	err = stmtOut.QueryRow(linkName).Scan(&result)
	if err != nil {
		panic(err.Error())
	}

	return result
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

func InsertLink(db *sql.DB, linkName string) int64 {
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

	return linkId
}

func InsertRedirect(db *sql.DB, linkName string, uri string, encrypted bool) bool {
	linkId := InsertLink(db, linkName)

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

func InsertPaste(db *sql.DB, linkName string, content string, encrypted bool) bool {
	linkId := InsertLink(db, linkName)

	pasteIns, err := db.Prepare("INSERT INTO paste (link_id, content, encrypted) VALUES (?, ?, ?)")
	if err != nil {
		panic(err.Error())
	}
	defer pasteIns.Close()

	_, err = pasteIns.Exec(linkId, content, encrypted)
	if err != nil {
		panic(err.Error())
	}

	return true
}
