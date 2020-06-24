package main

import (
	"fmt"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/examples/internal/signal"
)

type Session struct {
	Id               string
	StreamId         string
	pc               *webrtc.PeerConnection
	videoSenderTrack *webrtc.Track
	audioSenderTrack *webrtc.Track
	VideoChan        chan []byte
	AudioChan        chan []byte
}

func CreateSession(viewId, streamId string) (*Session, string) {
	s := Session{
		Id:        viewId,
		StreamId:  streamId,
		VideoChan: make(chan []byte, 128),
		AudioChan: make(chan []byte, 128),
	}
	mediaEngine := webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	settingEngine := webrtc.SettingEngine{}
	//settingEngine.SetTrickle(true) // 실제로 받는 candidate가 없을 것임.
	settingEngine.SetLite(true)
	settingEngine.SetEphemeralUDPPortRange(20000, 25000)
	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine), webrtc.WithSettingEngine(settingEngine))
	peerConnectionConfig := webrtc.Configuration{}

	peerConnection, err := api.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		panic(err)
	}

	vt, err := peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo)
	if err != nil {
		panic(err)
	}
	s.videoSenderTrack = vt.Sender().Track()
	at, err := peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio)
	if err != nil {
		panic(err)
	}
	s.audioSenderTrack = at.Sender().Track()

	sd, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	if err := peerConnection.SetLocalDescription(sd); err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		fmt.Println(state)
	})
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			encodedDescr := signal.Encode(peerConnection.LocalDescription())
			fmt.Println("encodedDescr: " + encodedDescr)
		}
	})
	s.pc = peerConnection
	return &s, sd.SDP
}

func (s *Session) StartSession(sdp string) error {
	err := s.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	})
	if err != nil {
		panic(err)
	}
	go s.run()
	return nil
}

func (s *Session) run() {
	go s.videoPump()
	go s.audioPump()
}

func (s *Session) videoPump() {
	for buf := range s.VideoChan {
		packet := &rtp.Packet{}
		if err := packet.Unmarshal(buf); err != nil {
			panic(err)
		}
		packet.Header.PayloadType = webrtc.DefaultPayloadTypeVP8
		packet.Header.SSRC = s.videoSenderTrack.SSRC()
		if writeErr := s.videoSenderTrack.WriteRTP(packet); writeErr != nil {
			panic(writeErr)
		}
	}
}

func (s *Session) audioPump() {
	for buf := range s.AudioChan {
		packet := &rtp.Packet{}
		if err := packet.Unmarshal(buf); err != nil {
			panic(err)
		}
		packet.Header.PayloadType = webrtc.DefaultPayloadTypeOpus
		packet.Header.SSRC = s.audioSenderTrack.SSRC()
		if writeErr := s.audioSenderTrack.WriteRTP(packet); writeErr != nil {
			panic(writeErr)
		}
	}
}

func (s *Session) Close() {
	s.pc.Close()
}
