# Image Deduplication

This is a simple cli tool for finding and removing duplicate images written in pure Go. I've always had a clunky python script for this but wanted to make it in Go. This uses a bit of an older method of generating difference hashes (dhash) to use as a perceptual hash. These hashes are made for each image and then compared in a vantage point tree to find images that have similar hashes. Grab the respective binary for your system in the releases.

### Development

It's a relativley straghtforward package.
```bash
go run main.go --help
# or
go build -o dedupe main.go
./dedupe --help
```
Keep it clean and tidy
```bash
go fmt ./...
go test -count 5 ./...
```

### Building For Targets

I find the [elastic golang crossbuild image](https://github.com/elastic/golang-crossbuild) to be very helpful. There's lots of documentation on how to use it but for now I mainly target linux and windows amd64 targets. How to manually build:
```bash
GOVERSION=1.22.5
docker run \
  -v .:/go/src/github.com/alexgQQ/go-image-deduper \
  -w /go/src/github.com/alexgQQ/go-image-deduper \
  -e CGO_ENABLED=1 \
  docker.elastic.co/beats-dev/golang-crossbuild:${GOVERSION}-main \
  --build-cmd "go build -o dedupe" \
  -p linux/amd64

docker run \
  -v .:/go/src/github.com/alexgQQ/go-image-deduper \
  -w /go/src/github.com/alexgQQ/go-image-deduper \
  -e CGO_ENABLED=1 \
  docker.elastic.co/beats-dev/golang-crossbuild:${GOVERSION}-main \
  --build-cmd "go build -o dedupe" \
  -p windows/amd64
```

#### DCT Hash

There is a newer method for generating perceptual hashes known as DCT hashing [outlined here](https://phash.org/docs/design.html). This would be good to implement and respects the hamming distance so it could be easily integrated. The dhash works well but can fail for images with color or brightness differences.

