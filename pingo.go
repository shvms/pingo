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
)

type Ping struct {
	count            uint
	nPacketsSent     uint
	nPacketsReceived uint
	interval         uint
	timeout          uint
	nSequence        int
	osPid            int
	pingSize         uint

	ipaddr *net.IPAddr
	addr   string

	RTTs       []time.Duration
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
	p.nSequence = 1
	p.osPid = os.Getpid() & 0xffff
	p.addr = addr
	p.ipaddr = ipaddr
	p.pingSize = 32

	return p, nil
}

func (p *Ping) setIPAddr(ipaddr *net.IPAddr) {
	p.ipaddr = ipaddr
}

func (p *Ping) setAddr(addr string) error {
	ipaddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return err
	}
	p.setIPAddr(ipaddr)
	p.addr = addr

	return nil
}

func (p *Ping) setCount(c uint) {
	p.count = c
	fmt.Println(p.count, c)
}

func (p *Ping) setInterval(i uint) {
	p.interval = i
}

func (p *Ping) setTimeout(t uint) {
	p.timeout = t
}

func (p *Ping) setPingSize(s uint) {
	if s < 8 {
		fmt.Println("Cannot be lesser than 8 bytes. Defaulting to 8.")
		s = 8
	}
	p.pingSize = s
}

func (p *Ping) start() error {
	CloseHandler(p)

	fmt.Printf("Pinging %s with %d bytes of data:\n", p.addr, p.pingSize)

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}
	defer conn.Close()

	var arbitraryMsg string
	var i uint
	for i = 0; i < p.pingSize-8; i++ {
		arbitraryMsg += "g"
	}

	for {
		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   p.osPid,
				Seq:  p.nSequence,
				Data: []byte(arbitraryMsg),
			},
		}

		msgBytes, err := msg.Marshal(nil)
		if err != nil {
			return err
		}

		sendTime := time.Now()
		_, err = conn.WriteTo(msgBytes, p.ipaddr)
		if err != nil {
			fmt.Println("Sending failed. Retrying...")
			continue
		}
		p.nPacketsSent++
		p.nSequence++

		replyBytes := make([]byte, 1500)
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 1000))
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
		elapsedTime := time.Since(sendTime)
		replyMsg, err := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), replyBytes)
		if err != nil {
			return nil
		}

		switch replyMsg.Type {
		case ipv4.ICMPTypeEchoReply:
			switch pkt := replyMsg.Body.(type) {
			case *icmp.Echo:
				if pkt.ID == p.osPid {
					fmt.Printf("%d bytes from %s icmp_seq=%d rtt=%s\n", size, p.addr, p.nSequence-1, elapsedTime)
					p.nPacketsReceived++
					p.RTTs = append(p.RTTs, elapsedTime)
				} else {
					fmt.Printf("%s: Not our EchoReply", p.addr)
				}
			}
		case ipv4.ICMPTypeDestinationUnreachable:
			fmt.Printf("%s: Destination Host Unreacheable", p.addr)
		case ipv4.ICMPTypeTimeExceeded:
			fmt.Printf("%s: Time Exceeded", p.addr)
		default:
			fmt.Println("Unexpected ICMP message type")
		}

		if p.count > 0 && p.nPacketsReceived >= p.count {
			p.GenerateStatistics()
			return nil
		}

		time.Sleep(time.Millisecond * time.Duration(p.interval))
	}
}

func (p *Ping) GenerateStatistics() {
	p.packetLoss = float64(p.nPacketsSent-p.nPacketsReceived) / float64(p.nPacketsSent) * 100

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

func CloseHandler(p *Ping) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		p.GenerateStatistics()
		os.Exit(0)
	}()
}