package main // jeehoon ha 20164064

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Client struct {
	conn  net.Conn
	read  chan string
	quit  chan int
	name  string
	room  *Room
	quitz chan int
}

const (
	LOGIN          = "1"
	CHAT           = "2"
	ROOM_MAX_USER  = 8
	ROOM_MAX_COUNT = 1
)

type Room struct {
	num        int
	clientlist *list.List
}

var roomlist *list.List
var dest string

func main() { // main for tcp server

	serverPort := "4064"
	roomlist = list.New()
	for i := 0; i < ROOM_MAX_COUNT; i++ {
		room := &Room{i + 1, list.New()}
		roomlist.PushBack(*room)
	}

	ln, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		handleError(nil, err, "listen waiting..")
	}
	defer ln.Close()
	SetupCloseHandler()
	for {
		// waiting connection
		conn, err := ln.Accept()
		if err != nil {
			handleError(conn, err, "listen waiting..")
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) { // always handle connection and basic state
	read := make(chan string)
	quit := make(chan int)
	quitz := make(chan int)
	client := &Client{conn, read, quit, "unknown", &Room{-1, list.New()}, quitz}

	go handleClient(client, conn)

	fmt.Printf("remote Addr = %s\n", conn.RemoteAddr().String())
}

func handleError(conn net.Conn, err error, errmsg string) { // when error cames in printout error and it's message
	if conn != nil {
		conn.Close()
	}
	fmt.Println(err)
	fmt.Println(errmsg)
}

func recvFromClient(client *Client, conn net.Conn) { // receiving from client reads it and discriminate its command by split
	recvmsg, err := bufio.NewReader(client.conn).ReadString('\n')
	if err != nil {
		handleError(client.conn, err, "read waiting..")
		client.quit <- 0
		return
	}

	strmsgs := strings.Split(recvmsg, "|")

	switch strmsgs[0] {
	case LOGIN: // when login start
		client.name = strings.TrimSpace(strmsgs[1])

		room := allocateRoom(client, conn) // gives a location for user and it's room
		if room.num < 1 {
			handleError(client.conn, nil, "max user limit!")
		}
		client.room = room

		if !client.dupUserCheck() { // check whether it is dup or not

			//handleError(client.conn, nil, "duplicate user!"+client.name)
			client.quitz <- 0
			return
		}

		for re := roomlist.Front(); re != nil; re = re.Next() {
			r := re.Value.(Room)
			for e := r.clientlist.Front(); e != nil; e = e.Next() {
				c := e.Value.(Client)
				user_num := strconv.Itoa(r.clientlist.Len() + 1)

				room := allocateRoom(client, conn)
				if room.num < 1 { // when full
					sendToClient(client, "", client.name+" tried to participate but this room is full.")

				} else if !client.dupUserCheck() { // when dup
					user_num = strconv.Itoa(r.clientlist.Len())
					sendToClient(&c, "", "that nickname is already used by another user. cannot connect.")
				} else { // when it is normally

					sendToClient(&c, "", "[Welcome "+client.name+" to CAU network class chat room at "+conn.RemoteAddr().String()+".] [There are "+user_num+" users connected. \n]")
				}
			}
		}
		//fmt.Printf("\nhello = %s, your room number is = %d\n", client.name, client.room.num)
		room.clientlist.PushBack(*client)

	case CHAT:
		fmt.Printf("\nrecv message = %s\n", strmsgs[1])
		client.read <- strmsgs[1]
	}
}

func handleClient(client *Client, conn net.Conn) { // always handle the input msg and handle it
	for {
		select {
		case msg := <-client.read:

			if strings.Contains(msg, "\\") { // when \ this is detected
				if strings.HasPrefix(msg, "\\list") { // when list function
					sendlistClients(client, client.name, "")
				} else if strings.HasPrefix(msg, "\\exit") { // exit function
					sendToClient(client, client.name, "gg~\n")
					client.conn.Close()
				} else if strings.HasPrefix(msg, "\\ver") { // version conceal out
					sendToClient(client, "", "Go = 1.18\n")
				} else if strings.HasPrefix(msg, "\\rtt") {
					checkRTT(client, client.conn)
				} else if strings.HasPrefix(msg, "\\dm") { // whisper message direct message
					send_Whisper(client, msg)
				} else {
					sendToClient(client, client.name, "invalid command\n")
				}
			} else if strings.Contains(msg, "i hate professor") { // when i hate professor is detected
				sendToAllClients("client", msg)
				client.conn.Close()
			} else {
				sendToOtherClients(client.name, msg)
			}

		case <-client.quit:
			for e := roomlist.Front(); e != nil; e = e.Next() {
				r := e.Value.(Room)
				client.conn.Close()
				client.deleteFromList()
				user_num := strconv.Itoa(r.clientlist.Len())

				{
					sendToAllClients("", "["+client.name+" is disconnected. There are "+user_num+" users in the chat room.]\n") // when its end connection printout and declare all clients remain in room
				}
			}

			return
		case <-client.quitz:

			client.deleteFromList()
			client.conn.Close()

		default:
			go recvFromClient(client, conn)
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func sendToClient(client *Client, sender string, msg string) { // send messages to special client
	var buffer bytes.Buffer

	buffer.WriteString(sender)
	buffer.WriteString("> ")
	buffer.WriteString(msg)
	buffer.WriteString("\n")

	fmt.Printf("client = %s ==> %s", client.name, buffer.String())

	fmt.Fprintf(client.conn, "%s", buffer.String())
}

func sendToAllClients(sender string, msg string) { // send messages to all client
	fmt.Printf("broad cast message = %s", msg)
	for re := roomlist.Front(); re != nil; re = re.Next() {
		r := re.Value.(Room)
		for e := r.clientlist.Front(); e != nil; e = e.Next() {
			c := e.Value.(Client)

			sendToClient(&c, sender, msg+"\n")

		}
	}
}
func sendToOtherClients(sender string, msg string) { // send messages to other clients
	fmt.Printf("broad cast message without self = %s", msg)
	for re := roomlist.Front(); re != nil; re = re.Next() {
		r := re.Value.(Room)
		for e := r.clientlist.Front(); e != nil; e = e.Next() {
			c := e.Value.(Client)
			if sender != c.name {
				sendToClient(&c, sender, msg)
			}

		}
	}
}
func sendlistClients(client *Client, sender string, msg string) { // send messages to other clients

	for re := roomlist.Front(); re != nil; re = re.Next() {
		r := re.Value.(Room)
		k := re.Value.(Room).clientlist.Front().Value.(Client)
		for e := r.clientlist.Front(); e != nil; e = e.Next() {
			c := e.Value.(Client)
			msg = c.name + " " + c.conn.RemoteAddr().String() + " " + msg // nickname + ip + port number
			if sender == c.name {
				k = c
			}
		}
		sendToClient(&k, sender, "list = "+msg+"\n") // send to client who request \list
		fmt.Printf("list = %s", msg)
	}
}

func (client *Client) deleteFromList() { // delete client from the chat room list
	for re := roomlist.Front(); re != nil; re = re.Next() {
		r := re.Value.(Room)
		for e := r.clientlist.Front(); e != nil; e = e.Next() {
			c := e.Value.(Client)
			if client.conn == c.conn {
				r.clientlist.Remove(e)
			}
		}
	}
}

func (client *Client) dupUserCheck() bool { // checking the nickname is dup or not
	for re := roomlist.Front(); re != nil; re = re.Next() {
		r := re.Value.(Room)
		for e := r.clientlist.Front(); e != nil; e = e.Next() {
			c := e.Value.(Client)
			if strings.Compare(client.name, c.name) == 0 {

				return false
			} else {

			}
		}
	}

	return true
}
func checkRTT(client *Client, conn net.Conn) { // check RTT time
	startTime := time.Now()
	startT := startTime.Format("15:04:05.000")

	startHour := startT[0:2]   // cut hour
	startMinute := startT[3:5] // cut minute
	startSecond := startT[6:8] // cut second
	startMili := startT[9:12]

	convHours, _ := time.ParseDuration(startHour + "h")     // cut format hour
	convMinutes, _ := time.ParseDuration(startMinute + "m") // cut format minute
	convSecond, _ := time.ParseDuration(startSecond + "s")  // cut format second
	convMiliseconds, _ := time.ParseDuration(startMili + "ms")
	conn.Write([]byte("time checking.."))
	conn.Write([]byte(".."))
	endTime := time.Now()
	endnanoT := endTime.Add(-convHours - convMinutes - convSecond - convMiliseconds).Format(".000000") // calculate
	endmiliT := endTime.Add(-convHours - convMinutes - convSecond - convMiliseconds).Format(".000")

	conn.Write([]byte("rtt:" + endmiliT + "ms or " + endnanoT + "ns\n")) // send time gap with a formular format

}
func allocateRoom(client *Client, conn net.Conn) *Room { // make client into the room
	for e := roomlist.Front(); e != nil; e = e.Next() {
		r := e.Value.(Room)
		var buffer bytes.Buffer

		client_num := strconv.Itoa(r.clientlist.Len() + 1)
		var msg string

		if !client.dupUserCheck() {
			msg = "that nickname is already used by another user.cannot connect."
		} else {
			msg = "[Welcome " + client.name + " to CAU network class chat room at " + conn.RemoteAddr().String() + ".] [There are " + client_num + " users connected.\n" //port number and user number
			if r.clientlist.Len() >= ROOM_MAX_USER {                                                                                                                     // when its exceed limit of room
				msg = "chatting room full. cannot connect."
			}
		}

		buffer.WriteString(msg)
		fmt.Fprintf(client.conn, "%s", buffer.String())

		if r.clientlist.Len() < ROOM_MAX_USER {
			return &r
		}

	}

	// when it is full room returns it out for common value

	return &Room{-1, list.New()}
}

func sendToRoomClients(room *Room, sender string, msg string) { // send messages to client in this room
	fmt.Printf("room broad cast message = %s", msg)
	for e := room.clientlist.Front(); e != nil; e = e.Next() {
		c := e.Value.(Client)
		sendToClient(&c, sender, msg)
	}
}

func findClientByName(name string) *Client { // find name for the whisper
	for re := roomlist.Front(); re != nil; re = re.Next() {
		r := re.Value.(Room)
		for e := r.clientlist.Front(); e != nil; e = e.Next() {
			c := e.Value.(Client)
			if strings.Compare(c.name, name) == 0 {
				return &c // return its location of user
			}
		}
	}

	return &Client{nil, nil, nil, "unknown", nil, nil}
}

func send_Whisper(client *Client, msg string) { // whisper for \\dm function
	strmsgs := strings.Split(msg, " ")

	target := findClientByName(strmsgs[1])
	if target.conn == nil {
		fmt.Println("Can't find target User")
		return
	}

	sendToClient(target, "from: "+client.name, strmsgs[2]) // sends to whisper object
}

func SetupCloseHandler() { // ctrl+c handling for input option
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r      \ngg~") // end message printout

		os.Exit(0)
	}()
}
