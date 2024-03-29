package websocket

import (
	"net/url"
	"testing"

	ma "github.com/multiformats/go-multiaddr"
)

func TestMultiaddrParsing(t *testing.T) {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/5555/wss")
	if err != nil {
		t.Fatal(err)
	}

	wsaddr, err := parseMultiaddr(addr)
	if err != nil {
		t.Fatal(err)
	}
	if wsaddr != "wss://127.0.0.1:5555" {
		t.Fatalf("expected wss://127.0.0.1:5555, got %s", wsaddr)
	}
}

type httpAddr struct {
	*url.URL
}

func (addr *httpAddr) Network() string {
	return "http"
}

func TestParseWebsocketNetAddr(t *testing.T) {
	notWs := &httpAddr{&url.URL{Host: "http://127.0.0.1:1234"}}
	_, err := ParseWebsocketNetAddr(notWs)
	if err.Error() != "not a websocket address" {
		t.Fatalf("expect \"not a websocket address\", got \"%s\"", err)
	}

	wsAddr := NewAddr("127.0.0.1:5555")
	parsed, err := ParseWebsocketNetAddr(wsAddr)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.String() != "/ip4/127.0.0.1/tcp/5555/wss" {
		t.Fatalf("expected \"/ip4/127.0.0.1/tcp/5555/wss\", got \"%s\"", parsed.String())
	}
}

func TestConvertWebsocketMultiaddrToNetAddr(t *testing.T) {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/5555/wss")
	if err != nil {
		t.Fatal(err)
	}

	wsaddr, err := ConvertWebsocketMultiaddrToNetAddr(addr)
	if err != nil {
		t.Fatal(err)
	}
	if wsaddr.String() != "//127.0.0.1:5555" {
		t.Fatalf("expected //127.0.0.1:5555, got %s", wsaddr)
	}
	if wsaddr.Network() != "websocket-tls" {
		t.Fatalf("expected network: \"websocket-tls\", got \"%s\"", wsaddr.Network())
	}
}
