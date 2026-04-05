package main

import (
	"bufio"
	"bytes"
	"io"
	"math/rand"
	"os"
	"testing"

	"yapper/markov"
)

func BenchmarkMarkov(b *testing.B) {

	f, _ := os.Open("./data/bench_text")
	fc, _ := io.ReadAll(f)
	f.Close()
	const cl = 3
	mk, err := markov.NewRecorder(cl, func(s string) string { return s }, StringHasherConst3)
	if err != nil {
		panic(err)
	}
	for iter := range bytes.SplitSeq(fc, []byte("\n\n")) {
		sc := bufio.NewScanner(bytes.NewReader(iter))
		sc.Split(SplitWords)
		for sc.Scan() {
			text := sc.Text()
			mk.Push(text)
			if text == "." {
				mk.Flush()
			}
		}
		if err := sc.Err(); err != nil {
			panic(err)
		}
		mk.Flush()
	}
	ms := mk.Compile().NewSampler()
	rnd := rand.New(rand.NewSource(0))

	const wc = 500
	for b.Loop() {
		ms.Flush()
		rnd.Seed(0)
		for range wc {
			_, err := ms.Next(rnd)
			if err == io.EOF {
				ms.Flush()
				continue
			}
			if err != nil {
				panic(err)
			}
		}
	}

}
