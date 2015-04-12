/*

Package chunker implements a chunker based on a rolling Rabin fingerprint to determine block boundaries.

Implementation similar to https://github.com/cschwede/python-rabin-fingerprint

*/
package chunker

type Chunker struct {
	cache   [256]uint64
	window  []byte
	pos     int
	prevPos int

	WindowSize uint64
	Prime      uint64

	Fingerprint uint64

	ChunkMinSize uint64
	ChunkAvgSize uint64
	ChunkMaxSize uint64

	BlockSize uint64
}

// Same window size as LBFS 48
var windowSize = 64

// TODO build the cache in init

func New() *Chunker {
	chunker := &Chunker{
		window:       make([]byte, windowSize),
		pos:          0,
		prevPos:      windowSize - 1,
		WindowSize:   uint64(windowSize),
		Prime:        31,
		ChunkMinSize: 256 * 1024,
		ChunkAvgSize: 1024 * 1024,
		ChunkMaxSize: 4 * 1024 * 1024,
	}
	// calculates result = Prime ^ WindowSize first
	result := uint64(1)
	for i := uint64(1); i < chunker.WindowSize; i++ {
		result *= chunker.Prime
	}
	// caches the result for all 256 bytes
	for i := uint64(0); i < 256; i++ {
		chunker.cache[i] = i * result
	}
	return chunker
}

func (chunker *Chunker) Write(data []byte) (n int, err error) {
	for _, c := range data {
		chunker.WriteByte(c)
	}
	return len(data), nil
}

func (chunker *Chunker) WriteByte(c byte) error {
	ch := uint64(c) + 1 // add 1 to prevent long sequences of 0
	chunker.Fingerprint *= chunker.Prime
	chunker.Fingerprint += ch
	//fmt.Printf("chunker=%+v/%+v/%+v\n", chunker.pos, len(chunker.window), chunker.prevPos)
	chunker.Fingerprint -= chunker.cache[chunker.window[chunker.prevPos]]

	chunker.window[chunker.pos] = c
	chunker.prevPos = chunker.pos
	chunker.pos = (chunker.pos + 1) % int(chunker.WindowSize)
	chunker.BlockSize++
	return nil
}

func (chunker *Chunker) OnSplit() bool {
	if chunker.BlockSize > chunker.ChunkMinSize {
		if chunker.Fingerprint%chunker.ChunkAvgSize == 1 || chunker.BlockSize >= chunker.ChunkMaxSize {
			return true
		}
	}
	return false
}

func (chunker *Chunker) Reset() {
	chunker.BlockSize = 0
}
