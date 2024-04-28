package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type LinkUpMessage struct {
	Header  unix.NlMsghdr
	Payload unix.IfInfomsg
}

func main() {

	// read the eth. index from arguments
	ethIndex, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("the usage is netlinkdemo [eth index]\n")
		return
	}

	// open the Netlink socket
	sock, err := unix.Socket(
		unix.AF_NETLINK,
		unix.SOCK_RAW,
		unix.NETLINK_ROUTE,
	)

	if err != nil {
		fmt.Printf("Error creating socket: %s\n", err)
		return
	}

	defer unix.Close(sock)

	// bind the socket to group and PID
	err = unix.Bind(sock, &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: 0,
		Pid:    0,
	})

	if err != nil {
		fmt.Printf("Error in binding socket: %s\n", err)
		return
	}

	payload := unix.IfInfomsg{
		Family: unix.AF_UNSPEC,
		Change: unix.IFF_UP,
		Flags:  unix.IFF_UP,
		Index:  int32(ethIndex), // index of network interface I would like to enable (in my case it's 7 - veth0)
	}

	// total length of message is size of header + size of payload
	length := unix.SizeofNlMsghdr + unix.SizeofIfInfomsg

	header := unix.NlMsghdr{
		Len:   uint32(length),
		Type:  uint16(unix.RTM_NEWLINK),
		Flags: uint16(unix.NLM_F_REQUEST) | uint16(unix.NLM_F_ACK),
		Seq:   1,
	}

	msg := struct {
		header  unix.NlMsghdr
		payload unix.IfInfomsg
	}{
		header:  header,
		payload: payload,
	}

	// first I need convert the `msg` to slice of bytes
	var asByteSlice []byte = (*(*[unix.SizeofNlMsghdr + unix.SizeofIfInfomsg]byte)(unsafe.Pointer(&msg)))[:]

	// write the data to the socket
	err = unix.Sendto(sock, asByteSlice, 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK})
	if err != nil {
		fmt.Printf("Could not write message to socket:%s\n", err)
	}

	// receiving data
	var buf [1024]byte
	n, _, err := unix.Recvfrom(sock, buf[:], 0)

	if err != nil {
		fmt.Printf("Could not read data from socket: %s\n", err)
		return
	}

	// parse data to messages
	msgs, err := syscall.ParseNetlinkMessage(buf[:n])

	if err != nil {
		fmt.Printf("Could not parse the response: %s\n", err)
		return
	}

	if msgs[0].Header.Type != unix.NLMSG_ERROR {
		fmt.Printf("The first received message is not NLMSG_ERROR\n")
		return
	}

	// cast the data to NlMsgerr payload
	errPayload := (*unix.NlMsgerr)(unsafe.Pointer(&msgs[0].Data[0]))
	if errPayload.Error != 0 {
		fmt.Printf("Error returned by Netlink\n")
	}

	fmt.Printf("Interface is UP\n")
}
