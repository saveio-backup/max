// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autocert_test

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmW7VUmSvhvSGbYbdsh7uRjhGmsYkc9fL8aJ5CorxxrU5N/go-crypto/acme/autocert"
)

func ExampleNewListener() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, TLS user! Your config: %+v", r.TLS)
	})
	log.Fatal(http.Serve(autocert.NewListener("example.com"), mux))
}

func ExampleManager() {
	m := &autocert.Manager{
		Cache:      autocert.DirCache("secret-dir"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("example.org"),
	}
	go http.ListenAndServe(":http", m.HTTPHandler(nil))
	s := &http.Server{
		Addr:      ":https",
		TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
	}
	s.ListenAndServeTLS("", "")
}
