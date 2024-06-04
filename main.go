package main

import (
	"fmt"
	"log"
	"net"
)

type Message struct {
	from    string
	payload []byte
}

type Server struct {
	listenAddr string
	ln         net.Listener
	connList   []net.Conn
	quitch     chan struct{}
	msgCh      chan Message
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgCh:      make(chan Message, 10),
		connList:   make([]net.Conn, 0),
	}
}
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln
	go s.acceptLoop()
	<-s.quitch
	close(s.msgCh)
	return nil
}
func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}
		fmt.Println("accept conn:", conn.RemoteAddr())
		s.connList = append(s.connList, conn)
		go s.readLoop(conn)
	}
}
func (s *Server) readLoop(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 2028)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				for i, connL := range s.connList {
					if connL == conn {
						s.connList = append(s.connList[:i], s.connList[i+1:]...)
						break
					}
				}
				for _, connL := range s.connList {
					connL.Write([]byte("conn " + conn.RemoteAddr().String() + " closed\n"))
				}
				return
			}
			fmt.Println("read error:", err)
			continue
		}

		s.msgCh <- Message{
			from:    conn.RemoteAddr().String(),
			payload: buf[:n],
		}
		for _, connL := range s.connList {
			if connL != conn {
				connL.Write([]byte("message from " + conn.RemoteAddr().String() + "\n" + string(buf[:n])))
			}
		}
		conn.Write([]byte("thanks for your message\n"))
	}
}
func main() {
	server := NewServer(":8080")
	go func() {
		for msg := range server.msgCh {
			fmt.Printf("msg from %s: %s\n", msg.from, string(msg.payload))
		}
	}()
	log.Fatal(server.Start())
}
