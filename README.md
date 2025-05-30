# Image Deduplication

This is a simple cli tool for finding and removing duplicate images written in pure Go. When I was back in college I put together a clunky python script to handle removing duplicates for my desktop wallpapers and from image classification datasets using dhash. Now I figured I'd make a more proper tool for it and draw from additional sources to enhance it while practicing Golang.

## Install

Binaries are available for each [release](https://github.com/alexgQQ/dedupe/releases). Download and use whichever for your system. For now this only contains binaries for Intel based Windows and Linux systems.

Alternatively you can install through go or build it from source. This requires Go version 1.23+.
```bash
go install github.com/alexgQQ/dedupe/cmd/dedupe@latest
# or
git clone https://github.com/alexgQQ/dedupe.git
cd dedupe
go build -ldflags '-s -w' -o dedupe cmd/dedupe/main.go
```

## Usage

Check the usage
```bash
dedupe --help
```

Or import this package and use it in your code
```bash
go get github.com/alexgQQ/dedupe@latest
```
and in your code
```golang
package main

import (
	"fmt"
	"strings"

	"github.com/alexgQQ/dedupe"
)

func main() {

	images := []string{
		"testimages/cat-on-couch.jpg",
		"testimages/cat.jpg",
		"testimages/copycat.jpg",
	}

	results, total, _ := dedupe.Duplicates(images, dedupe.DCT)
	fmt.Printf("Found %d duplicates\n", total)
	if total <= 0 {
		return
	}
	for _, files := range results {
		msg := strings.Join(files, ", ")
		fmt.Printf("%s\n", msg)
	}
}
```

## Development

It's a straightforward package so clone and use whatever go workflow you like. The cli at cmd/dedupe/main.go is the best entrypoint and I'd recommend to have the verbose flag set and point it at the test images in the repo. Configure your debugger to do that.
```bash
go run cmd/dedupe/main.go -v -t testimages
```

Keep it clean and tidy
```bash
go fmt ./...
go test -count 5 ./...
```

### Implementation

The process works by computing images perceptual hashes and using a vantage point tree to find hashes close to each other by their hamming distance. The hashing method has an impact and this currently implements the [dhash](https://www.hackerfactor.com/blog/index.php?/archives/529-Kind-of-Like-That.html) and [dct](https://github.com/alangshur/perceptual-dct-hash?tab=readme-ov-file#perceptual-hash-algorithm) perceptual hashes. Reasonable thresholds are defined from [here](https://phash.org/docs/design.html) and the dhash implementation description. By default the dct method is used as it is more accurate and resilient to image variation. However the dhash method is a bit faster and might be more appropriate for large amounts of images at the cost of some accuracy. For now this is sufficient but would be fun to implement more hashing methods like average hash or radial hash.

### Test Images

The testimages directory contains some images to test against. These are a collection of cat images and images from a wallpaper dump. In particular these images have variation of direct duplicates, recolorings, and similar looking images for a solid test case. We can observe the accuracy difference between dhash and dct against these.

### Building For Additional Targets

I find the [elastic golang crossbuild image](https://github.com/elastic/golang-crossbuild) to be very helpful. There's lots of documentation on how to use it but for now I mainly target linux and windows amd64 targets. At some point it would be good to do this for OSX targets but it's more of a hassle. Mac users should build from source. How to manually build:
```bash
GOVERSION=1.24.3
docker run \
  -v .:/go/src/github.com/alexgQQ/go-image-deduper \
  -w /go/src/github.com/alexgQQ/go-image-deduper \
  -e CGO_ENABLED=1 \
  docker.elastic.co/beats-dev/golang-crossbuild:${GOVERSION}-main \
  --build-cmd "go build -ldflags '-s -w' -o dedupe cmd/dedupe/main.go" \
  -p linux/amd64

docker run \
  -v .:/go/src/github.com/alexgQQ/go-image-deduper \
  -w /go/src/github.com/alexgQQ/go-image-deduper \
  -e CGO_ENABLED=1 \
  docker.elastic.co/beats-dev/golang-crossbuild:${GOVERSION}-main \
  --build-cmd "go build -ldflags '-s -w' -o dedupe.exe cmd/dedupe/main.go" \
  -p windows/amd64
```
