package mtree

import (
	"bufio"
	"io"
)

const posixPolynomial uint32 = 0x04C11DB7

// cksum is an implementation of the POSIX CRC algorithm
func cksum(r io.Reader) (uint32, int, error) {
	in := bufio.NewReader(r)
	count := 0
	var sum uint32
	f := func(b byte) {
		for i := 7; i >= 0; i-- {
			msb := sum & (1 << 31)
			sum = sum << 1
			if msb != 0 {
				sum = sum ^ posixPolynomial
			}
		}
		sum ^= uint32(b)
	}

	for done := false; !done; {
		switch b, err := in.ReadByte(); err {
		case io.EOF:
			done = true
		case nil:
			f(b)
			count++
		default:
			return ^sum, count, err
		}
	}
	for m := count; ; {
		f(byte(m) & 0xff)
		m = m >> 8
		if m == 0 {
			break
		}
	}
	f(0)
	f(0)
	f(0)
	f(0)
	return ^sum, count, nil
}
