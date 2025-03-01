package echos

import (
	"fmt"
	"net"

	"github.com/pion/stun/v3"
)

func startSTUNon(network, address string) {
	conn, err := net.ListenPacket(network, address)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("STUN server is listening on %s (%s)...\n", address, network)
	buffer := make([]byte, 1500)

	for {
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			continue
		}

		message := &stun.Message{}
		if err := message.UnmarshalBinary(buffer[:n]); err != nil {
			fmt.Printf("Invalid STUN message from %v: %v\n", addr, err)
			continue
		}

		if message.Type.Method == stun.MethodBinding {
			fmt.Printf("Received STUN Binding request from %v\n", addr)

			response := stun.MustBuild(
				stun.TransactionID,
				stun.BindingSuccess,
				&stun.XORMappedAddress{
					IP:   addr.(*net.UDPAddr).IP,
					Port: addr.(*net.UDPAddr).Port,
				},
			)

			if _, err := conn.WriteTo(response.Raw, addr); err != nil {
				fmt.Printf("Failed to send STUN response: %v\n", err)
			} else {
				fmt.Printf("Sent STUN Binding response to %v\n", addr)
			}
		}
	}
}

func StartSTUN() {
	go startSTUNon("udp4", "0.0.0.0:3478")
}
