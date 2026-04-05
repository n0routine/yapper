package markov

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"slices"
)

// Insert given elements at index, shifting elements up as needed and discarding overflow.
// Does not grow the slice.
func slicesInsertPop[S ~[]E, E any](s S, i int, v ...E) S {
	n, m := len(s), len(v)
	copy(s[i+m:], s[i:n-m])
	copy(s[i:], v)
	return s
}

type MarkovRecorder[T any, C comparable, CS comparable] struct {
	contextLength int
	hashFunc      func(T) C
	hashFuncSlice func([]T) CS
	records       map[CS](map[C]markovRecord[T])
	current       []T
}

type MarkovChain[T any, CS comparable] struct {
	contextLength int
	hashFuncSlice func([]T) CS
	records       map[CS](markovChainElem[T])
}

type markovChainElem[T any] struct {
	occurences int
	records    [](markovRecord[T])
}

type markovRecord[T any] struct {
	element    T
	occurences int
}

type MarkovSampler[T any, CS comparable] struct {
	chain   MarkovChain[T, CS]
	current []T
}

func NewRecorder[T comparable, C comparable, CS comparable](contextLength int, hashFunc func(T) C, hashFuncSlice func([]T) CS) (MarkovRecorder[T, C, CS], error) {
	if contextLength < 1 {
		return MarkovRecorder[T, C, CS]{}, errors.New("context length must be >= 1")
	}
	return MarkovRecorder[T, C, CS]{
		contextLength: contextLength,
		hashFunc:      hashFunc,
		hashFuncSlice: hashFuncSlice,
		records:       make(map[CS]map[C]markovRecord[T]),
		current:       make([]T, contextLength),
	}, nil
}

func (m *MarkovRecorder[T, C, CS]) Push(e T) {
	m.record(e)
	m.current = slicesInsertPop(m.current, 0, e)
}

func (m *MarkovRecorder[T, C, CS]) record(e T) {
	next, hash := m.hashFunc(e), m.hashFuncSlice(m.current)
	if _, exs := m.records[hash]; !exs {
		m.records[hash] = make(map[C]markovRecord[T])
	}
	occurences := m.records[hash][next].occurences
	occurences += 1
	m.records[hash][next] = markovRecord[T]{element: e, occurences: occurences}
}

func (m *MarkovRecorder[T, C, CS]) Flush() {
	clear(m.current)
}

func (m MarkovRecorder[T, C, CS]) Compile() MarkovChain[T, CS] {
	records := make(map[CS]markovChainElem[T])
	for hash, currRecord := range m.records {
		totalOccurences := 0
		occurenceRecords := make([]markovRecord[T], 0, 8)
		for _, recordElem := range currRecord {
			occurenceRecords = append(occurenceRecords, markovRecord[T]{element: recordElem.element, occurences: recordElem.occurences})
			totalOccurences += recordElem.occurences
		}
		slices.SortFunc(occurenceRecords, func(a markovRecord[T], b markovRecord[T]) int {
			return cmp.Compare(a.occurences, b.occurences)
		})
		records[hash] = markovChainElem[T]{occurences: totalOccurences, records: occurenceRecords}
	}
	return MarkovChain[T, CS]{
		contextLength: m.contextLength,
		hashFuncSlice: m.hashFuncSlice,
		records:       records,
	}
}

func (m MarkovChain[T, CS]) sample(slc []T, rnd rand.Source) (T, error) {
	hash := m.hashFuncSlice(slc)
	record, exs := m.records[hash]
	if !exs || record.occurences == 0 {
		return *new(T), io.EOF
	}
	trg := int(rnd.Int63()) % record.occurences
	occurenceCounter := 0
	for idx := range record.records {
		occurenceCounter += record.records[idx].occurences
		if occurenceCounter > trg {
			return record.records[idx].element, nil
		}
	}
	// Some record should have exceeded idx
	panic("impossible")
}

func (m MarkovChain[T, CS]) NewSampler() MarkovSampler[T, CS] {
	return MarkovSampler[T, CS]{
		chain:   m,
		current: make([]T, m.contextLength),
	}
}

func (m *MarkovSampler[T, CS]) Next(rnd rand.Source) (T, error) {
	next, err := m.chain.sample(m.current, rnd)
	if err != nil {
		return *new(T), err
	}
	m.current = slicesInsertPop(m.current, 0, next)
	return next, nil
}

func (m *MarkovSampler[T, CS]) Flush() {
	clear(m.current)
}

func Dump[T any, CS comparable](w io.Writer, m MarkovChain[T, CS]) {
	freq, options := map[int]int{}, map[int]int{}
	for hash, record := range m.records {
		for _, recordElem := range record.records {
			fmt.Fprintf(w, "[%v] [%v] [%v]\n", hash, recordElem.element, recordElem.occurences)
			freq[recordElem.occurences] += 1
		}
		options[len(record.records)] += 1
	}
	optionpnt, optioncnt := 0, 0
	for k, v := range options {
		if k == 1 {
			continue
		}
		optioncnt += k * v
	}
	optionpnt = len(options) - 1
	fmt.Fprintf(w, "%#v\n", freq)
	fmt.Fprintf(w, "%#v\n", options)
	fmt.Fprintf(w, "%#v %#v\n", optionpnt, optioncnt)
}
