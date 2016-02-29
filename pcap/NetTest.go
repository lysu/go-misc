package main

import (
	"fmt"
	"github.com/akrennmair/gopcap"
)

func main() {

	ifs, err := pcap.Findalldevs()
	if err != nil {
		panic(err)
	}
	fmt.Println("Network Interface List:")
	var nif pcap.Interface
	for _, ifr := range ifs {
		fmt.Println("  ", ifr.Description, " ", ifr.Name)
		if ifr.Name == "en0" {
			nif = ifr
		}
	}

	handler, err := pcap.Openlive(nif.Description, 10*1024, true, 1)
	if err != nil {
		panic(err)
	}

	go func() {
		pcap.
	}()
}
