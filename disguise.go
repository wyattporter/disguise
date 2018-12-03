// disguise - stripped-down 'atmos/camo' image proxy variant
//
// Description:
//
// An image proxy provides access to insecure (not accessible via SSL) images.
// This image proxy partially reimplements the API
// https://github.com/atmos/camo exposes.
package main
/*
    Copyright (C) 2018  Wyatt Porter

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published
    by the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

import "context"
import "crypto/hmac"
import "crypto/sha1"
import "encoding/hex"
import "errors"
import "flag"
import "io"
import "log"
import "net"
import "net/http"
import "os"
import "os/signal"
import "strings"
import "time"

const (
	digest int = iota
	url
	segments
)

var network = "tcp"
var address = "[::1]:8081"
var handler = http.Handler(disguise{})
var secret = []byte(os.Getenv("CAMO_KEY"))
var timeout = 10 * time.Second
var transferredHeaders = []string{"Via", "User-Agent", "Accept-Encoding"}

func init() {
	// Network type to use. Used by golang's "Listen" from package "net".
	// Defaults to "tcp".
	flag.StringVar(&network, "n", network, "connection type")
	// Address to listen on. Used by golang's "Listen" from package "net".
	// Defaults to "[::1]:8081".
	flag.StringVar(&address, "a", address, "connection listen address")
	// Shared secret used to compute HMAC authentication codes.
	// Defaults to value of CAMO_KEY environment variable.
	secret = []byte(*flag.String("s", string(secret), "shared secret"))
}

func main() {
	flag.Parse()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)

	d := disguise{
		Server: &http.Server{
			Addr:    address,
			Handler: handler,
		},
	}
	if err := d.Serve(s); err != nil {
		log.Fatalln("E:", err)
	}
}

type disguise struct{ *http.Server }

func (d disguise) Serve(s <-chan os.Signal) error {
	var err error
	var listener net.Listener

	if d.Server == nil {
		return errors.New("*http.Server nil")
	}

	if listener, err = net.Listen(network, address); err != nil {
		return err
	}

	go func(s *http.Server, sig <-chan os.Signal) {
		select {
		case <-sig:
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			s.Shutdown(ctx)
		}
	}(d.Server, s)

	if err = d.Server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (d disguise) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	var wrote int
	var request *http.Request
	var response *http.Response

	if r.Method != http.MethodGet {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	paths := strings.SplitN(strings.TrimLeft(r.URL.Path, "/"), "/", segments)
	if len(paths) != segments {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	decoded := make([][]byte, segments)
	for i := 0; i < segments; i++ {
		if decoded[i], err = hex.DecodeString(paths[i]); err != nil {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
	}

	mac := hmac.New(sha1.New, secret)
	if wrote, err = mac.Write(decoded[url]); wrote < len(decoded[url]) || err != nil {
		goto E500
	}
	if !hmac.Equal(mac.Sum(nil), decoded[digest]) {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	if request, err = http.NewRequest(http.MethodGet, string(decoded[url]), nil); err != nil {
		goto E500
	}
	for _, k := range transferredHeaders {
		request.Header.Set(k, r.Header.Get(k))
	}
	request.Header.Set("Accept", "image/*")

	log.Println("I:", string(decoded[url]))
	if response, err = http.DefaultClient.Do(request); err != nil {
		goto E500
	}

	if cts := response.Header.Get("Content-Type"); !strings.HasPrefix(cts, "image/") {
		http.Error(w, cts, http.StatusNotAcceptable)
		return
	}

	for k, vs := range response.Header {
		w.Header().Del(k)
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if _, err = io.Copy(w, response.Body); err != nil {
		goto E500
	}
	return
E500:
	log.Println("W:", err)
	http.Error(w, "", http.StatusInternalServerError)
	return
}
