package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
)

const (
	crlf       = "\r\n"
	colonspace = ": "
)

func GetSortedHeaderKeys(headerMap http.Header) []string {
	//Pre-allocate memory for all of the keys
	keys := make([]string, len(headerMap))[:0]
	for key := range headerMap {
		keys = append(keys, key)
	}
	sort.Sort(sort.StringSlice(keys))
	return keys
}

func ChecksumMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer h.ServeHTTP(w, r)
		// How do I get the response status code, content body, and headers?
		// At this point they haven't been written
		// I want to hook in at the end of the chain
		// What if I create a fake responsewriter to capture the body?
		fakeResponseWriter := httptest.NewRecorder()
		h.ServeHTTP(fakeResponseWriter, r)
		// now create the canonical response string
		canonicalResponse := sha1.New()
		canonicalResponse.Write([]byte(strconv.Itoa(fakeResponseWriter.Code) + crlf))
		checksumHeaders := GetSortedHeaderKeys(fakeResponseWriter.HeaderMap)
		xChecksumHeaders := strings.Join(checksumHeaders, ";")
		for _, key := range checksumHeaders {
			canonicalResponse.Write([]byte(key + colonspace + fakeResponseWriter.HeaderMap.Get(key) + crlf))
		}
		canonicalResponse.Write([]byte("X-Checksum-Headers" + colonspace + xChecksumHeaders + crlf + crlf))
		canonicalResponse.Write(fakeResponseWriter.Body.Bytes())
		w.Header().Set("X-Checksum-Headers", xChecksumHeaders)

		// generate the SHA-1 hash
		hash := canonicalResponse.Sum(nil)
		w.Header().Set("X-Checksum", hex.EncodeToString(hash[:]))
		w.WriteHeader(fakeResponseWriter.Code)
	})
}

// Do not change this function.
func main() {
	var listenAddr = flag.String("http", ":8080", "address to listen on for HTTP")
	flag.Parse()

	http.Handle("/", ChecksumMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Foo", "bar")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Date", "Sun, 08 May 2016 14:04:53 GMT")
		msg := "Curiosity is insubordination in its purest form.\n"
		w.Header().Set("Content-Length", strconv.Itoa(len(msg)))
		fmt.Fprintf(w, msg)
	})))

	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
