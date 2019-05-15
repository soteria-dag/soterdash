package rand

import (
	"crypto/rand"
	"math/big"
	"net"
	"strings"
)

// RandInt returns a random number between 0 and max value.
// The zero is inclusive, and the max value is exclusive; randInt(3) returns values from 0 to 2.
func RandInt(max int) (int, error) {
	if max == 0 {
		return 0, nil
	}

	val, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return -1, err
	}
	return int(val.Int64()), nil
}

// RandString returns a randomly-generated string of the given length
func RandString(length int) (string, error) {
	runes := []rune(
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")

	var s strings.Builder
	for i := 0; i < length; i++ {
		choice, err := RandInt(len(runes))
		if err != nil {
			return "", err
		}
		s.WriteRune(runes[choice])
	}

	return s.String(), nil
}

// RandLoopAddr returns an available loopback address and TCP port
func RandLoopAddr() (string, error) {
	// When port 0 is specified, net.ListenTCP will automatically choose a port
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}

	defer ln.Close()

	return ln.Addr().String(), nil
}

// RandLoopPort returns an available TCP port on the loopback address
func RandLoopPort() (string, error) {
	addr, err := RandLoopAddr()
	if err != nil {
		return "", err
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	return port, nil
}