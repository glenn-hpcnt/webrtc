package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const TIMEOUT = time.Second * 10

var httpClient = http.Client{Timeout: TIMEOUT}

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

type EndpointData struct {
	Host       string `json:"host"`
	Simulcast  bool   `json:"simulcast"`
	AudioPtype uint8  `json:"audioPtype"`
	VideoPtype uint8  `json:"videoPtype"`
	AudioPort  uint16 `json:"audioPort"`
	VideoPort  uint16 `json:"videoPort"`
	VideoPort2 uint16 `json:"videoPort2"`
	AudioSsrc  uint32 `json:"audioSsrc"`
	VideoSsrc  uint32 `json:"videoSsrc"`
	VideoSsrc2 uint32 `json:"videoSsrc2"`
}

func (c *Client) RequestForward(origin string, streamId string, endpointData EndpointData) (string, error) {
	b, err := json.Marshal(endpointData)
	if err != nil {
		fmt.Println("Failed to marshal error: ", err)
		return "", err
	}
	fmt.Println("sending POST /forward/", streamId, " to ", origin, " with ", string(b))
	addr := fmt.Sprintf("http://%s/forward/%s", origin, streamId)
	resp, err := httpClient.Post(addr, "application/json", bytes.NewBuffer(b))
	if err != nil {
		fmt.Println("Failed to sending data: ", err)
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		_, err := buf.ReadFrom(resp.Body)
		if err != nil {
			fmt.Println("Failed to read forward error response: ", err)
			return "", err
		}
		return "", errors.New(buf.String())
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		fmt.Println("Failed to sending data: ", err)
		return "", err
	}
	return buf.String(), nil
}

func (c *Client) RequestStopForward(origin string, streamId string, forwardId string) error {
	fmt.Println("sending DELETE /forward/", streamId, "/", forwardId, " to ", origin)
	addr := fmt.Sprintf("http://%s/forward/%s/%s", origin, streamId, forwardId)
	req, err := http.NewRequest("DELETE", addr, nil)
	if err != nil {
		fmt.Println("Failed to construct HTTP DELETE request: ", err)
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println("Failed to sending data: ", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		_, err := buf.ReadFrom(resp.Body)
		if err != nil {
			fmt.Println("Failed to read stop forward error response: ", err)
			return err
		}
		return errors.New(buf.String())
	}
	return nil
}
