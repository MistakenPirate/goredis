//go:build darwin

package server

import (
	"log"
	"net"
	"syscall"
	"time"

	"github.com/mistakenpirate/goredis/config"
	"github.com/mistakenpirate/goredis/core"
)

func RunAsyncTCPServer() error {
	log.Println("starting an asynchronous TCP server on", config.Host, config.Port)

	max_clients := 20000

	// create a socket
	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(serverFD)

	// set the socket to operate in a non-blocking mode
	if err = syscall.SetNonblock(serverFD, true); err != nil {
		return err
	}

	// bind the IP and the port
	ip4 := net.ParseIP(config.Host)
	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		return err
	}

	// start listening
	if err = syscall.Listen(serverFD, max_clients); err != nil {
		return err
	}

	// AsyncIO starts here!!

	// creating kqueue instance
	kqFD, err := syscall.Kqueue()
	if err != nil {
		log.Fatal(err)
	}
	defer syscall.Close(kqFD)

	// register the server socket for read events
	serverEvent := syscall.Kevent_t{
		Ident:  uint64(serverFD),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
	}
	if _, err = syscall.Kevent(kqFD, []syscall.Kevent_t{serverEvent}, nil, nil); err != nil {
		return err
	}

	events := make([]syscall.Kevent_t, max_clients)

	for {
		if time.Now().After(lastCronExecTime.Add(cronFrequency)) {
			core.DeleteExpiredKeys()
			lastCronExecTime = time.Now()
		}

		// wait for events
		nevents, err := syscall.Kevent(kqFD, nil, events, nil)
		if err != nil {
			continue
		}

		for i := 0; i < nevents; i++ {
			fd := int(events[i].Ident)

			// if the socket server itself is ready for an IO
			if fd == serverFD {
				// accept the incoming connection from a client
				clientFD, _, err := syscall.Accept(serverFD)
				if err != nil {
					log.Println("err", err)
					continue
				}

				// increase the number of concurrent clients count
				con_clients++
				syscall.SetNonblock(clientFD, true)

				// add this new TCP connection to be monitored
				clientEvent := syscall.Kevent_t{
					Ident:  uint64(clientFD),
					Filter: syscall.EVFILT_READ,
					Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
				}
				if _, err := syscall.Kevent(kqFD, []syscall.Kevent_t{clientEvent}, nil, nil); err != nil {
					log.Fatal(err)
				}
			} else {
				comm := core.FDComm{Fd: fd}
				cmds, err := readCommands(comm)
				if err != nil {
					syscall.Close(fd)
					con_clients -= 1
					continue
				}
				respond(cmds, comm)
			}
		}
	}
}
