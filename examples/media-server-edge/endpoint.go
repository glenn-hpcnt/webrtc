package main

import (
	"fmt"
	"math/rand"
	"net"
)

const (
	DefaultMountpointAudioPt = 96
	DefaultMountpointVideoPt = 97
)

var hClient = NewClient()

type EndPoint struct {
	Id               string
	forwarderId      string
	originAddr       string
	videoUdpConn     *net.UDPConn
	video2UdpConn    *net.UDPConn
	audioUdpConn     *net.UDPConn
	audioSessionChan chan *mediaChannel
	videoSessionChan chan *mediaChannel
	EndpointData
}

type mediaChannel struct {
	id string
	ch chan []byte
}

type mediaData struct {
	buf []byte
	len int
}

func CreateEndPoint(addr string, streamId string) (ep *EndPoint, err error) {
	conns, err := GetConnections()
	if err != nil {
		return nil, err
	}
	epData := EndpointData{
		Host:       GetIPv4(),
		Simulcast:  false,
		AudioPtype: DefaultMountpointAudioPt,
		VideoPtype: DefaultMountpointVideoPt,
		AudioPort:  uint16(conns.Port),
		VideoPort:  uint16(conns.Port + 2),
		VideoPort2: uint16(conns.Port + 4),
		AudioSsrc:  rand.Uint32(),
		VideoSsrc:  rand.Uint32(),
		VideoSsrc2: rand.Uint32(),
	}
	forwarderId, err := hClient.RequestForward(addr, streamId, epData)
	if err != nil {
		return nil, err
	}

	ep = &EndPoint{
		Id:               streamId,
		forwarderId:      forwarderId,
		originAddr:       addr,
		EndpointData:     epData,
		audioUdpConn:     conns.audioUdpConn,
		videoUdpConn:     conns.videoUdpConn,
		video2UdpConn:    conns.video2UdpConn,
		audioSessionChan: make(chan *mediaChannel, 128),
		videoSessionChan: make(chan *mediaChannel, 128),
	}
	go ep.run()
	return ep, nil
}

func (e *EndPoint) Close() {
	err := hClient.RequestStopForward(e.originAddr, e.Id, e.forwarderId)
	if err != nil {
		fmt.Println(err)
	}
	ReleaseConnections(int(e.EndpointData.AudioPort))
}

func (e *EndPoint) AttachSession(s *Session) error {
	e.audioSessionChan <- &mediaChannel{
		id: s.Id,
		ch: s.AudioChan,
	}
	e.videoSessionChan <- &mediaChannel{
		id: s.Id,
		ch: s.VideoChan,
	}
	return nil
}

func (e *EndPoint) DetachSession(id string) {
	e.audioSessionChan <- &mediaChannel{
		id: id,
		ch: nil,
	}
	e.videoSessionChan <- &mediaChannel{
		id: id,
		ch: nil,
	}
}

func (e *EndPoint) run() {
	audioChan := make(chan *mediaData, 128)
	go e.audioListener(audioChan)
	go e.audioSender(audioChan)

	videoChan := make(chan *mediaData, 128)
	go e.videoListener(videoChan)
	go e.videoListener2(videoChan)
	go e.videoSender(videoChan)
}

func (e *EndPoint) audioListener(audioChan chan<- *mediaData) {
	inboundRTPPacket := make([]byte, 4096)
	for {
		n, _, err := e.audioUdpConn.ReadFromUDP(inboundRTPPacket)
		if err != nil {
			fmt.Println("failed to read udp, close audio listener err: ", err)
			close(audioChan)
			break
		}
		data := make([]byte, n)
		copy(data, inboundRTPPacket[:n])
		audioChan <- &mediaData{
			buf: data,
			len: n,
		}
	}
}

func (e *EndPoint) audioSender(audioChan <-chan *mediaData) {
	m := make(map[string]chan []byte)
	for {
		select {
		case mData, ok := <-audioChan:
			if !ok {
				fmt.Println("audio duplex session closed: ", e.Id)
				break
			}
			data := make([]byte, mData.len)
			copy(data, mData.buf)
			for _, userCh := range m {
				userCh <- data
			}
		case mc := <-e.audioSessionChan:
			if mc.ch == nil {
				delete(m, mc.id)
			}
			if _, prs := m[mc.id]; prs {
				fmt.Println("already has audio duplex channel, id:", mc.id)
			} else {
				m[mc.id] = mc.ch
			}
		}
	}
}

func (e *EndPoint) videoSender(videoChan <-chan *mediaData) {
	m := make(map[string]chan []byte)
	for {
		select {
		case mData, ok := <-videoChan:
			if !ok {
				fmt.Println("video duplex channel closed: ", e.Id)
				break
			}
			data := make([]byte, mData.len)
			copy(data, mData.buf)
			for _, userCh := range m {
				userCh <- data
			}
		case mc := <-e.videoSessionChan:
			if mc.ch == nil {
				delete(m, mc.id)
			}
			if _, prs := m[mc.id]; prs {
				fmt.Println("already has video duplex channel, id:", mc.id)
			} else {
				m[mc.id] = mc.ch
			}
		}
	}
}

func (e *EndPoint) videoListener(videoChan chan<- *mediaData) {
	inboundRTPPacket := make([]byte, 4096)
	for {
		_, _, err := e.videoUdpConn.ReadFromUDP(inboundRTPPacket)
		if err != nil {
			fmt.Println("failed to read udp, close video listener err: ", err)
			break
		}
		//TODO: impl
	}
}

func (e *EndPoint) videoListener2(videoChan chan<- *mediaData) {
	inboundRTPPacket := make([]byte, 4096)
	for {
		n, _, err := e.video2UdpConn.ReadFromUDP(inboundRTPPacket)
		if err != nil {
			fmt.Println("failed to read udp, close video2 listener err: ", err)
			break
		}
		data := make([]byte, n)
		copy(data, inboundRTPPacket[:n])
		videoChan <- &mediaData{
			buf: data,
			len: n,
		}
	}
}
