//go:build windows

package server

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/mistakenpirate/goredis/config"
	"github.com/mistakenpirate/goredis/core"
)

func RunAsyncTCPServer() error {
	log.Println("starting an asynchronous TCP server on", config.Host, config.Port)

	lsnr, err := net.Listen("tcp", config.Host+":"+strconv.Itoa(config.Port))
	if err != nil {
		return err
	}
	defer lsnr.Close()

	// handle each connection in a goroutine (Go's net package uses IOCP on Windows)
	go func() {
		for {
			time.Sleep(cronFrequency)
			core.DeleteExpiredKeys()
			lastCronExecTime = time.Now()
		}
	}()

	for {
		c, err := lsnr.Accept()
		if err != nil {
			log.Println("err", err)
			continue
		}

		con_clients++

		go func(conn net.Conn) {
			defer func() {
				conn.Close()
				con_clients--
			}()

			for {
				cmd, err := readCommand(conn)
				if err != nil {
					return
				}
				respond(cmd, conn)
			}
		}(c)
	}
}
