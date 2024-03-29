// Package websocket implements a websocket based transport for go-libp2p.
package websocket

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/transport"

	tptu "github.com/libp2p/go-libp2p-transport-upgrader"

	ws "github.com/gorilla/websocket"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multiaddr-fmt"
	"github.com/multiformats/go-multiaddr-net"
)

// WsProtocol is the multiaddr protocol definition for this transport.
var WssProtocol = ma.Protocol{
	Code:  498,
	Name:  "wss",
	VCode: ma.CodeToVarint(498),
}

// WsFmt is multiaddr formatter for WsProtocol
var WssFmt = mafmt.And(mafmt.TCP, mafmt.Base(WssProtocol.Code))

// WsCodec is the multiaddr-net codec definition for the websocket transport
var WssCodec = &manet.NetCodec{
	NetAddrNetworks:  []string{"websocket-tls"},
	ProtocolName:     "wss",
	ConvertMultiaddr: ConvertWebsocketMultiaddrToNetAddr,
	ParseNetAddr:     ParseWebsocketNetAddr,
}

// Default gorilla upgrader
var upgrader = ws.Upgrader{
	// Allow requests from *all* origins.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func init() {
	err := ma.AddProtocol(WssProtocol)
	if err != nil {
		panic(fmt.Errorf("error registering websocket protocol: %s", err))
	}

	manet.RegisterNetCodec(WssCodec)
}

// WebsocketTransport is the actual go-libp2p transport
type WebsocketTransport struct {
	Upgrader *tptu.Upgrader
}

func New(u *tptu.Upgrader) *WebsocketTransport {
	return &WebsocketTransport{u}
}

var _ transport.Transport = (*WebsocketTransport)(nil)

func (t *WebsocketTransport) CanDial(a ma.Multiaddr) bool {
	return WssFmt.Matches(a)
}

func (t *WebsocketTransport) Protocols() []int {
	return []int{WssProtocol.Code}
}

func (t *WebsocketTransport) Proxy() bool {
	return false
}

func (t *WebsocketTransport) maDial(ctx context.Context, raddr ma.Multiaddr) (manet.Conn, error) {
	wsurl, err := parseMultiaddr(raddr)
	if err != nil {
		return nil, err
	}

	wscon, _, err := ws.DefaultDialer.Dial(wsurl, nil)
	if err != nil {
		return nil, err
	}

	mnc, err := manet.WrapNetConn(NewConn(wscon))
	if err != nil {
		wscon.Close()
		return nil, err
	}
	return mnc, nil
}

func (t *WebsocketTransport) Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (transport.CapableConn, error) {
	macon, err := t.maDial(ctx, raddr)
	if err != nil {
		return nil, err
	}
	return t.Upgrader.UpgradeOutbound(ctx, t, macon, p)
}

func (t *WebsocketTransport) maListen(a ma.Multiaddr) (manet.Listener, error) {
	lnet, lnaddr, err := manet.DialArgs(a)
	if err != nil {
		return nil, err
	}

	nl, err := net.Listen(lnet, lnaddr)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("http://" + nl.Addr().String())
	if err != nil {
		nl.Close()
		return nil, err
	}

	malist, err := t.wrapListener(nl, u)
	if err != nil {
		nl.Close()
		return nil, err
	}

	go malist.serve()

	return malist, nil
}

func (t *WebsocketTransport) Listen(a ma.Multiaddr) (transport.Listener, error) {
	malist, err := t.maListen(a)
	if err != nil {
		return nil, err
	}
	return t.Upgrader.UpgradeListener(t, malist), nil
}

func (t *WebsocketTransport) wrapListener(l net.Listener, origin *url.URL) (*listener, error) {
	laddr, err := manet.FromNetAddr(l.Addr())
	if err != nil {
		return nil, err
	}
	wsma, err := ma.NewMultiaddr("/wss")
	if err != nil {
		return nil, err
	}
	laddr = laddr.Encapsulate(wsma)

	return &listener{
		laddr:    laddr,
		Listener: l,
		incoming: make(chan *Conn),
		closed:   make(chan struct{}),
	}, nil
}
