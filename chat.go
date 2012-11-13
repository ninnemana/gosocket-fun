package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"

	"code.google.com/p/go.net/websocket"
)

var rootTemplate = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8" />
<script>

var input, output, websocket;

function showMessage(m) {
	var p = document.createElement("p");
	p.innerHTML = m;
	output.appendChild(p);
}

function onMessage(e) {
	showMessage(e.data);
}

function onClose() {
	showMessage("Connection closed.");
}

function sendMessage() {
	var m = input.value;
	input.value = "";
	websocket.send(m);
	showMessage(m);
}

function onKey(e) {
	if (e.keyCode == 13) {
		sendMessage();
	}
}

function init() {
	input = document.getElementById("input");
	input.addEventListener("keyup", onKey, false);

	output = document.getElementById("output");

	websocket = new WebSocket("ws://ninnemana.xen.prgmr.com/socket");
	websocket.onmessage = onMessage;
	websocket.onclose = onClose;
}

window.addEventListener("load", init, false);

</script>
</head>
<body>
<input id="input" type="text">
<div id="output"></div>
</body>
</html>
`))

const listenAddr = ":80"

func main() {
	go netListen()
	http.HandleFunc("/", rootHandler)
	http.Handle("/socket", websocket.Handler(socketHandler))
	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func netListen() {
	l, err := net.Listen("tcp", "ninnemana.xen.prgmr.com:4001")
	if err != nil {
		log.Fatal(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go match(c)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	rootTemplate.Execute(w, listenAddr)
}

type socket struct {
	io.ReadWriter
	done chan bool
}

func (s socket) Close() error {
	s.done <- true
	return nil
}

func socketHandler(ws *websocket.Conn) {
	s := socket{ws, make(chan bool)}
	go match(s)
	<-s.done
}

var partner = make(chan io.ReadWriteCloser)

func match(c io.ReadWriteCloser) {
	fmt.Fprint(c, "Waiting for a partner...")
	select {
	case partner <- c:
		// now handled by the other goroutine
	case p := <-partner:
		chat(p, c)
	}
}

func chat(a, b io.ReadWriteCloser) {
	fmt.Fprintln(a, "Found one! Say hi.")
	fmt.Fprintln(b, "Found one! Say hi.")
	errc := make(chan error, 1)
	go cp(a, b, errc)
	go cp(b, a, errc)
	if err := <-errc; err != nil {
		log.Println(err)
	}
	a.Close()
	b.Close()
}

func cp(w io.Writer, r io.Reader, errc chan<- error) {
	_, err := io.Copy(w, r)
	errc <- err
}
