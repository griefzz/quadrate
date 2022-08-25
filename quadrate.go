package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Client struct {
	Name string
	Conn *websocket.Conn
}

type Message struct {
	Type string
	Data string
}

var Clients = make(map[Client]bool)

var Words []string

func createUsername() string {
	var name string

	r := []rune(Words[rand.Intn(len(Words)-1)])
	name += string(append([]rune{unicode.ToUpper(r[0])}, r[1:]...))

	r = []rune(Words[rand.Intn(len(Words)-1)])
	name += string(append([]rune{unicode.ToUpper(r[0])}, r[1:]...))

	r = []rune(Words[rand.Intn(len(Words)-1)])
	name += string(append([]rune{unicode.ToUpper(r[0])}, r[1:]...))

	return name
}

func sendToAll(clients map[Client]bool, msg Message) {
	jresp, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	for c := range clients {
		c.Conn.WriteMessage(websocket.TextMessage, jresp)
	}
}

func sendToAllExcept(clients map[Client]bool, client Client, msg Message) {
	jresp, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	for c := range clients {
		if c != client {
			c.Conn.WriteMessage(websocket.TextMessage, jresp)
		}
	}
}

func sendClientList(clients map[Client]bool) {
	var names string
	for c := range clients {
		names += c.Name + ","
	}

	jresp, err := json.Marshal(Message{"update", names})
	if err != nil {
		panic(err)
	}

	for c := range clients {
		c.Conn.WriteMessage(websocket.TextMessage, jresp)
	}
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %+v", err)
		return
	}
	defer conn.Close()

	client := Client{createUsername(), conn}
	Clients[client] = true

	// give client a list of connected users
	sendClientList(Clients)

	for {
		// Read message from browser
		_, msg, err := conn.ReadMessage()
		if err != nil {
			delete(Clients, client)
			sendClientList(Clients)
			break
		}

		m := Message{}
		err = json.Unmarshal(msg, &m)
		if err != nil {
			fmt.Println("Failed to unmarshal message: ", msg)
		}

		switch m.Type {
		case "chat":
			sendToAll(Clients, Message{"chat", client.Name + "," + m.Data})
		case "add":
			sendToAll(Clients, Message{"add", m.Data})
		case "pause":
			sendToAllExcept(Clients, client, Message{"pause", ""})
		case "play":
			sendToAllExcept(Clients, client, Message{"play", ""})
		case "seek":
			sendToAllExcept(Clients, client, Message{"seek", m.Data})
		}

		// Print the message to the console
		fmt.Printf("%s:%s sent: %s\n", client.Name, client.Conn.RemoteAddr(), string(msg))
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// create word list
	content, err := os.ReadFile("wordlist.txt")
	if err != nil {
		panic(err)
	}

	Words = strings.Split(string(content), "\n")

	r := gin.Default()
	r.LoadHTMLFiles("index.html")
	r.SetTrustedProxies([]string{"192.168.1.4"})

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.GET("/ws", func(c *gin.Context) {
		wshandler(c.Writer, c.Request)
	})

	//r.Run("192.168.1.4:12312")
	r.Run("localhost:12312")
}
