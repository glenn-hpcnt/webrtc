package main

type MessageType struct {
	MessageType string `json:"messageType"`
}

var MessageTypePing = MessageType{MessageType: "ping"}
var MessageTypePong = MessageType{MessageType: "pong"}
var MessageTypeRequest = MessageType{MessageType: "request"}
var MessageTypeResponse = MessageType{MessageType: "response"}

type Ping struct {
	MessageType
	Request     string `json:"request"`
	Transaction string `json:"id"`
}

type Pong struct {
	MessageType
	Response     string `json:"response"`
	Transaction  string `json:"id"`
	SessionCount int    `json:"sessionCount"`
}

type Request interface {
	Base() *RequestBase
}

type RequestBase struct {
	MessageType
	Request     string `json:"request"`
	SessionId   string `json:"sessionId"`
	Transaction string `json:"id"`
}

var _ Request = &RequestBase{}

func (rb *RequestBase) Base() *RequestBase {
	return rb
}

type Response interface {
	Base() *ResponseBase
}

type ResponseBase struct {
	MessageType
	Response    string `json:"response"`
	SessionId   string `json:"sessionId"`
	Transaction string `json:"id"`
	Status      string `json:"status"`
}

var _ Response = &ResponseBase{}

func (rb *ResponseBase) Base() *ResponseBase {
	return rb
}

type CreateViewRequest struct {
	RequestBase
	ViewConf
	Region   string `json:"region"`
	StreamId string `json:"streamId"`
	UseSdes  bool   `json:"useSdes"`
	Force    bool   `json:"force"`
}

type CreateViewResponse struct {
	ResponseBase
	EndpointFull   bool   `json:"endpointFull"`
	VideoEnabled   bool   `json:"videoEnabled"`
	AudioEnabled   bool   `json:"audioEnabled"`
	Offer          string `json:"offer"`
	SubstreamRange int    `json:"substreamRange"`
	SessionCount   int    `json:"sessionCount"`
}

type ViewConf struct {
	PlayVideo   bool   `json:"playVideo"`
	PlayAudio   bool   `json:"playAudio"`
	PlayoutMode string `json:"playoutMode"`
}

type StartStreamRequest struct {
	RequestBase
	Answer   string                 `json:"answer"`
	UserData map[string]interface{} `json:"userData"`
}

type StartStreamResponse struct {
	ResponseBase
}

type ConfigureViewRequest struct {
	RequestBase
	ViewConf
	PreferredSubstream int `json:"preferredSubstream"`
}

type ConfigureViewResponse struct {
	ResponseBase
}

type KeepAliveRequest struct {
	RequestBase
}

type DestroyViewRequest struct {
	RequestBase
}

type DestroyViewResponse struct {
	ResponseBase
}

type KeepAliveResponse struct {
	ResponseBase
}
