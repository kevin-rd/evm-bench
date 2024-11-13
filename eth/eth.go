package eth

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type Client struct {
	URL string
	ws  *websocket.Conn
}

func NewClient(url string, rpcAddr string) (*Client, error) {
	c := Client{URL: url}
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	_ = ws.SetReadDeadline(time.Now().Add(time.Second * 20))
	_ = ws.SetWriteDeadline(time.Now().Add(time.Second * 60))
	c.ws = ws
	return &c, nil
}

func (c *Client) ReConn() error {
	if err := c.ws.Close(); err != nil {
		log.Fatalf("Error Close ws: %v", err)
	}
	ws, _, err := websocket.DefaultDialer.Dial(c.URL, nil)
	if err != nil {
		log.Fatalf("Error ReConn to ws: %v", err)
	}
	_ = ws.SetReadDeadline(time.Now().Add(time.Second * 20))
	_ = ws.SetWriteDeadline(time.Now().Add(time.Second * 60))
	c.ws = ws
	return nil
}

func (c *Client) WriteJSON(id MethodId, params []interface{}) error {
	return c.ws.WriteJSON(&JSONRPCRequest{
		Version: DefaultVersion,
		Method:  id.String(),
		Params:  params,
		ID:      int(id),
	})
}

func (c *Client) ReadMessage() (messageType int, p []byte, err error) {
	return c.ws.ReadMessage()
}

func (c *Client) ReadResponse() (resp JSONRPCResponse, err error) {
	_, message, err := c.ReadMessage()
	if err != nil {
		return
	}
	err = json.Unmarshal(message, &resp)
	return
}

func (c *Client) Close() error {
	return c.ws.Close()
}
