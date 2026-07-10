// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-facter/facter authors

package facter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultHTTPGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			if r.Header.Get("Metadata") != "true" {
				t.Errorf("header not propagated: %q", r.Header.Get("Metadata"))
			}
			_, _ = w.Write([]byte("body\n"))
		case "/truncated":
			// Promise more bytes than we send, then drop the connection, so the
			// client's body read fails mid-stream.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server does not support hijack")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nshort"))
			_ = conn.Close()
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	if body, ok := defaultHTTPGet(srv.URL+"/ok", map[string]string{"Metadata": "true"}); !ok || body != "body\n" {
		t.Fatalf("ok GET = %q %v", body, ok)
	}
	if _, ok := defaultHTTPGet(srv.URL+"/missing", nil); ok {
		t.Fatal("404 should report unreachable")
	}
	if _, ok := defaultHTTPGet(srv.URL+"/truncated", nil); ok {
		t.Fatal("truncated body read should report unreachable")
	}
	// A malformed request URL fails at construction.
	if _, ok := defaultHTTPGet("http://\x7f/bad", nil); ok {
		t.Fatal("malformed URL should fail")
	}
	// A closed server yields a transport error.
	srv.Close()
	if _, ok := defaultHTTPGet(srv.URL+"/ok", nil); ok {
		t.Fatal("closed server should report unreachable")
	}
}

func TestDefaultEnvHTTPGet(t *testing.T) {
	if defaultEnv().httpGet == nil {
		t.Fatal("defaultEnv missing httpGet seam")
	}
}
