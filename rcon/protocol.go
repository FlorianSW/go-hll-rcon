package rcon

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	MSGLEN = 8196
)

var (
	CommandFailed          = errors.New("FAIL")
	ReconnectTriesExceeded = errors.New("there are no reconnects left")
)

type socket struct {
	con            net.Conn
	key            []byte
	pw             string
	host           string
	port           int
	reconnectCount int
}

func makeConnection(h string, p int) (net.Conn, []byte, error) {
	con, err := net.DialTimeout("tcp4", fmt.Sprintf("%s:%d", h, p), 5*time.Second)
	if err != nil {
		return nil, nil, err
	}
	xorKey := make([]byte, MSGLEN)
	_, err = con.Read(xorKey)
	if err != nil {
		return nil, nil, err
	}
	xorKey = bytes.Trim(xorKey, "\x00")

	return con, xorKey, err
}

func newSocket(h string, p int, pw string) (*socket, error) {
	r := &socket{
		pw:             pw,
		host:           h,
		port:           p,
		reconnectCount: 0,
	}
	return r, r.reconnect(nil)
}

func (r *socket) Close() error {
	return r.con.Close()
}

func (r *socket) login() error {
	login := r.xor([]byte(fmt.Sprintf("login %s", r.pw)))
	_, err := r.con.Write(login)
	if err != nil {
		return err
	}

	res, err := r.read()
	if err != nil {
		return err
	}
	if string(res) == "FAIL" {
		return CommandFailed
	}
	return nil
}

func (r *socket) listCommand(cmd string) ([]string, error) {
	err := r.write(cmd)
	if err != nil {
		return nil, err
	}
	a, err := r.read()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(a), "\t")
	l, err := strconv.Atoi(lines[0])
	if err != nil {
		return nil, err
	}
	for {
		if strings.Count(string(a), "\t")-1 >= l {
			break
		}
		var ir []byte
		ir, err = r.read()
		if err != nil {
			panic(err)
		}
		a = append(a, ir...)
	}
	lines = strings.Split(string(a), "\t")[1:]
	var res []string
	for _, line := range lines {
		if line != "" {
			res = append(res, line)
		}
	}
	return res, nil
}

func (r *socket) command(cmd string) (string, error) {
	err := r.write(cmd)
	if err != nil {
		return "", err
	}
	a, err := r.read()
	if err != nil {
		return "", err
	}
	if string(a) == "FAIL" {
		return "", CommandFailed
	}
	return string(a), nil
}

func (r *socket) write(cmd string) error {
	_, err := r.con.Write(r.xor([]byte(cmd)))
	if errors.Is(err, syscall.EPIPE) {
		err = r.reconnect(err)
		if err != nil {
			return err
		}
		return r.write(cmd)
	}
	if err != nil {
		r.resetReconnectCount()
	}
	return err
}

func (r *socket) reconnect(orig error) error {
	if r.reconnectCount > 3 {
		return ReconnectTriesExceeded
	}
	con, xorKey, err := makeConnection(r.host, r.port)
	r.con = con
	r.key = xorKey
	if err != nil {
		return fmt.Errorf("reconnect failed: %s, original error: %w", err.Error(), orig)
	}
	r.reconnectCount++
	err = r.login()
	if err != nil {
		return fmt.Errorf("reconnect failed: %s, original error: %w", err.Error(), orig)
	}
	return nil
}

func (r *socket) read() ([]byte, error) {
	var answer []byte
	for {
		rb := make([]byte, MSGLEN)
		l, err := r.con.Read(rb)
		if errors.Is(err, syscall.ECONNRESET) {
			err = r.reconnect(err)
			if err != nil {
				return nil, err
			}
			l, err = r.con.Read(rb)
		}
		if err != nil {
			return nil, err
		} else {
			r.resetReconnectCount()
		}
		rb = rb[:l]

		answer = append(answer, r.xor(rb)...)

		if len(rb) >= MSGLEN {
			continue
		}
		break
	}

	return answer, nil
}

func (r *socket) xor(b []byte) []byte {
	var msg []byte
	for i := range b {
		mb := b[i] ^ r.key[i%len(r.key)]
		msg = append(msg, mb)
	}
	return msg
}

func (r *socket) resetReconnectCount() {
	if r.reconnectCount != 0 {
		r.reconnectCount = 0
	}
}
