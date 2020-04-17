# Ping implementation in Go
## Installation
* `git clone https://github.com/shvms/pingo.git`
* `cd pingo`
* `go build`
* `./pingo google.com`

## Requirements
* Reports RTT and loss percentage for sent messages. 
* Allows to set custom TTL as `-ttl=<value>` argument and displays corresponding '*Time/TTL Exceeded*' message.
* Extra features include arguments:
    * `count` for sending & receiving fixed number of packets
    * `interval` for sending ping requests at a custom interval
    * `timeout` for waiting a certain time period for a ping reply before sending new one.
    * `packetsize` for sending ICMP packets of required size (in bytes).
* Supports both IPv4 & IPv6.

## Usage
`ping [-c=count] [-i=interval] [-t=timeout] [-ttl=TTL] [-s=packetsize] [-6] host`<br>
Example:
`pingo -c=7 -i=1200 google.com`

`count`: Unsigned integer. 0 represents infinite ping.<br>
`interval`: Unsigned Integer. Interval between each ICMP Echo request (in ms). Defaults to 1000ms.<br>
`timeout`: Unsigned Integer. Timeout to wait for response from the host (in ms). Defaults to 1000ms.<br>
`packetsize`: Unsigned Integer. Packet size, in bytes, to ping with. Defaults to 32 bytes.<br>
`ttl`: Integer. Time-to-live of the ICMP packets sent. Defaults to 64.<br>
`6`: Use IPv6.