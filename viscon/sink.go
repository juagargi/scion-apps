package main

import (
	"encoding/binary"
	"fmt"

	"github.com/netsec-ethz/scion-apps/lib/scionutil"
	"github.com/scionproto/scion/go/lib/addr"
	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/snet"
)

// RECEPTION_BUFF_SIZE is the size of the buffer to read the packet from a client.
const RECEPTION_BUFF_SIZE = 1024 * 65

// PORT is the UDP port used to listen for incoming traffic.
const PORT = 12345

var localAddress *snet.Addr

func fatal(err error) {
	panic(err)
}
func logError(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func initNetwork() {
	localAddressStr, err := scionutil.GetLocalhostString()
	if err != nil {
		fatal(err)
	}
	localAddress, err = snet.AddrFromString(localAddressStr)
	if err != nil {
		fatal(err)
	}
	dispatcherPath := scionutil.GetDefaultDispatcher()
	sciondPath := sciond.GetDefaultSCIONDPath(nil)
	err = snet.Init(localAddress.IA, sciondPath, dispatcherPath)
	if err != nil {
		fatal(err)
	}
}

func handleClients(conn snet.Conn) {

	for {
		buffer := make([]byte, RECEPTION_BUFF_SIZE)
		n, clientAddr, err := conn.ReadFromSCION(buffer)
		if err != nil {
			logError("Error while reading from connection", err)
		}
		go handleClient(conn, buffer[:n], clientAddr)
	}
}

func handleClient(conn snet.Conn, buffer []byte, clientAddr *snet.Addr) {
	fmt.Printf("Doing whatever we do with a client. Client: %s, buffer size = %d\n", clientAddr, len(buffer))
	sendAck(conn, clientAddr, uint16(len(buffer)))
}

func sendAck(conn snet.Conn, clientAddr *snet.Addr, message uint16) {
	ackMessage := make([]byte, 2)
	binary.LittleEndian.PutUint16(ackMessage, message)
	_, err := conn.WriteTo(ackMessage, clientAddr)
	if err != nil {
		logError("Error sending ACK: %v", err)
	}
}

func main() {
	fmt.Println("On!")
	initNetwork()
	localAddress.Host.L4 = addr.NewL4UDPInfo(PORT)
	conn, err := snet.ListenSCION("udp4", localAddress)
	if err != nil {
		fatal(err)
	}
	handleClients(conn)
}
