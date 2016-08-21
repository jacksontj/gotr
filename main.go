package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/jacksontj/traceroute"
)

// take a given name, and return an IP. This includes converting from ip -> string
// and potentially doing a DNS lookup
func parseCLIAddr(a string) net.IP {
	return net.ParseIP(a)
}

/*

type ProbeResponse struct {
	Success      bool
	Error        error
	Address      net.IP
	Duration     time.Duration
	TTL          int
	ResponseSize int
}

*/

func handleTracerouteProbes(opts *traceroute.TracerouteOptions, wg *sync.WaitGroup) {
	prevTTL := 0
	var prevAddr net.IP
	for {
		r, ok := <-opts.ResultChan
		// If the channel was closed, we are done
		if !ok {
			break
		}
		// if this is a new TTL, lets print the newline, and the header
		if r.TTL != prevTTL {
			fmt.Printf("\n%"+strconv.Itoa(len(strconv.Itoa(opts.MaxTTL)))+"d", r.TTL)
			prevTTL = r.TTL
			prevAddr = nil
		}
		if r.Error != nil {
			fmt.Printf(" *")
		} else {
			if prevAddr == nil {
				fmt.Printf(" %s ", r.Address.String())
				prevAddr = r.Address
			} else if !prevAddr.Equal(r.Address) {
				fmt.Println("")
				fmt.Printf(" %s ", r.Address.String())
				prevAddr = r.Address
			}
			fmt.Printf(" %.2f ms", float64(r.Duration.Nanoseconds())/float64(time.Millisecond.Nanoseconds()))
		}
	}
	fmt.Println("")
	wg.Done()
}

// TODO: add a "map the routes" all
func main() {
	sourceAddr := flag.String("srcAddr", "", "source address")
	sourcePort := flag.Int("srcPort", 0, "source port")

	dstAddr := flag.String("dstAddr", "", "destination address")
	dstPort := flag.Int("dstPort", 33434, "destination port")

	//udpProbe := flag.Bool("udp", true, "do a UDP probe")

	startingTTL := flag.Int("startingTTL", 1, "what TTL to start with")
	maxTTL := flag.Int("maxTTL", 30, "max TTL to go to")

	probeTimeout := flag.Int("probeTimeout", 1, "probe timeout (in seconds)")
	probeCount := flag.Int("probeCount", 3, "number of probes to do at each TTL")

	flag.Parse()

	// `flag` doesn't support "required" flags. To workaround this we'll simply
	// enforce our own things here
	if *dstAddr == "" {
		logrus.Fatalf("Must have a destination set")
	}

	// Create the opts to send to traceroute
	opts := traceroute.TracerouteOptions{
		SourceAddr: parseCLIAddr(*sourceAddr),
		SourcePort: *sourcePort,

		DestinationAddr: parseCLIAddr(*dstAddr),
		DestinationPort: *dstPort,

		// TODO: switch
		// enumerated value of tcp/udp/icmp
		ProbeType: traceroute.UdpProbe,

		// TTL options
		StartingTTL: *startingTTL,
		MaxTTL:      *maxTTL,

		// Probe options
		ProbeTimeout: time.Second * time.Duration(*probeTimeout),
		ProbeCount:   *probeCount,
		//ProbeWait: 0,
	}

	if opts.SourcePort == 0 {
		opts.SourcePort = opts.DestinationPort
	}
	if opts.SourceAddr == nil {
		a, err := traceroute.GetLocalIP()
		if err != nil {
			logrus.Fatalf("Unable to get source IP: %v", err)
		}
		opts.SourceAddr = a
	}

	opts.ResultChan = make(chan *traceroute.ProbeResponse)
	logrus.Debugf("Parsed Opts: %v", opts)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go handleTracerouteProbes(&opts, &wg)

	_, err := traceroute.Traceroute(&opts)
	wg.Wait()
	if err != nil {
		logrus.Fatalf("err: %v", err)
	}
}
