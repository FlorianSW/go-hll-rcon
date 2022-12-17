package rcon

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	MSGLEN = 8196
)

var (
	CommandFailed = errors.New("FAIL")
)

type socket struct {
	con net.Conn
	key []byte
	pw  string
}

func newSocket(h string, p int, pw string) (*socket, error) {
	con, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", h, p))
	if err != nil {
		return nil, err
	}
	xorKey := make([]byte, MSGLEN)
	_, err = con.Read(xorKey)
	if err != nil {
		return nil, err
	}
	xorKey = bytes.Trim(xorKey, "\x00")
	r := &socket{
		con: con,
		key: xorKey,
		pw:  pw,
	}
	return r, r.login()
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
	return err
}

func (r *socket) read() ([]byte, error) {
	var answer []byte
	for {
		rb := make([]byte, MSGLEN)
		l, err := r.con.Read(rb)
		if err != nil {
			return nil, err
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
