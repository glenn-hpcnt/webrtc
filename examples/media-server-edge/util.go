package main

import (
	"errors"
	"fmt"
	"net"
)

func GetIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return ""
		}
		if (i.Flags & net.FlagLoopback) != 0 {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			return ip.String()
		}
	}
	return ""
}

const (
	GET = iota
	RELEASE
)

type generator struct {
	usedPortMap map[int]*Connections
	portMin     int
	portMax     int
	reqCh       chan request
}

type request struct {
	kind  int
	port  int
	resCh chan *Connections
}

type Connections struct {
	Port          int
	audioUdpConn  *net.UDPConn
	videoUdpConn  *net.UDPConn
	video2UdpConn *net.UDPConn
}

var gen generator

func init() {
	gen = generator{
		usedPortMap: make(map[int]*Connections),
		reqCh:       make(chan request),
		portMin:     45000,
		portMax:     50000,
	}
	go gen.run()
}

func (p *generator) run() {
	// audio, video, video2 port를 각각 2 간격으로 할당한다.
	if p.portMin >= p.portMax-6 || p.portMax >= 65535 {
		panic("port cap is invalid")
	}
	currentPort := p.portMin
	for req := range gen.reqCh {
		switch req.kind {
		case GET:
			conns := &Connections{
				audioUdpConn:  nil,
				videoUdpConn:  nil,
				video2UdpConn: nil,
			}
			for {
				if _, prs := p.usedPortMap[currentPort]; !prs && currentPort < p.portMax-6 {
					if ac, vc, vc2, err := tryGetConnection(currentPort); err != nil {
						fmt.Println("failed to create udp socket, err: ", err.Error())
						currentPort += 6
					} else {
						conns.Port = currentPort
						conns.audioUdpConn = ac
						conns.videoUdpConn = vc
						conns.video2UdpConn = vc2
						p.usedPortMap[currentPort] = conns
						break
					}
				} else if currentPort < p.portMax-6 {
					currentPort += 6
				} else {
					fmt.Println("all port already used")
					currentPort = p.portMin
					conns = nil
					break
				}
			}
			req.resCh <- conns
		case RELEASE:
			if conns, prs := p.usedPortMap[req.port]; prs {
				conns.audioUdpConn.Close()
				conns.videoUdpConn.Close()
				conns.video2UdpConn.Close()
				delete(p.usedPortMap, req.port)
			}

		default:
			fmt.Println("unexpected port request is arrived")
		}
	}
}

func tryGetConnection(port int) (*net.UDPConn, *net.UDPConn, *net.UDPConn, error) {
	audioConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: port})
	if err != nil {
		return nil, nil, nil, err
	}
	videoConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: port + 2})
	if err != nil {
		return nil, nil, nil, err
	}
	videoConn2, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: port + 4})
	if err != nil {
		return nil, nil, nil, err
	}
	return audioConn, videoConn, videoConn2, nil
}

func GetConnections() (*Connections, error) {
	r := make(chan *Connections)
	gen.reqCh <- request{
		kind:  GET,
		resCh: r,
	}
	connections := <-r
	if connections == nil {
		return nil, errors.New("can't getting port")
	}
	return connections, nil
}

func ReleaseConnections(port int) {
	gen.reqCh <- request{
		kind: RELEASE,
		port: port,
	}
}
