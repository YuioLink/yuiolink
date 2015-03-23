package utils

import (
	"fmt"
)

func BuildRootUrl(domain string, port int, tls bool) string {
	var rootUrl string
	if tls {
		rootUrl = fmt.Sprintf("https://%s", domain)
	} else {
		rootUrl = fmt.Sprintf("http://%s", domain)
	}

	if port != 80 {
		rootUrl += fmt.Sprintf(":%d", port)
	}

	rootUrl += "/"

	return rootUrl
}
