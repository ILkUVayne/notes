package main

import "fmt"

type option func(s *server)

type server struct {
	protocol string
	host     string
	port     int
}

func newServer(opts ...option) *server {
	s := new(server)
	for _, v := range opts {
		v(s)
	}
	return s
}

func withProtocol(protocol string) option {
	return func(s *server) {
		s.protocol = protocol
	}
}

func withHost(host string) option {
	return func(s *server) {
		s.host = host
	}
}

func withPort(port int) option {
	return func(s *server) {
		s.port = port
	}
}

func (s *server) String() string {
	return fmt.Sprintf("%s:%s:%d", s.protocol, s.host, s.port)
}

// -------------------------------------- use options --------------------------------

func main() {
	s := newServer(
		withProtocol("tcp"),
		withHost("127.0.0.1"),
		withPort(8181),
	)
	fmt.Printf("%s\n", s)
	s2 := newServer(
		withProtocol("tcp"),
		withHost("127.0.1.1"),
	)
	fmt.Printf("%s\n", s2)
}
