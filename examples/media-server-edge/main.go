package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	http.HandleFunc("/", websocketHandlerFunc)
	http.HandleFunc("/validateStream", validateStreamHandlerFunc)
	go func() {
		addr := fmt.Sprintf("0.0.0.0:%d", 8080)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Println("Failed to listen server on ", addr, ", err:", err)
			panic(err.Error)
		}
	}()
	fmt.Println("Boot Pion SFU")

	sig := make(chan os.Signal, 64)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Pion server stopped by signal: ", <-sig)
}

func websocketHandlerFunc(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to upgrade http connection to websocket")
		return
	}
	go Pump(c)
}
func validateStreamHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	type ValidateStreamResponse struct {
		InvalidIds []string `json:"invalidIds"`
	}
	b, err := json.Marshal(&ValidateStreamResponse{InvalidIds: nil})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func Pump(conn *websocket.Conn) {
	for {
		_, buf, err := conn.ReadMessage()
		if err != nil {
			panic(err)
		}
		var request RequestBase
		if err = json.Unmarshal(buf, &request); err != nil {
			panic(err)
		}

		switch request.MessageType {
		case MessageTypeRequest:
			switch request.Request {
			case "keepAlive":
				go func() {
					var kar KeepAliveRequest
					if err = json.Unmarshal(buf, &kar); err != nil {
						panic(err)
					}
					buf, err := json.Marshal(&KeepAliveResponse{ResponseBase{
						MessageType: MessageTypeResponse,
						Response:    kar.Request,
						SessionId:   kar.SessionId,
						Transaction: kar.Transaction,
						Status:      "ok",
					}})
					if err != nil {
						panic(err)
					}
					if err = conn.WriteMessage(websocket.TextMessage, buf); err != nil {
						panic(err)
					}
				}()
			case "createView":
				go func() {
					var cvr CreateViewRequest
					if err = json.Unmarshal(buf, &cvr); err != nil {
						panic(err)
					}
					res := handleCreateView(cvr)
					buf, err := json.Marshal(res)
					if err != nil {
						panic(err)
					}
					if err = conn.WriteMessage(websocket.TextMessage, buf); err != nil {
						panic(err)
					}
				}()
			case "startStream":
				go func() {
					var ssr StartStreamRequest
					if err = json.Unmarshal(buf, &ssr); err != nil {
						panic(err)
					}
					res := handleStartStream(ssr)
					buf, err := json.Marshal(res)
					if err != nil {
						panic(err)
					}
					if err = conn.WriteMessage(websocket.TextMessage, buf); err != nil {
						panic(err)
					}
				}()
			case "configureView":
				go func() {
					var cvr ConfigureViewRequest
					if err = json.Unmarshal(buf, &cvr); err != nil {
						panic(err)
					}
					buf, err := json.Marshal(&ConfigureViewResponse{ResponseBase{
						MessageType: MessageTypeResponse,
						Response:    cvr.Request,
						SessionId:   cvr.SessionId,
						Transaction: cvr.Transaction,
						Status:      "ok",
					}})
					if err != nil {
						panic(err)
					}
					if err = conn.WriteMessage(websocket.TextMessage, buf); err != nil {
						panic(err)
					}
				}()
			case "destroyView":
				go func() {
					var dvr DestroyViewRequest
					if err = json.Unmarshal(buf, &dvr); err != nil {
						panic(err)
					}
					res := handleDestroyView(dvr)
					buf, err := json.Marshal(res)
					if err != nil {
						panic(err)
					}
					if err = conn.WriteMessage(websocket.TextMessage, buf); err != nil {
						panic(err)
					}
				}()
			default:
				fmt.Println("unexpected message is arrived, msg: ", string(buf))
			}
		case MessageTypePing:
			go func() {
				var ping Ping
				if err = json.Unmarshal(buf, &ping); err != nil {
					panic(err)
				}
				buf, err := json.Marshal(&Pong{
					MessageType:  MessageTypePong,
					Response:     "pong",
					Transaction:  ping.Transaction,
					SessionCount: 0,
				})
				if err != nil {
					panic(err)
				}
				if err = conn.WriteMessage(websocket.TextMessage, buf); err != nil {
					panic(err)
				}
			}()
		}
	}
}

var sessionMap sync.Map
var endPointMap sync.Map

func handleCreateView(cvr CreateViewRequest) *CreateViewResponse {
	var endPoint *EndPoint
	if e, prs := endPointMap.Load(cvr.StreamId); prs {
		fmt.Println(cvr.StreamId + " already has endpoint.")
		endPoint = e.(*EndPoint)
	} else {
		e, err := CreateEndPoint("10.207.11.156:8080", cvr.StreamId)
		if err != nil {
			panic(err)
		}
		endPointMap.Store(cvr.StreamId, e)
		endPoint = e
	}
	sess, sdp := CreateSession(cvr.SessionId, cvr.StreamId)
	if sess == nil {
		panic("failed to create session")
	}
	if err := endPoint.AttachSession(sess); err != nil {
		panic(err)
	}
	sessionMap.Store(cvr.SessionId, sess)
	return &CreateViewResponse{
		ResponseBase: ResponseBase{
			MessageType: MessageTypeResponse,
			Response:    cvr.RequestBase.Request,
			SessionId:   cvr.RequestBase.SessionId,
			Transaction: cvr.RequestBase.Transaction,
			Status:      "ok",
		},
		EndpointFull:   false,
		VideoEnabled:   true,
		AudioEnabled:   true,
		Offer:          sdp,
		SubstreamRange: 2,
		SessionCount:   1,
	}
}

func handleStartStream(ssr StartStreamRequest) *StartStreamResponse {
	sess, prs := sessionMap.Load(ssr.SessionId)
	if !prs {
		panic("no such session")
	}
	if err := sess.(*Session).StartSession(ssr.Answer); err != nil {
		panic(err)
	}
	return &StartStreamResponse{ResponseBase{
		MessageType: MessageTypeResponse,
		Response:    ssr.RequestBase.Request,
		SessionId:   ssr.RequestBase.SessionId,
		Transaction: ssr.RequestBase.Transaction,
		Status:      "ok",
	}}
}

func handleDestroyView(dvr DestroyViewRequest) *DestroyViewResponse {
	dvRes := &DestroyViewResponse{ResponseBase{
		MessageType: MessageTypeResponse,
		Response:    dvr.Request,
		SessionId:   dvr.SessionId,
		Transaction: dvr.Transaction,
		Status:      "ok",
	}}
	sess, prs := sessionMap.Load(dvr.SessionId)
	if !prs {
		fmt.Println("no such session, id: ", dvr.SessionId)
		return dvRes
	}
	session := sess.(*Session)
	session.Close()
	ep, prs := endPointMap.Load(session.StreamId)
	if !prs {
		fmt.Println("no such endpoint, streamId: ", session.StreamId)
		return dvRes
	}
	ep.(*EndPoint).DetachSession(session.Id)
	return dvRes
}
