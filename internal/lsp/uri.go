package lsp

import (
	"net/url"
	"strings"
)

func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return strings.TrimPrefix(uri, "file://")
	}
	return u.Path
}

func pathToURI(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}
