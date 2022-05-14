package main // jeehoon ha 20164064

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"regexp"
)

const (
	LOGIN = "1"
	CHAT  = "2"
)

func main() {
	serverName := "nsl2.cau.ac.kr"
	serverPort := "4064" // 20164064 my port number
	conn, err := net.Dial("tcp", serverName+":"+serverPort)
	if err != nil {
		handleError(conn, "there is no server")
	}

	msgch := make(chan string)

	reader := bufio.NewReader(os.Stdin)
	SetupCloseHandler()

	fmt.Print("Input your name :")
	setname, err := reader.ReadString('\n')
	re := regexp.MustCompile(`[\{\}\[\]\/?.,;:|\)*~!^\-_+<>@\#$%&\\\=\(\'\"]+`) // make special letter to none
	name := re.ReplaceAllString(setname, "")                                    // special letter eliminate and set new name
	if len(name) >= 32 {                                                        // inspect length of nickname
		fmt.Print("Id error too long:\n")
		os.Exit(0) // call end
	}

	if err != nil {
		handleError(conn, "read fail..")
	}

	// login
	fmt.Fprintf(conn, "%s|%s", LOGIN, name)

	// chat
	go handleRecvMsg(conn, msgch)
	handleSendMsg(conn)
}

func handleError(conn net.Conn, errmsg string) { // when error comes in handle it
	fmt.Println(errmsg)
	if conn != nil {
		conn.Close()
	}

}

func handleSendMsg(conn net.Conn) { // printout message
	for {
		reader := bufio.NewReader(os.Stdin)

		text, err := reader.ReadString('\n')
		if err != nil {
			handleError(conn, "read input failed..")
		}

		fmt.Fprintf(conn, "%s|%s", CHAT, text)
	}
}

func handleRecvMsg(conn net.Conn, msgch chan string) { // first function that handle msg
	for {
		select {
		case msg := <-msgch:
			fmt.Printf("\n%s\n", msg)
		default:
			go recvFromServer(conn, msgch)
			time.Sleep(1000 * time.Millisecond)
		}
	}
}

func recvFromServer(conn net.Conn, msgch chan string) { // when nil is detected it ends tcp connection and end it
	msg, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf(msg)
	if err != nil {
		handleError(conn, "")
		os.Exit(2)
		return
	}

}
func SetupCloseHandler() { // ctrl+c handling for input option
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {

		<-c
		fmt.Println("\r   		   \ngg~") // end message printout

		os.Exit(0)
	}()
}
