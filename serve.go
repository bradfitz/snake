//go:build !wasm
// +build !wasm

package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

//go:embed index.html
var indexHTML []byte

//go:embed snake.wasm
var snakeWASM []byte

func main() {
	log.Printf("listening on :9090")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	http.HandleFunc("/snake.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		w.Write(snakeWASM)
	})
	http.HandleFunc("/wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(filepath.Join(runtime.GOROOT(), "misc/wasm/wasm_exec.js"))
		if err != nil {
			log.Print(err)
			http.Error(w, "can't find wasm_exec.js", 500)
			return
		}
		defer f.Close()
		var modTime time.Time
		if fi, err := f.Stat(); err == nil {
			modTime = fi.ModTime()
		}
		http.ServeContent(w, r, "wasm_exec.js", modTime, f)
	})
	http.Handle("/apple.png", redPNG)
	http.Handle("/white.png", whitePNG)
	http.Handle("/black.png", blackPNG)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal(err)
		return
	}
}

type lazyPNG struct {
	r, g, b uint8

	o   sync.Once
	png []byte // png bytes
}

func (p *lazyPNG) gen() {
	im := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	for i := 0; i < len(im.Pix); i += 4 {
		im.Pix[i+0] = p.r
		im.Pix[i+1] = p.g
		im.Pix[i+2] = p.b
		im.Pix[i+3] = 0xff
	}
	var buf bytes.Buffer
	png.Encode(&buf, im)
	p.png = buf.Bytes()
}

func (p *lazyPNG) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.o.Do(p.gen)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Expires", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
	w.Header().Set("Connection", "close")
	w.Write(p.png)
}

var (
	blackPNG = &lazyPNG{r: 0, g: 0, b: 0}
	redPNG   = &lazyPNG{r: 209, g: 8, b: 8}
	whitePNG = &lazyPNG{r: 255, g: 255, b: 255}
)
