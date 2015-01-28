// Package rolling implements a 32 bit rolling checksum similar to rsync's
// algorithm taken from https://github.com/rwcarlsen/gobup.
package rolling

const blobSize = 1 << 13
const charOffset = 31 // Same value as Bup

type RollingSum struct {
	a      uint32
	b      uint32
	window []byte
	size   int
	i      int
}

func New(window int) *RollingSum {
	return &RollingSum{
		window: make([]byte, window),
		size:   window,
	}
}

func (rs *RollingSum) Write(data []byte) (n int, err error) {
	for _, c := range data {
		rs.WriteByte(c)
	}
	return len(data), nil
}

func (rs *RollingSum) WriteByte(c byte) error {
	rs.a += -uint32(rs.window[rs.i]) + uint32(c)
	rs.b += rs.a - uint32(rs.size)*uint32(rs.window[rs.i]+charOffset)
	rs.window[rs.i] = c
	rs.i = (rs.i + 1) % rs.size

	return nil
}

func (rs *RollingSum) OnSplit() bool {
	return (rs.b & (blobSize - 1)) == ((^0) & (blobSize - 1))
}

func (rs *RollingSum) Reset() {
	rs.window = make([]byte, rs.size)
	rs.a, rs.b = 0, 0
}
