package echos

import (
	"fmt"
	"net"

	"github.com/pion/stun/v3"
)

func StartSTUN() {
	conn, err := net.ListenPacket("udp", ":3478")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("STUN server is listening on port 3478...")

	buffer := make([]byte, 1500)
	for {
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Errorf("Read error: %v", err)
			continue
		}

		udpAddr, ok := addr.(*net.UDPAddr)
		if !ok {
			log.Errorf("Failed to cast net.Addr to *net.UDPAddr")
			continue
		}

		message := &stun.Message{}
		if err := message.UnmarshalBinary(buffer[:n]); err != nil {
			log.Errorf("Invalid STUN message from %v: %v", addr, err)
			continue
		}

		if message.Type.Method == stun.MethodBinding {
			log.Infof("Received STUN Binding request from %v\n", addr)

			response := stun.MustBuild(
				stun.TransactionID,
				stun.BindingSuccess,
				&stun.MappedAddress{
					IP:   udpAddr.IP,
					Port: udpAddr.Port,
				},
			)

			if _, err := conn.WriteTo(response.Raw, addr); err != nil {
				log.Errorf("Failed to send STUN response: %v", err)
			} else {
				log.Infof("Sent STUN Binding response to %v\n", addr)
			}
		}
	}
}
