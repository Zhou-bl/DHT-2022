package kademlia

import (
	"crypto/sha1"
	"math/big"
	"net"
	"time"
)

const M int = 16
const K int = 10
const Alpha int = 3
const IDLength int = 160
const RemoteTryInterval = 25 * time.Millisecond
const RePublishInterval = 100 * time.Millisecond
const UpdateInterval = 25 * time.Millisecond
const waitTime = 250 * time.Millisecond

var Mod = big.NewInt(0).Exp(big.NewInt(2), big.NewInt(int64(M)), nil)
var localAddress string

func init() {
	localAddress = GetLocalAddress()
}

func GetLocalAddress() string {
	var localaddress string

	ifaces, err := net.Interfaces()
	if err != nil {
		panic("init: failed to find network interfaces")
	}

	// find the first non-loopback interface with an IP address
	for _, elt := range ifaces {
		if elt.Flags&net.FlagLoopback == 0 && elt.Flags&net.FlagUp != 0 {
			addrs, err := elt.Addrs()
			if err != nil {
				panic("init: failed to get addresses for network interface")
			}

			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if ok {
					if ip4 := ipnet.IP.To4(); len(ip4) == net.IPv4len {
						localaddress = ip4.String()
						break
					}
				}
			}
		}
	}
	if localaddress == "" {
		panic("init: failed to find non-loopback interface with valid address on this node")
	}

	return localaddress
}

func Hash(str string) big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(str))
	var ret big.Int
	ret.SetBytes(hasher.Sum(nil))
	ret.Mod(&ret, Mod)
	return ret
}
