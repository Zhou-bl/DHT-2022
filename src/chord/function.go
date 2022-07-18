package chord

import (
	"crypto/sha1"
	"math/big"
	"net"
	"time"
)

var localAddress string
var timeCut time.Duration
var waitTime time.Duration
var mod *big.Int
var base *big.Int

type KeyValue struct {
	Key   string
	Value string
}

func init() {
	localAddress = GetLocalAddress()
	base = big.NewInt(2)
	mod = new(big.Int).Exp(base, big.NewInt(160), nil)
	timeCut = 200 * time.Millisecond
	waitTime = 250 * time.Millisecond
}

func ConsistentHash(str string) *big.Int {
	hash := sha1.New()
	hash.Write([]byte(str))
	return (&big.Int{}).SetBytes(hash.Sum(nil))
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

func inDur(x, l, r *big.Int, isClose bool) bool {

	if r.Cmp(l) > 0 {
		//isClose==true : x > l && x <= r
		//isClose==false : x > l && x < r
		if isClose {
			return (x.Cmp(l) > 0) && ((x.Cmp(r) < 0) || x.Cmp(r) == 0)
		} else {
			return (x.Cmp(l) > 0) && (x.Cmp(r) < 0)
		}
	} else {
		//isClose==true : x > l || x <= r
		//isClose==false : x > l || x < r
		if isClose {
			return (x.Cmp(l) > 0) || ((x.Cmp(r) < 0) || (x.Cmp(r) == 0))
		} else {
			return (x.Cmp(l) > 0) || (x.Cmp(r) < 0)
		}
	}
}

func getID(x *big.Int, p int) *big.Int {
	//(x + base ^ p) % mod
	y := new(big.Int).Exp(base, big.NewInt(int64(p)), nil)
	ans := new(big.Int).Add(x, y)
	res := new(big.Int).Mod(ans, mod)
	return res
}
