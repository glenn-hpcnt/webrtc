package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
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
	videoChannelMap  sync.Map
	video2ChannelMap sync.Map
	audioChannelMap  sync.Map
	EndpointData
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
		Id:            streamId,
		forwarderId:   forwarderId,
		originAddr:    addr,
		EndpointData:  epData,
		audioUdpConn:  conns.audioUdpConn,
		videoUdpConn:  conns.videoUdpConn,
		video2UdpConn: conns.video2UdpConn,
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
	_, prs := e.audioChannelMap.LoadOrStore(s.Id, s.AudioChan)
	if prs {
		return errors.New("already has session")
	}
	_, prs = e.videoChannelMap.LoadOrStore(s.Id, s.VideoChan)
	if prs {
		return errors.New("already has session")
	}
	return nil
}

func (e *EndPoint) DetachSession(id string) {
	e.audioChannelMap.Delete(id)
	e.videoChannelMap.Delete(id)
}

func (e *EndPoint) run() {
	go e.audioListener()
	go e.videoListener()
	go e.videoListener2()
}

func (e *EndPoint) audioListener() {
	inboundRTPPacket := make([]byte, 4096)
	for {
		n, _, err := e.audioUdpConn.ReadFromUDP(inboundRTPPacket)
		if err != nil {
			fmt.Println("failed to read udp, close audio listener err: ", err)
			break
		}
		e.audioChannelMap.Range(func(id, audioChan interface{}) bool {
			data := make([]byte, n)
			copy(data, inboundRTPPacket[:n])
			audioChan.(chan []byte) <- data
			return true
		})
	}
}

func (e *EndPoint) videoListener() {
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

func (e *EndPoint) videoListener2() {
	inboundRTPPacket := make([]byte, 4096)
	for {
		n, _, err := e.video2UdpConn.ReadFromUDP(inboundRTPPacket)
		if err != nil {
			fmt.Println("failed to read udp, close video2 listener err: ", err)
			break
		}
		e.videoChannelMap.Range(func(id, videoChan interface{}) bool {
			data := make([]byte, n)
			copy(data, inboundRTPPacket[:n])
			videoChan.(chan []byte) <- data
			return true
		})
	}
}
