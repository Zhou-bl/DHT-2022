package kademlia

import (
	"container/list"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/big"
	"sync"
	"time"
)

var localAddres string

type Pair struct {
	Key   string
	Value string
}

type AddrType struct {
	Ip string
	Id big.Int
}

func (this *AddrType) addr_init(port int) {
	this.Ip = fmt.Sprintf("%s:%d", localAddress, port)
	this.Id = Hash(this.Ip)
}

type DataType struct {
	hashMap       map[string]string
	validTime     map[string]time.Time
	republishTime map[string]time.Time
	lock          sync.RWMutex
}

func (this *DataType) data_init() {
	this.hashMap = make(map[string]string)
	this.validTime = make(map[string]time.Time)
	this.republishTime = make(map[string]time.Time)
}

type RoutingTable struct {
	nodeAddr       AddrType
	rwLock         sync.Mutex
	buckets        [IDLength]*list.List
	refreshIndex   int
	refreshTimeSet [IDLength]time.Time
}

type KadNode struct {
	address        AddrType
	data           DataType
	station        *network
	conRoutineFlag bool
	table          RoutingTable
}

func (this *KadNode) Init(port int) {
	this.address.addr_init(port)
	this.reset()
}

func (this *KadNode) Run() {
	this.station = new(network)
	tmp_err := this.station.Init(this.address.Ip, this)
	if tmp_err != nil {
		log.Errorln("[Run error] can not init station, the node IP is : ", this.address.Ip)
		return
	} else {
		log.Infoln("[Run success] in : ", this.address.Ip)
		this.conRoutineFlag = true
		this.bgMaintain()
	}
}

func (this *KadNode) Join() bool {

}

func (this *KadNode) Ping(addr string) bool {
	isOnline := CheckOnline(addr)
	return isOnline
}

func (this *KadNode) Put(key string, value string) bool {

}

func (this *KadNode) Get(key string) (bool, string) {

}

//private functions:
func (this *KadNode) reset() {
	this.conRoutineFlag = false
	this.data.data_init()
}

func (this *KadNode) bgMaintain() {
	go func() {
		for this.conRoutineFlag {

		}
	}()

	go func() {
		for this.conRoutineFlag {

		}
	}()
}

//none used function
func (this *KadNode) Create() {
	return
}
func (this *KadNode) Quit() {
	this.station.ShutDown()
	this.reset()
}
func (this *KadNode) ForceQuit() {
	this.station.ShutDown()
	this.reset()
}

func (this *KadNode) Delete(key string) bool {
	return true
}
