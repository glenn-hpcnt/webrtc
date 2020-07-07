package main

import (
	"fmt"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/examples/internal/signal"
	"time"
)

type Session struct {
	Id          string
	StreamId    string
	pc          *webrtc.PeerConnection
	videoSender *webrtc.RTPSender
	audioSender *webrtc.RTPSender
	VideoChan   chan []byte
	AudioChan   chan []byte
}

func CreateSession(viewId, streamId string) (*Session, string) {
	s := Session{
		Id:        viewId,
		StreamId:  streamId,
		VideoChan: make(chan []byte, 128),
		AudioChan: make(chan []byte, 128),
	}
	mediaEngine := webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000,
		[]webrtc.RTCPFeedback{{Type: webrtc.TypeRTCPFBGoogREMB},
			{Type: webrtc.TypeRTCPFBNACK, Parameter: "pli"},
			{Type: webrtc.TypeRTCPFBNACK}},
		""))

	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	settingEngine := webrtc.SettingEngine{}
	//settingEngine.SetTrickle(true)
	settingEngine.SetLite(true)
	settingEngine.SetNAT1To1IPs([]string{"3.112.113.96"}, webrtc.ICECandidateTypeHost)
	settingEngine.SetEphemeralUDPPortRange(10000, 40000)
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
	s.videoSender = vt.Sender()
	at, err := peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio)
	if err != nil {
		panic(err)
	}
	s.audioSender = at.Sender()
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
	go s.GetREMBPeriodically()
}

func (s *Session) GetREMBPeriodically() {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for range timer.C {
		rtcps, err := s.videoSender.ReadRTCP()
		if err != nil {
			panic(err)
		}
		for _, v := range rtcps {
			remb, ok := v.(*rtcp.ReceiverEstimatedMaximumBitrate)
			if ok {
				println(fmt.Sprintf("bitrate: %d", remb.Bitrate))
			}
		}
	}
}

func (s *Session) videoPump() {
	for buf := range s.VideoChan {
		packet := &rtp.Packet{}
		if err := packet.Unmarshal(buf); err != nil {
			panic(err)
		}
		packet.Header.PayloadType = webrtc.DefaultPayloadTypeVP8
		packet.Header.SSRC = s.videoSender.Track().SSRC()
		if writeErr := s.audioSender.Track().WriteRTP(packet); writeErr != nil {
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
		packet.Header.SSRC = s.audioSender.Track().SSRC()
		if writeErr := s.audioSender.Track().WriteRTP(packet); writeErr != nil {
			panic(writeErr)
		}
	}
}

func (s *Session) Close() {
	s.pc.Close()
}
