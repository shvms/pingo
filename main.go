package main

import (
	"flag"
	"fmt"
	"log"
)

var usage = `
Ping implementation in Golang.
ping [-c=count] [-i=interval] [-t=timeout] [-ttl=TTL value] [-s=packetsize] host
------------------------------------------------------------------
count: Unsigned integer. 0 represents infinite ping.

interval: Unsigned Integer. Interval between each ICMP Echo request (in ms). Defaults to 1000ms.

timeout: Unsigned Integer. Timeout to wait for response from the host (in ms). Defaults to 1000ms.

packetsize: Unsigned Integer. Packet size, in bytes, to ping with. Defaults to 32 bytes.

TTL value: Integer. Time-to-live of the ICMP packets sent. Defaults to 64.
`

func main() {
	count := flag.Uint("c", 0, "Unsigned integer. 0 represents infinite ping.")
	interval := flag.Uint("i", 1000, "Integer. Interval between each ICMP Echo request (in ms). Defaults to 1000ms.")
	timeout := flag.Uint("t", 1000, "Unsigned Integer. Timeout to wait for response from the host (in ms). Defaults to 1000ms.")
	packetSize := flag.Uint("s", 32, "Unsigned Integer. Packet size, in bytes, to ping with. Defaults to 32 bytes.")
	ttl := flag.Int("ttl", 64, "Integer. Time-to-live of the ICMP packets sent. Defaults to 64.")
	flag.Usage = func() {
		fmt.Println(usage)
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	host := flag.Arg(0)
	ping, err := PingObj(host)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	ping.count = *count
	ping.interval = *interval
	ping.timeout = *timeout
	ping.pingSize = *packetSize
	ping.ttl = *ttl

	err = ping.start()
}
