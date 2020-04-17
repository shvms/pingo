package main

import (
	"flag"
	"fmt"
	"log"
)

var usage = `
Ping implementation in Golang.
ping [-c=count] [-i=interval] [-t=timeout] [-ttl=TTL value] [-s=packetsize] [-6] host
------------------------------------------------------------------
Example:
pingo -c=7 -i=1200 google.com

count: Unsigned integer. 0 represents infinite ping.

interval: Unsigned Integer. Interval between each ICMP Echo request (in ms). Defaults to 1000ms.

timeout: Unsigned Integer. Timeout to wait for response from the host (in ms). Defaults to 1000ms.

packetsize: Unsigned Integer. Packet size, in bytes, to ping with. Defaults to 32 bytes.

TTL value: Integer. Time-to-live of the ICMP packets sent. Defaults to 64.

-6: Use IPv6.
`

func main() {
	count := flag.Uint("c", 0, "Unsigned integer. 0 represents infinite ping.")
	interval := flag.Uint("i", 1000, "Integer. Interval between each ICMP Echo request (in ms). Defaults to 1000ms.")
	timeout := flag.Uint("t", 1000, "Unsigned Integer. Timeout to wait for response from the host (in ms). Defaults to 1000ms.")
	packetSize := flag.Uint("s", 32, "Unsigned Integer. Packet size, in bytes, to ping with. Defaults to 32 bytes.")
	ttl := flag.Int("ttl", 64, "Integer. Time-to-live of the ICMP packets sent. Defaults to 64.")
	ipv6 := flag.Bool("6", false, "Use IPv6.")
	flag.Usage = func() {
		fmt.Println(usage)
	}
	flag.Parse()

	if flag.NArg() == 0 { // if no arguments left
		flag.Usage()
		return
	}

	host := flag.Arg(0)
	ping, err := PingObj(host) // initialize ping object
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	// reset values
	ping.count = *count
	ping.interval = *interval
	ping.timeout = *timeout
	ping.pingSize = *packetSize
	ping.ttl = *ttl
	ping.ipv6 = *ipv6

	err = ping.start()
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
}
