// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"net/http"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/language"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVcxhXDbXjNoAdmYBWbY1eU67kQ8eZUHjG4mAYZUtZZu3/go-text/message"
)

var matcher = language.NewMatcher(message.DefaultCatalog.Languages())

func Generize(w http.ResponseWriter, r *http.Request) {
	lang, _ := r.Cookie("lang")
	accept := r.Header.Get("Accept-Language")
	tag := message.MatchLanguage(lang.String(), accept)
	p := message.NewPrinter(tag)

	p.Fprintf(w, "Hello %s!\n", r.Header.Get("From"))

	p.Fprintf(w, "Do you like your browser (%s)?\n", r.Header.Get("User-Agent"))
}
