package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"yapper/markov"
)

// CLI flags
var flagContext int
var flagPort int
var flagPoison string
var flagHelp bool

func init() {
	flag.IntVar(&flagContext, "c", 3, "context length for chain")
	flag.IntVar(&flagPort, "p", 5000, "port to serve on")
	flag.StringVar(&flagPoison, "t", "", "poison text to inject into pages")
	flag.BoolVar(&flagHelp, "h", false, "print help message")
}

//go:embed template.html
var tmpls string

// Output template
var tmpl = template.Must(template.New("template").Parse(tmpls))

var logger = log.New(os.Stdout, "", 0)

func trainBook(b []byte) (markov.MarkovChain[string, string], error) {
	// Record
	mk, err := markov.NewRecorder(flagContext, func(s string) string { return s }, StringHasher)
	if err != nil {
		return markov.MarkovChain[string, string]{}, err
	}
	// split into paragraphs for context break
	for iter := range bytes.SplitSeq(b, []byte("\n\n")) {
		sc := bufio.NewScanner(bytes.NewReader(iter))
		sc.Split(SplitWords)
		for sc.Scan() {
			text := sc.Text()
			mk.Push(text)
		}
		if err := sc.Err(); err != nil {
			return markov.MarkovChain[string, string]{}, err
		}
		mk.Flush()
	}
	return mk.Compile(), nil
}

func sampleBook(ms markov.MarkovSampler[string, string], rnd rand.Source, wc int) ([]byte, error) {
	ms.Flush()
	// Initialize to approx size
	const approxWordBytes = 4
	buf := make([]byte, 0, wc*approxWordBytes)
	first := true
	prevOpening := false
	for i := 0; i < wc; {
		next, err := ms.Next(rnd)
		if err == io.EOF {
			ms.Flush()
			continue
		}
		if err != nil {
			return []byte{}, err
		}
		nextr := []rune(next)
		if !first && !prevOpening && !isClosingPunct(nextr[0]) {
			buf = append(buf, " "...)
		}
		buf = append(buf, next...)
		prevOpening = isOpeningPunct(nextr[len(nextr)-1])
		first = false
		i += 1
	}
	return buf, nil
}

func main() {

	flag.Parse()

	if flagHelp {
		fmt.Fprintf(os.Stderr, "Yapper: An infinite Markov text generating webserver\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "github.com/n0routine/yapper/\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Run with: %v [options...] input_files...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n")
		flag.PrintDefaults()
		os.Exit(0)
	}
	if flagContext < 1 {
		panic("context length should be >= 1")
	}
	if flagPort < 0 {
		panic("port should be valid")
	}

	// Build library
	chains := [](markov.MarkovChain[string, string]){}
	for _, filen := range flag.Args() {
		logger.Printf("opening book: %v", filen)
		file, err := os.Open(filen)
		if err != nil {
			panic(err)
		}
		filec, err := io.ReadAll(file)
		if err != nil {
			panic(err)
		}
		ms, err := trainBook(filec)
		if err != nil {
			panic(err)
		}
		chains = append(chains, ms)
	}

	// rand pool
	rndPool := sync.Pool{
		New: func() any {
			return rand.New(rand.NewSource(0))
		},
	}

	// Stat counters
	statRequests := 0
	statDuration := time.Duration(0)
	statTokens := 0
	statBytes := 0

	// Serve customers
	handler := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		paths := r.URL.Path
		// Seed rand from path
		rnd := rndPool.Get().(*rand.Rand)
		seed := Int64FromBytes([]byte(paths))
		rnd.Seed(seed)
		// Sample page
		const tokensTitle, tokensPara, tokensRef, tokensRefl = 10, 200, 8, 1
		const numParas, numRefs = 3, 4
		reqTokens := 0
		msi := rnd.Intn(len(chains))
		sampler := chains[msi].NewSampler()
		title, err := sampleBook(sampler, rnd, tokensTitle)
		if err != nil {
			panic(err)
		}
		reqTokens += tokensTitle
		paras := make([][]byte, 0, numParas)
		for range numParas {
			para, err := sampleBook(sampler, rnd, tokensPara)
			if err != nil {
				panic(err)
			}
			paras = append(paras, para)
			reqTokens += tokensPara
		}
		refs, refls := make([][]byte, 0, numRefs), make([][]byte, 0, numRefs)
		for range numRefs {
			ref, err := sampleBook(sampler, rnd, tokensRef)
			if err != nil {
				panic(err)
			}
			refs = append(refs, ref)
			reqTokens += tokensRef
			refl, err := sampleBook(sampler, rnd, tokensRefl)
			if err != nil {
				panic(err)
			}
			refls = append(refls, refl)
			reqTokens += tokensRefl
		}
		rndPool.Put(rnd)
		// Write response
		wc := NewCountingWriter(w)
		w.Header().Add("Content-Type", "text/html")
		err = tmpl.Execute(wc, map[string]any{
			"title":  title,
			"para":   paras,
			"ref":    refs,
			"refl":   refls,
			"poison": flagPoison,
		})
		if err != nil {
			panic(err)
		}
		elapsed := time.Since(start)
		statRequests += 1
		statDuration += elapsed
		statTokens += reqTokens
		statBytes += wc.C
		logger.Printf("generated for %v in %v\n", paths, elapsed)
	}

	handlerStatus := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"served_requests": statRequests,
			"served_duration": statDuration,
			"served_tokens":   statTokens,
			"served_bytes":    statBytes,
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.HandleFunc("/_status/", handlerStatus)
	logger.Printf("serving on %v", flagPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", flagPort), mux)
	if err != nil {
		panic(err)
	}

}
