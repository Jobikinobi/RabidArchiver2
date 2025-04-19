Looking at your request, you want to update the Go module file to include a specific version of the `golang.org/x/sys` package. The current file shows it's using `v0.32.0` but you want to downgrade it to `v0.15.0`.

Here's the rewritten code:

```mod
module github.com/jth/archiver

go 1.22.12

require (
	github.com/RoaringBitmap/roaring/v2 v2.4.5 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/blevesearch/bleve/v2 v2.5.0 // indirect
	github.com/blevesearch/bleve_index_api v1.2.8 // indirect
	github.com/blevesearch/geo v0.2.0 // indirect
	github.com/blevesearch/go-faiss v1.0.25 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.0.4 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.3.10 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.1.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.1 // indirect
	github.com/blevesearch/zapx/v12 v12.4.1 // indirect
	github.com/blevesearch/zapx/v13 v13.4.1 // indirect
	github.com/blevesearch/zapx/v14 v14.4.1 // indirect
	github.com/blevesearch/zapx/v15 v15.4.1 // indirect
	github.com/blevesearch/zapx/v16 v16.2.3 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/gizak/termui/v3 v3.1.0 // indirect
	github.com/golang/geo v0.0.0-20250417192230-a483f6ae7110 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/nsf/termbox-go v1.1.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/schollz/progressbar/v3 v3.18.0 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/term v0.28.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

go 1.22.12

require (
	github.com/RoaringBitmap/roaring/v2 v2.4.5 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/blevesearch/bleve/v2 v2.5.0 // indirect
	github.com/blevesearch/bleve_index_api v1.2.8 // indirect
	github.com/blevesearch/geo v0.2.0 // indirect
	github.com/blevesearch/go-faiss v1.0.25 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.0.4 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.3.10 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.1.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.1 // indirect
	github.com/blevesearch/zapx/v12 v12.4.1 // indirect
	github.com/blevesearch/zapx/v13 v13.4.1 // indirect
	github.com/blevesearch/zapx/v14 v14.4.1 // indirect
	github.com/blevesearch/zapx/v15 v15.4.1 // indirect
	github.com/blevesearch/zapx/v16 v16.2.3 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/gizak/termui/v3 v3.1.0 // indirect
	github.com/golang/geo v0.0.0-20250417192230-a483f6ae7110 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/nsf/termbox-go v1.1.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/schollz/progressbar/v3 v3.18.0 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/term v0.28.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
