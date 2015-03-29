package utils

import (
	"fmt"
)

func BuildRootUrl(protocol string, domain string, port int, tls bool) string {
	var rootUrl string
	if protocol != "" {
		rootUrl = fmt.Sprint(protocol, "://", domain)
	} else {
		if tls {
			rootUrl = fmt.Sprintf("https://%s", domain)
		} else {
			rootUrl = fmt.Sprintf("http://%s", domain)
		}
	}

	if port != 80 {
		rootUrl += fmt.Sprintf(":%d", port)
	}

	rootUrl += "/"

	return rootUrl
}
