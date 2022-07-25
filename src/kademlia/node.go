package kademlia

import (
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

type ClosestList struct {
	Size     int
	Standard big.Int
	List     [K]AddrType
}

type KBucketType struct {
	size   int
	bucket [K]AddrType
	mux    sync.Mutex
}

type KadNode struct {
	address        AddrType
	data           DataType
	station        *network
	conRoutineFlag bool
	routeTable     [M]KBucketType
	mux            sync.RWMutex
}

type FindNodeArg struct {
	TarID  big.Int
	Sender AddrType
}

type FindValueArg struct {
	Key    string
	Hash   big.Int
	Sender AddrType
}

type StoreArg struct {
	Key    string
	Value  string
	Sender AddrType
}

type FindValueRet struct {
	First  ClosestList
	Second string
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

func (this *KadNode) Join(ip string) bool {
	tmpAddr := AddrType{ip, Hash(ip)}
	this.kBucketUpdate(tmpAddr)
	client, tmp_err := Diag(ip)
	if tmp_err != nil {
		log.Errorln("[Diag error] in ", ip)
	} else {
		var res ClosestList
		tmp_err = client.Call("WrapperNode.FindNode", &FindNodeArg{this.address.Id, this.address}, &res)
		for i := 0; i < res.Size; i++ {
			this.kBucketUpdate(res.List[i])
		}
		client.Close()
	}

	closestlist := this.NodeLookup(&this.address.Id)

}

func (this *KadNode) Ping(addr string) bool {
	isOnline := CheckOnline(addr)
	return isOnline
}

func (this *KadNode) Put(key string, value string) bool {
	ketID := Hash(key)

}

func (this *KadNode) Get(key string) (bool, string) {

}

func (this *KadNode) FindNode(tarID *big.Int) (closestList ClosestList) {
	this.mux.RLock()
	defer this.mux.RUnlock()
	closestList.Standard = *tarID
	for i := 0; i < M; i++ {
		for j := 0; j < this.routeTable[i].size; j++ {
			if Ping(this.routeTable[i].bucket[j].Ip) == nil { // if online
				closestList.Insert(this.routeTable[i].bucket[j])
			}
		}
	}
	return
}

func (this *KadNode) FindValue(key string, hash *big.Int) FindValueRet {
	//firstly find in node "this" then find it in other nodes
	this.mux.RLock()
	defer this.mux.RUnlock()
	founded, value := this.data.GetValue(key)
	if founded {
		return FindValueRet{ClosestList{}, value}
	}
	retClosest := ClosestList{Standard: *hash}
	for i := 0; i < M; i++ {
		for j := 0; j < this.routeTable[i].size; j++ {
			if Ping(this.routeTable[i].bucket[j].Ip) == nil { //if online
				retClosest.Insert(this.routeTable[i].bucket[j])
			}
		}
	}
	return FindValueRet{retClosest, ""}
}

func (this *KadNode) NodeLookup(tarID *big.Int) (closestList ClosestList) {
	closestList = this.FindNode(tarID)
	closestList.Insert(this.address)
	isUpdate := true
	diaged := make(map[string]bool)
	for isUpdate {
		isUpdate = false
		var tmp ClosestList
		var removeList []AddrType
		for i := 0; i < closestList.Size; i++ {
			if diaged[closestList.List[i].Ip] == true {
				continue
			}
			this.kBucketUpdate(closestList.List[i])
			client, tmp_err := Diag(closestList.List[i].Ip)
			diaged[closestList.List[i].Ip] = true
			var res ClosestList
			if tmp_err != nil {
				removeList = append(removeList, closestList.List[i])
			} else {
				tmp_err = client.Call("WrapperNode.FindNode", &FindNodeArg{TarID: *tarID, Sender: this.address}, &res)
				for j := 0; j < res.Size; j++ {
					tmp.Insert(res.List[j])
				}
				client.Close()
			}
		}
		for _, key := range removeList {
			closestList.Remove(key)
		}
		for i := 0; i < tmp.Size; i++ {
			isUpdate = isUpdate || closestList.Insert(tmp.List[i])
		}
	}
	return
}

//private functions:
func (this *KadNode) reset() {
	this.conRoutineFlag = false
	this.data.data_init()
}

func (this *KadNode) kBucketUpdate(addr AddrType) {
	this.mux.Lock()
	defer this.mux.Unlock()
	if addr.Ip == "" || addr.Ip == this.address.Ip {
		return
	}
	this.routeTable[cpl(&this.address.Id, &addr.Id)].Update(addr)
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
