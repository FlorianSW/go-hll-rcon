package shiftpath

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

func ShiftPath(req *http.Request) (string, *http.Request) {
	var head, tail string

	ctx := req.Context()
	p := path.Clean("/" + req.URL.EscapedPath())
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		head, tail = p[1:], "/"
	} else {
		head, tail = p[1:i], p[i:]
	}

	rOut := req.Clone(ctx)
	unescape, _ := url.PathUnescape(tail)
	rOut.URL.Path = unescape
	if rOut.URL.RawPath != "" {
		rOut.URL.RawPath = tail
	}
	return head, rOut
}
