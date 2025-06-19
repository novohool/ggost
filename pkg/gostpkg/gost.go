package gostpkg

import (
	"context"
	"log"
	"net"
	"net/url"

	"github.com/go-gost/core/chain"
	"github.com/go-gost/core/handler"
	socks5c "github.com/go-gost/x/connector/socks/v5"
	socks5h "github.com/go-gost/x/handler/socks/v5"
	"github.com/go-gost/x/listener/tcp"
)

// StartSocks5Chain 启动 socks5+chain 代理服务（go-gost v3 正确用法）
func StartSocks5Chain(listenAddr, remoteAddr, username, password string) error {
	// 创建 socks5 connector
	conn := socks5c.NewConnector()

	// 组装 node
	n := chain.NewNode("proxy-node", remoteAddr)
	// 组装 hop
	h := &chain.Hop{Nodes: []*chain.Node{n}}
	// 组装 chain
	c := &chain.Chain{Hops: []*chain.Hop{h}}

	// 创建 socks5 handler
	hdl := socks5h.NewHandler(
		handler.AuthOption(url.UserPassword(username, password)),
		handler.RouterOption(c),
	)

	// 创建 TCP listener
	ln, err := tcp.NewListener(listenAddr)
	if err != nil {
		return err
	}

	// 启动服务
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("accept error: %v", err)
				continue
			}
			go func(c net.Conn) {
				defer c.Close()
				if err := hdl.Handle(context.Background(), c); err != nil {
					log.Printf("handle error: %v", err)
				}
			}(conn)
		}
	}()
	return nil
}
