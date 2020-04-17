package main

import (
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type Ping struct {
	count            uint // number of packets sent and PacketsReceived before stop pinging
	nPacketsSent     uint // number of packets sent
	nPacketsReceived uint // number of packets received
	interval         uint // interval between two ping requests in ms
	timeout          uint // time (ms) to wait for a ICMP reply
	nSequence        int  // sequence number
	osPid            int  // process id
	pingSize         uint // size (bytes) of ping message, including ICMP header
	ttl              int  // time-to-live
	ipv6             bool // is ipv6?

	ipaddr *net.IPAddr // ip address object
	addr   string      // string address

	RTTs       []time.Duration // all RTT times
	minRTT     time.Duration
	maxRTT     time.Duration
	avgRTT     time.Duration
	stdDevRtt  time.Duration
	total      time.Duration
	packetLoss float64
}

func PingObj(addr string) (*Ping, error) {
	ipaddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return nil, err
	}

	p := new(Ping)
	p.count = 0
	p.interval = 1000
	p.timeout = 1000
	p.nSequence = 1
	p.osPid = os.Getpid() & 0xffff
	p.addr = addr
	p.ipaddr = ipaddr
	p.pingSize = 32
	p.ttl = 64
	p.ipv6 = false

	return p, nil
}

func (p *Ping) start() error {
	CloseHandler(p)

	fmt.Printf("Pinging %s with %d bytes of data:\n", p.addr, p.pingSize)

	var conn *icmp.PacketConn
	var err error
	if p.ipv6 {
		conn, err = icmp.ListenPacket("ip6:ipv6-icmp", "::")
	} else {
		conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	// construct arbitrary message of size p.pingSize-8 to send along with 8 bytes header
	var arbitraryMsg string
	var i uint
	for i = 0; i < p.pingSize-8; i++ {
		arbitraryMsg += "g"
	}

	// setting ttl values
	if p.ipv6 {
		conn.IPv6PacketConn().SetHopLimit(p.ttl)
	} else {
		conn.IPv4PacketConn().SetTTL(p.ttl)
	}

	var typ icmp.Type
	if p.ipv6 {
		typ = ipv6.ICMPTypeEchoRequest
	} else {
		typ = ipv4.ICMPTypeEcho
	}

	// start pinging
	for {
		// constructing ICMP Echo Request message
		msg := icmp.Message{
			Type: typ,
			Code: 0,
			Body: &icmp.Echo{
				ID:   p.osPid,
				Seq:  p.nSequence,
				Data: []byte(arbitraryMsg),
			},
		}

		// serializing msg object to byte-array
		msgBytes, err := msg.Marshal(nil)
		if err != nil {
			return err
		}

		sendTime := time.Now() // start timer
		_, err = conn.WriteTo(msgBytes, p.ipaddr)
		if err != nil {
			fmt.Println("Network unreacheable")
			time.Sleep(time.Millisecond * time.Duration(p.interval))
			continue
		}
		p.nPacketsSent++
		p.nSequence++

		replyBytes := make([]byte, 1500)
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(p.timeout)))
		size, _, err := conn.ReadFrom(replyBytes)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					fmt.Println("Request timeout")
					continue
				}
			} else {
				return err
			}
		}
		elapsedTime := time.Since(sendTime) // RTT

		// Parse reply according to the protocol used
		var replyMsg *icmp.Message
		if p.ipv6 {
			replyMsg, err = icmp.ParseMessage(ipv6.ICMPTypeEchoReply.Protocol(), replyBytes)
		} else {
			replyMsg, err = icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), replyBytes)
		}
		if err != nil {
			return nil
		}

		// log appropriate response
		if replyMsg.Type == ipv4.ICMPTypeEchoReply || replyMsg.Type == ipv6.ICMPTypeEchoReply {
			switch pkt := replyMsg.Body.(type) {
			case *icmp.Echo:
				if pkt.ID == p.osPid {
					fmt.Printf("%d bytes from %s icmp_seq=%d ttl=%d rtt=%s\n", size, p.addr, p.nSequence-1, p.ttl, elapsedTime)
					p.nPacketsReceived++
					p.RTTs = append(p.RTTs, elapsedTime)
				} else {
					fmt.Printf("%s: Not our EchoReply", p.addr)
				}
			}
		} else if replyMsg.Type == ipv4.ICMPTypeDestinationUnreachable || replyMsg.Type == ipv6.ICMPTypeEchoReply {
			if _, ok := replyMsg.Body.(*icmp.DstUnreach); ok {
				fmt.Printf("%s: Destination Host Unreacheable.\n", p.addr)
			}
		} else if replyMsg.Type == ipv4.ICMPTypeTimeExceeded || replyMsg.Type == ipv6.ICMPTypeTimeExceeded {
			if _, ok := replyMsg.Body.(*icmp.TimeExceeded); ok {
				fmt.Printf("%s: TTL Exceeded.\n", p.addr)
			}
		} else {
			fmt.Println("Unexpected ICMP message type.")
		}

		if p.count > 0 && p.nPacketsReceived >= p.count {
			p.GenerateStatistics()
			return nil
		}

		time.Sleep(time.Millisecond * time.Duration(p.interval))
	}
}

// Generate statistics report at the end of the session
func (p *Ping) GenerateStatistics() {
	if p.nPacketsSent == 0 {
		p.packetLoss = float64(100)
	} else {
		p.packetLoss = float64(p.nPacketsSent-p.nPacketsReceived) / float64(p.nPacketsSent) * 100
	}

	if len(p.RTTs) > 0 {
		p.minRTT = p.RTTs[0]
		p.maxRTT = p.RTTs[0]
	}

	for i := 0; i < len(p.RTTs); i++ {
		if p.minRTT > p.RTTs[i] {
			p.minRTT = p.RTTs[i]
		}
		if p.maxRTT < p.RTTs[i] {
			p.maxRTT = p.RTTs[i]
		}
		p.total += p.RTTs[i]
	}

	if len(p.RTTs) > 0 {
		p.avgRTT = p.total / time.Duration(len(p.RTTs))
		var sqSum time.Duration
		for i := 0; i < len(p.RTTs); i++ {
			sqSum += (p.RTTs[i] - p.avgRTT) * (p.RTTs[i] - p.avgRTT)
		}
		p.stdDevRtt = time.Duration(math.Sqrt(float64(sqSum / time.Duration(len(p.RTTs)))))
	}

	fmt.Println("\n=================== STATISTICS ===================")
	fmt.Printf("Packets sent: %d, Packets received: %d, Packet loss: %.2f%%\n", p.nPacketsSent, p.nPacketsReceived, p.packetLoss)
	fmt.Printf("Max/Min/Avg/StdDev/Total RTT ==> %v/%v/%v/%v/%v\n", p.maxRTT, p.minRTT, p.avgRTT, p.stdDevRtt, p.total)
}

// handles Ctrl-C signal
func CloseHandler(p *Ping) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		p.GenerateStatistics()
		os.Exit(0)
	}()
}
