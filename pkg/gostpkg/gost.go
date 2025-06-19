package gostpkg

import (
	"context"
	"log"

	"github.com/go-gost/core/chain"
	socks5c "github.com/go-gost/x/connector/socks/v5"
	socks5h "github.com/go-gost/x/handler/socks/v5"
	"github.com/go-gost/x/listener/tcp"
)

// StartSocks5Chain 启动 socks5+chain 代理服务（go-gost v3 正确用法）
func StartSocks5Chain(listenAddr, remoteAddr, username, password string) error {
	// 创建 socks5 connector
	conn := socks5c.NewConnector()

	// 组装 node
	n := &chain.Node{
		Addr:      remoteAddr,
		Connector: conn,
		Auth: &chain.Auth{
			Username: username,
			Password: password,
		},
	}
	// 组装 hop
	h := &chain.Hop{
		Nodes: []*chain.Node{n},
	}
	// 组装 chain
	c := &chain.Chain{
		Hops: []*chain.Hop{h},
	}

	// 创建 socks5 handler
	hdl := socks5h.NewHandler(&socks5h.Config{
		Auth: &socks5h.Auth{
			Username: username,
			Password: password,
		},
		Chain: c,
	})

	// 创建 TCP listener
	ln, err := tcp.NewListener(listenAddr)
	if err != nil {
		return err
	}

	// 启动服务
	go func() {
		if err := hdl.Serve(ln); err != nil {
			log.Fatalf("socks5 服务启动失败: %v", err)
		}
	}()
	return nil
}
