package gostpkg

import (
	"context"
	"log"

	"github.com/go-gost/core/chain"
	"github.com/go-gost/core/hop"
	socks5c "github.com/go-gost/x/connector/socks/v5"
	socks5h "github.com/go-gost/x/handler/socks/v5"
	"github.com/go-gost/x/listener/tcp"
)

// StartSocks5Chain 启动 socks5+chain 代理服务
func StartSocks5Chain(listenAddr, remoteAddr, username, password string) error {
	// 创建 socks5 v5 connector
	conn := socks5c.NewConnector(
		socks5c.AuthOption(username, password),
	)

	// 组装 chain
	n := chain.NewNode(remoteAddr, chain.WithConnector(conn))
	h := hop.NewHop(hop.WithNodes(n))
	c := chain.NewChain(chain.WithHops(h))

	// 创建 socks5 handler（本地认证）
	hdl := socks5h.NewHandler(
		socks5h.AuthOption(username, password),
		socks5h.ChainOption(c),
	)

	// 创建 TCP listener
	ln, err := tcp.NewListener(listenAddr)
	if err != nil {
		return err
	}

	// 启动服务
	go func() {
		if err := hdl.Serve(context.Background(), ln); err != nil {
			log.Fatalf("socks5 服务启动失败: %v", err)
		}
	}()
	return nil
}
