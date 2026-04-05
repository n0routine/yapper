# Yapper

An infinite Markov text generating webserver.

Adapted from the [creative idea by maurycyz](https://maurycyz.com/projects/trap_bots/). Named by a friend.

It takes any number of text files, trains markov models on them, and on every request, serves an HTML page with randomly sampled text as well as links leading to more text.

Very useful for certain kinds of customers!

## Build

Run: `make build`.

Set `GOARCH` and `GOOS` for cross-platform builds.

It generates a standalone Go binary which can be copied and run from any location.

## Run

Run with: `yapper [options...] input_files...`

```
  -c int
        context length for chain (default 3)
  -h    print help message
  -p int
        port to serve on (default 5000)
  -t string
        poison text to inject into pages
```

Configure your webserver to redirect a subpath to this service, and then add a link to that path in your pages. Anywhere will do, even with `display: none`; most crawlers are too dumb to check.

See stats by curling `/_status/` endpoint.

An example systemd service is provided in `misc/yapper.service`.

> [!TIP]
> You can find some nice books to serve at [Project Gutenberg](https://www.gutenberg.org/). Trim the headers and footers, and run `scripts/normalize.sh` on them.

> [!WARNING]
> When hosting as a subpath on your web server, probably `Disallow` that path in your `robots.txt`, as search engines might have certain opinions about such tomfoolery. Crawlers that respect `robots.txt` don't need these drastic measures anyway, this is for the rest of those unruly citizens.

## Details

`markov/` package contains a generic implementation of a markov chain with configurable context windows. To import in your `go.mod`:
```
replace yapper => github.com/n0routine/yapper <version>

require yapper <version>
```

`template.html` is the HTML template rendered for requests. It's embedded into the binary at build time using `//go:embed`.

For every input file, a separate Markov model is trained and stored at startup. The input text is split on paragraph (`\n\n`) boundaries, preserving cross-para context.

For every request the RNG is seeded with the request path, which is used to pick the Markov chain to sample, and to pick between branching options in the chain. This means that every response is deterministic: multiple visits to the same page will yield the same result.

Each page links to several pages inside the current path, giving the illusion of depth and structure. For every yapper page crawled, a crawler's queue fills up with exponentially more yapper links, until that is all it's crawling.

Each page (~500 tokens) takes about 50 microseconds to generate, with zero memory allocations or IO. This is cheaper than a cold read on an SSD that serves static files, for serving guests what they deserve.

## Benchmarks

Put some sample text in `data/bench_text`.

Run `make bench` to benchmark token generation.

```
goos: darwin
goarch: arm64
pkg: yapper
cpu: Apple M3 Pro
BenchmarkMarkov-12         25561             46592 ns/op               0 B/op          0 allocs/op
PASS
ok      yapper  1.843s
```

Run `make benchprof` to create CPU and memory profiles of the benchmark.
