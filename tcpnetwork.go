package main

import (
	"bufio"
	"encoding/gob"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

// definig a data that needs to be sent via TCP
type complexinputData struct {
	Num int
	Str string
	M   map[string]int
	P   []byte
}

const Port = ":62000" //might not change the port number

// To open a TCP Connection
func OpenTcp(addr string) (*bufio.ReadWriter, error) {
	log.Println("Dial: " + addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)), nil
}

// To Handle incoming commands
type Handlefunction func(*bufio.ReadWriter)

type Endpoint struct {
	listener net.Listener
	handler  map[string]Handlefunction
	//mutex to maintain thread-safety
	m sync.RWMutex
}

func NewEndPoint() *Endpoint {

	return &Endpoint{
		handler: map[string]Handlefunction{},
	}
}

func (e *Endpoint) AddHandlefunction(name string, f Handlefunction) {
	e.m.Lock()
	e.handler[name] = f
	e.m.Unlock()
}

func (e *Endpoint) handleMessage(conn net.Conn) {

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer conn.Close()

	for {
		log.Print("Receive command '")
		cmd, err := rw.ReadString('\n')
		switch {
		case err == io.EOF:
			log.Println("End of File")
			return
		case err != nil:
			log.Println("Error in command")
			return
		}

		cmd = strings.Trim(cmd, "\n ")
		log.Print(cmd + "'")

		e.m.RLock()
		handlecommand, ok := e.handler[cmd]
		e.m.RUnlock()
		if !ok {
			log.Println("command is not registered")
			return
		}
		handlecommand(rw)

	}
}

func handleString(rw *bufio.ReadWriter) {
	log.Print("receive a string")
	s, err := rw.ReadString('\n')
	if err != nil {
		log.Println("Cannot read data from the connection", err)
	}
	s = strings.Trim(s, "\n ")
	log.Println(s)

	_, err = rw.WriteString("Got it \n")
	if err != nil {
		log.Println("Cannot write into the client connection")
	}
	err = rw.Flush()
	if err != nil {
		log.Println("flush failed", err)
	}

}

func handleGob(rw *bufio.ReadWriter) {
	log.Println("receive a gob data:")
	var data complexinputData

	dec := gob.NewDecoder(rw)
	err := dec.Decode(&data)
	if err != nil {
		log.Println("Error while decoding the data", err)
		return
	}
	log.Println(data)
	//log.Printf("complex input data: \n%#v\n ", data)
}

func (e *Endpoint) Listen() error {
	var err error
	e.listener, err = net.Listen("tcp", Port)
	if err != nil {
		log.Println("Failed accepting connection", err)
	}
	log.Println("listen ", e.listener.Addr().String())
	for {
		log.Println("Accepting a request")
		conn, err := e.listener.Accept()
		if err != nil {
			log.Println("failed to connect:", err)
		}
		log.Println("Handling the incoming messages")
		go e.handleMessage(conn)
	}
}

func client(ip string) error {

	testdata := complexinputData{
		Num: 10,
		Str: "String sample",
		P:   []byte("lol"),
		M:   map[string]int{"Messi": 10, "Neymar": 11, "Cristiano": 7},
	}

	rw, err := OpenTcp(ip + Port)
	if err != nil {
		return errors.Join(err, errors.New("couldnt connect to the ip address"))
	}

	log.Printf("Sending string request")
	n, err := rw.WriteString("STRING\n")

	log.Println(strconv.Itoa(n))
	if err != nil {
		return errors.Join(err, errors.New("Coudnt send the string request"))
	}
	n, err = rw.WriteString("This is a string data. This is the actual data that must be carried over. \n")
	if err != nil {
		return errors.Join(err, errors.New("coudnt send the string data"))
	}

	log.Println("flushing the buffer")
	err = rw.Flush()
	if err != nil {
		return errors.Join(err, errors.New("flush failed"))
	}

	log.Println("reading the reply")
	response, err := rw.ReadString('\n')
	if err != nil {
		return errors.Join(err, errors.New("error while reading the response from the server"))
	}

	log.Println("String response from the server: ", response)
	//need to code for the gob part

	log.Println("Complexinputdata", testdata)
	//this only works with the fields that start with Upper case letter. Ridiculous
	enc := gob.NewEncoder(rw)
	n, err = rw.WriteString("GOB\n")

	if err != nil {
		return errors.Join(err, errors.New("Trouble sending the string in connection"))
	}

	err = enc.Encode(testdata)
	if err != nil {
		return errors.Join(err, errors.New("Trouble encoding the struct data"))
	}

	err = rw.Flush()
	if err != nil {
		return errors.Join(err, errors.New("Flush failed"))
	}

	return nil
}

func server() error {
	endpoint := NewEndPoint()

	endpoint.AddHandlefunction("STRING", handleString)
	endpoint.AddHandlefunction("GOB", handleGob)

	return endpoint.Listen()
}

func main() {
	connect := flag.String("Connect", "", "IP Address.(If empty, goes into listen mode.")
	flag.Parse()

	if *connect != "" {
		err := client(*connect)
		if err != nil {
			log.Println("Error:", err)
		}
		log.Println("Client done.")
		return
	}
	err := server()
	if err != nil {
		log.Println("Error:", err)
	}

	log.Println("Server done.")
}

func init() {
	log.SetFlags(log.Lmicroseconds)
}
