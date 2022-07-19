package chord

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/big"
	"sync"
	"time"
)

const successorListLength int = 5
const fingerTableLength int = 160

type ChordNode struct {
	//address & ID
	address string
	ID      *big.Int

	//network
	station *network

	//for quit
	IsQuit         chan bool
	conRoutineFlag bool

	//for communication
	successorList [successorListLength]string
	fingerTable   [fingerTableLength]string
	predecessor   string
	rwLock        sync.RWMutex

	//for data
	dataSet    map[string]string
	backupSet  map[string]string
	dataLock   sync.RWMutex
	backupLock sync.RWMutex

	next int
}

func (this *ChordNode) Init(port int) {
	this.address = fmt.Sprintf("%s:%d", localAddress, port)
	this.ID = ConsistentHash(this.address)
	this.conRoutineFlag = false
	this.reset()
}

func (this *ChordNode) Run() {
	this.station = new(network)
	//create a station for this node.
	tmp_err := this.station.Init(this.address, this)
	if tmp_err != nil {
		log.Errorln("Run error in ", this.address)
		return
	}
	log.Infoln("Run success in ", this.address)
	this.conRoutineFlag = true //after joining in the network always run stablize and fix_finger.
	this.next = 1
}

func (this *ChordNode) Create() {
	this.predecessor = ""
	this.fingerTable[0] = this.address
	this.successorList[0] = this.address
	this.bgMaintain()
}

func (this *ChordNode) Join(addr string) bool {
	//Node "this" join in a network by node "addr"
	//function join just indicates the existence of the node
	isOnline := CheckOnline(addr)
	if !isOnline {
		log.Errorln("Node Join Error : Node is not online!")
		return false
	}
	var succAddr string
	//Call node "addr" to find the successor of node "this"
	tmp_err := RemoteCall(addr, "WrapNode.FindSuccessor", this.ID, &succAddr)
	if tmp_err != nil {
		log.Errorln("In function Join FindSuccessor remote call error")
		return false
	}
	var tmpSuccList [successorListLength]string
	//Call node successor to get successor list of node "succAddr"
	//the result is in "tmpSuccList"
	tmp_err = RemoteCall(succAddr, "WrapNode.GetSuccessorList", 0, &tmpSuccList)
	if tmp_err != nil {
		log.Errorln("In function Join GetSuccessor remote call error")
		return false
	}
	log.Infoln("Join node success! The address is", this.address)
	this.rwLock.Lock()
	this.predecessor = ""
	this.successorList[0] = succAddr
	this.fingerTable[0] = succAddr
	for i := 1; i < successorListLength; i++ {
		this.successorList[i] = tmpSuccList[i-1]
	}
	this.rwLock.Unlock()
	//Transfer data from succAddr to this
	tmp_err = RemoteCall(succAddr, "WrapNode.TransferData", this.address, &this.dataSet)
	if tmp_err != nil {
		log.Errorln("In function Join TransferDate error")
		return false
	}
	this.bgMaintain()
	return true
}

func (this *ChordNode) Quit() {
	if !this.conRoutineFlag {
		return
	}
	tmp_err := this.station.ShutDown()
	if tmp_err != nil {
		log.Errorln("In function Quit station shutDown error")
	}
	this.rwLock.Lock()
	this.conRoutineFlag = false
	this.rwLock.Unlock()
	var succAddr string
	this.find_first_online_succ(&succAddr)
	var o string
	tmp_err = RemoteCall(succAddr, "WrapNode.ChangePredecessor", 0, &o)
	if tmp_err != nil {
		log.Errorln("In function Quit checkpre error")
	}
	tmp_err = RemoteCall(this.predecessor, "WrapNode.Stabilize", 0, &o)
	if tmp_err != nil {
		log.Errorln("In function Quit stabilize error")
	}
	this.reset()
}

func (this *ChordNode) ForceQuit() {
	//compare to Quit, ForceQuit can't notify other nodes
	if !this.conRoutineFlag {
		return
	}
	tmp_err := this.station.ShutDown()
	if tmp_err != nil {
		log.Errorln("In function ForceQuit station shutDown error")
	}

	this.rwLock.Lock()
	this.conRoutineFlag = false
	this.rwLock.Unlock()

	this.reset()
}

func (this *ChordNode) Ping(addr string) bool {
	isOnline := CheckOnline(addr)
	return isOnline
}

type KeyValuePair struct {
	Key   string
	Value string
}

func (this *ChordNode) Put(key string, value string) bool {
	if !this.conRoutineFlag {
		//node this is sleep
		return false
	}

	//fmt.Println("Hello this is in function Put")

	var aimAddr string
	tmp_err := this.innner_find_successor(ConsistentHash(key), &aimAddr)
	if tmp_err != nil {
		log.Errorln("In function Put can not get successor of key : ", key)
		return false
	}
	p := KeyValuePair{key, value}
	var o string
	tmp_err = RemoteCall(aimAddr, "WrapNode.InsertPairInData", p, &o)
	if tmp_err != nil {
		log.Errorln("In function Put insert pair error", key, value)
		return false
	}
	log.Infoln("Put pair success", key, value, aimAddr)
	return true
}

func (this *ChordNode) Get(key string) (bool, string) {
	if !this.conRoutineFlag {
		return false, ""
	}
	var aimAddr string
	tmp_err := this.innner_find_successor(ConsistentHash(key), &aimAddr)
	if tmp_err != nil {
		log.Errorln("Can not find the aim node for key : ", key)
		return false, ""
	}
	var res string
	tmp_err = RemoteCall(aimAddr, "WrapNode.GetValue", key, &res)
	if tmp_err != nil {
		log.Errorln("Get value error", key)
		return false, ""
	}
	return true, res
}

func (this *ChordNode) Delete(key string) bool {
	if !this.conRoutineFlag {
		return false
	}
	var aimAddr string
	tmp_err := this.innner_find_successor(ConsistentHash(key), &aimAddr)
	if tmp_err != nil {
		log.Errorln("In function delete find successor error", key)
		return false
	}
	var o string
	tmp_err = RemoteCall(aimAddr, "WrapNode.ErasePairInData", key, &o)
	if tmp_err != nil {
		log.Errorln("In function delete can not erase key")
		return false
	}
	return true
}

//private functions:

func (this *ChordNode) innner_find_successor(aimID *big.Int, res *string) error {
	//use the first joined Node to find aimID's successor
	var firstNode string
	tmp_err := this.find_first_online_succ(&firstNode)
	if tmp_err != nil {
		log.Errorln("In function innner_find_successor can not get the first firstNode")
		return tmp_err
	}
	if inDur(aimID, this.ID, ConsistentHash(firstNode), true) {
		*res = firstNode
		return nil
	}
	firstPre := this.first_pre_node(aimID)
	return RemoteCall(firstPre, "WrapNode.FindSuccessor", aimID, res)
}

func (this *ChordNode) get_successor_list(res *[successorListLength]string) error {
	//need use read lock
	this.rwLock.RLock()
	*res = this.successorList
	this.rwLock.RUnlock()
	return nil
}

func (this *ChordNode) get_predecessor(res *string) error {
	this.rwLock.RLock()
	*res = this.predecessor
	this.rwLock.RUnlock()
	return nil
}

func (this *ChordNode) transfer_data(preNode string, data *map[string]string) error {
	this.dataLock.Lock()
	this.backupLock.Lock()
	this.backupSet = make(map[string]string)
	for key, value := range this.dataSet {
		if !inDur(ConsistentHash(key), ConsistentHash(preNode), this.ID, true) {
			(*data)[key] = value
			this.backupSet[key] = value
			delete(this.dataSet, key)
		}
	}
	this.backupLock.Unlock()
	this.dataLock.Unlock()
	var succAddr string
	this.find_first_online_succ(&succAddr)
	var o string
	tmp_err := RemoteCall(succAddr, "WrapNode.SubBackup", *data, &o)
	if tmp_err != nil {
		log.Errorln("In function transfer_data can not sub backup")
	}
	this.rwLock.Lock()
	this.predecessor = preNode
	this.rwLock.Unlock()
	return nil
}

func (this *ChordNode) find_first_online_succ(res *string) error {
	for i := 0; i < successorListLength; i++ {
		flag := CheckOnline(this.successorList[i])
		if flag == true {
			*res = this.successorList[i]
			return nil
		}
	}
	res_error := errors.New("Can not find online successor")
	return res_error
}

func (this *ChordNode) first_pre_node(aimID *big.Int) string {
	for i := fingerTableLength - 1; i >= 0; i-- {
		if this.fingerTable[i] != "" && CheckOnline(this.fingerTable[i]) {
			if inDur(ConsistentHash(this.fingerTable[i]), this.ID, aimID, false) {
				return this.fingerTable[i]
			}
		}
	}
	var res string
	tmp_err := this.find_first_online_succ(&res)
	if tmp_err != nil {
		log.Errorln("In function first_pre_node can not find successor")
		return ""
	}
	return res
}

func (this *ChordNode) reset() {
	this.dataLock.Lock()
	this.dataSet = make(map[string]string)
	this.dataLock.Unlock()
	this.backupLock.Lock()
	this.backupSet = make(map[string]string)
	this.backupLock.Unlock()
	this.rwLock.Lock()
	this.IsQuit = make(chan bool, 2)
	this.next = 1
	this.rwLock.Unlock()
}

func (this *ChordNode) sub_backup(data map[string]string) error {
	this.backupLock.Lock()
	for key := range data {
		delete(this.backupSet, key)
	}
	this.backupLock.Unlock()
	return nil
}

func (this *ChordNode) add_backup(data map[string]string) error {
	this.backupLock.Lock()
	for key, value := range data {
		this.backupSet[key] = value
	}
	this.backupLock.Unlock()
	return nil
}

func (this *ChordNode) set_backup(backup *map[string]string) error {
	this.dataLock.RLock()
	*backup = make(map[string]string) //remember to clear the map
	for key, value := range this.dataSet {
		(*backup)[key] = value
	}
	this.dataLock.RUnlock()
	return nil
}

func (this *ChordNode) change_predecessor() error {
	if this.predecessor != "" && !CheckOnline(this.predecessor) {
		this.rwLock.Lock()
		this.predecessor = ""
		this.rwLock.Unlock()
		//then put backup into dataset
		this.dataLock.Lock()
		this.backupLock.RLock()
		for key, value := range this.backupSet {
			this.dataSet[key] = value
		}
		this.dataLock.Unlock()
		this.backupLock.RUnlock()
		//then add new back up
		var succAddr string
		tmp_err := this.find_first_online_succ(&succAddr)
		if tmp_err != nil {
			log.Errorln("In function change_predecessor can not find a succ")
			return tmp_err
		}
		var o string
		tmp_err = RemoteCall(succAddr, "WrapNode.AddBackup", this.backupSet, &o)
		this.backupLock.Lock()
		this.backupSet = make(map[string]string)
		this.backupLock.Unlock()
	}
	return nil
}

func (this *ChordNode) bgMaintain() {
	//this func always run three functions below
	//background maintain for finger_table & predecessor & stabilize
	//first is stabilize
	go func() {
		for this.conRoutineFlag {
			this.stabilize()
			time.Sleep(timeCut)
		}
	}()

	go func() {
		for this.conRoutineFlag {
			this.change_predecessor()
			time.Sleep(timeCut)
		}
	}()

	go func() {
		for this.conRoutineFlag {
			this.fix_fingerTable()
			time.Sleep(timeCut)
		}
	}()
}

func (this *ChordNode) stabilize() error {
	var succAddr string
	var preAddr string
	this.find_first_online_succ(&succAddr)
	log.Infoln("In stabilize find first online succ : ", this.address, succAddr)
	tmp_err := RemoteCall(succAddr, "WrapNode.GetPredecessor", 0, &preAddr)
	if tmp_err != nil {
		log.Errorln("In stabilize get pre error")
		return tmp_err
	}
	if preAddr != "" && inDur(ConsistentHash(preAddr), this.ID, ConsistentHash(succAddr), false) {
		succAddr = preAddr
	}
	var tmpSuccList [successorListLength]string
	tmp_err = RemoteCall(succAddr, "WrapNode.GetSuccessorList", 0, &tmpSuccList)
	if tmp_err != nil {
		log.Errorln("In stabilize GetSuccessorList error, because of : ", tmp_err, "this addr: ", this.address, "aimAddr : ", succAddr)
		return tmp_err
	}
	this.rwLock.Lock()
	this.successorList[0] = succAddr
	this.fingerTable[0] = succAddr
	for i := 1; i < successorListLength; i++ {
		this.successorList[i] = tmpSuccList[i-1]
	}
	this.rwLock.Unlock()
	var o string
	tmp_err = RemoteCall(succAddr, "WrapNode.Notify", this.address, &o)
	if tmp_err != nil {
		log.Errorln("In func satbilize can not let succ notify")
	}
	return nil
}

func (this *ChordNode) fix_fingerTable() {
	//change one item for each run this function
	var aimSucc string
	tmp_err := this.innner_find_successor(getID(this.ID, this.next), &aimSucc)
	if tmp_err != nil {
		log.Errorln("In function fix_finger find successor error")
		return
	}
	this.rwLock.Lock()
	this.fingerTable[this.next] = aimSucc
	//change next
	this.next = (this.next + 1) % fingerTableLength
	if this.next == 0 {
		this.next++
	}
	this.rwLock.Unlock()
}

func (this *ChordNode) notify(preNode string) error {
	if this.predecessor == "" || inDur(ConsistentHash(preNode), ConsistentHash(this.predecessor), this.ID, false) {
		this.rwLock.Lock()
		this.predecessor = preNode
		this.rwLock.Unlock()
		tmp_err := RemoteCall(this.predecessor, "WrapNode.SetBackup", 0, &this.backupSet)
		if tmp_err != nil {
			log.Errorln("In function notify can not set backup data")
			return tmp_err
		}
	}
	return nil
}

//func for hash table:
func (this *ChordNode) insert_pair_inData(p KeyValuePair) error {
	this.dataLock.Lock()
	this.dataSet[p.Key] = p.Value
	this.dataLock.Unlock()
	var succAddr string
	tmp_err := this.find_first_online_succ(&succAddr)
	if tmp_err != nil {
		log.Warningln("Can not find a succ", p)
	}
	if succAddr != "" {
		var o string
		tmp_err = RemoteCall(succAddr, "WrapNode.InsertPairInBackup", p, &o)
		if tmp_err != nil {
			log.Warningln("Can not success store pair in backup", p)
		}
	}
	return nil
}

func (this *ChordNode) insert_pair_inBackup(p KeyValuePair) error {
	this.backupLock.Lock()
	this.backupSet[p.Key] = p.Value
	this.backupLock.Unlock()
	return nil
}

func (this *ChordNode) get_value(key string, res *string) error {
	this.dataLock.RLock()
	value, flag := this.dataSet[key]
	this.dataLock.RUnlock()
	if flag {
		*res = value
		return nil
	} else {
		*res = ""
		return errors.New("Nil value")
	}
}

func (this *ChordNode) erase_pair_inData(key string) error {
	this.dataLock.Lock()
	_, ok := this.dataSet[key]
	if ok {
		delete(this.dataSet, key)
	}
	this.dataLock.Unlock()
	if !ok {
		//delete error
		log.Errorln("In erase_pair_inData delete not exit", key)
		return errors.New("Delete failed!")
	} else {
		var succAddr string
		tmp_err := this.innner_find_successor(ConsistentHash(key), &succAddr)
		if tmp_err != nil {
			log.Warningln("In erase_pair_inData delete pair in backup error")
		}
		if succAddr != "" && succAddr != this.address {
			var o string
			tmp_err = RemoteCall(succAddr, "WrapNode.ErasePairInBackup", key, &o)
			if tmp_err != nil {
				log.Warningln("Can not delete pair in backup")
			}
		}
		return nil
	}
}

func (this *ChordNode) erase_pair_inBackup(key string) error {
	this.backupLock.Lock()
	_, ok := this.backupSet[key]
	if ok {
		delete(this.backupSet, key)
	}
	this.backupLock.Unlock()
	if ok {
		return nil
	} else {
		return errors.New("Not found key backup")
	}
}
