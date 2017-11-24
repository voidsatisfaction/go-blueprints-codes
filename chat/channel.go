package main

import (
	"log"
	"net/http"
	"os"

	"github.com/stretchr/objx"
	"github.com/voidsatisfaction/blueprints/chat/trace"

	"github.com/gorilla/websocket"
)

type channel struct {
	roomMap map[string]*room
}

func newChannel() *channel {
	return &channel{
		roomMap: make(map[string]*room),
	}
}

func (ch *channel) checkRoomExist(roomName string) bool {
	_, ok := ch.roomMap[roomName]
	return ok
}

func (ch *channel) makeNewRoom(roomName string) {
	r := newRoom()
	ch.roomMap[roomName] = r
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  socketBufferSize,
	WriteBufferSize: socketBufferSize,
}

func (ch *channel) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}

	// TODO: make hash roomName
	roomName := req.RequestURI
	if roomName == "" {
		log.Fatal("Please input room name")
	}

	if !ch.checkRoomExist(roomName) {
		ch.makeNewRoom(roomName)
		r := ch.roomMap[roomName]
		r.tracer = trace.New(os.Stdout)
		go r.run()
	}
	r := ch.roomMap[roomName]

	authCookie, err := req.Cookie("auth")
	if err != nil {
		log.Fatal("Railed to get auth cookie:", err)
		return
	}
	client := &client{
		socket:   socket,
		send:     make(chan *message, messageBufferSize),
		room:     r,
		userData: objx.MustFromBase64(authCookie.Value),
	}

	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}
