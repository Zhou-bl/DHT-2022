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

	//for quit
	IsQuit         chan bool
	conRoutineFlag bool

	//network
	station *network

	rwLock sync.RWMutex

	//for communication
	successorList [successorListLength]string
	fingerTable   [fingerTableLength]string
	predecessor   string
	next          int

	//for data
	dataSet    map[string]string
	backupSet  map[string]string
	dataLock   sync.RWMutex
	backupLock sync.RWMutex
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
	this.copySuccessorList_and_init(&tmpSuccList)
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
	tmp_err = RemoteCall(succAddr, "WrapNode.ChangePredecessor", 0, 0)
	if tmp_err != nil {
		log.Errorln("In function Quit checkpre error")
	}
	tmp_err = RemoteCall(this.predecessor, "WrapNode.Stabilize", 0, 0)
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
	key   string
	value string
}

func (this *ChordNode) Put(key string, value string) bool {
	if !this.conRoutineFlag {
		//node this is sleep
		return false
	}
	var aimAddr string
	tmp_err := this.innner_find_successor(ConsistentHash(key), &aimAddr)
	if tmp_err != nil {
		log.Errorln("In function Put can not get successor of key : ", key)
		return false
	}
	p := KeyValuePair{key, value}
	tmp_err = RemoteCall(aimAddr, "WrapNode.InsertPairInData", p, 0)
	if tmp_err != nil {
		log.Errorln("In function Put insert pair error", key, value)
		return false
	}
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

}

func (this *ChordNode) Delete(key string) bool {

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
			delete(this.dataSet, key)
			(*data)[key] = value
			this.backupSet[key] = value
		}
	}
	this.backupLock.Unlock()
	this.dataLock.Unlock()
	var succAddr string
	this.find_first_online_succ(&succAddr)
	tmp_err := RemoteCall(succAddr, "WrapNode.SubBackup", *data, 0)
	if tmp_err != nil {
		log.Errorln("In function transfer_data can not sub backup")
		return nil
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

func (this *ChordNode) copySuccessorList_and_init(tmp *[successorListLength]string) {
	this.rwLock.Lock()
	this.successorList[0] = this.address
	this.fingerTable[0] = this.address
	this.predecessor = ""
	for i := 1; i < successorListLength; i++ {
		this.successorList[i] = tmp[i-1]
	}
	this.rwLock.Unlock()
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
	for key, _ := range data {
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
	(*backup) = make(map[string]string) //remember to clear the map
	for key, value := range this.dataSet {
		(*backup)[key] = value
	}
	this.dataLock.RUnlock()
	return nil
}

func (this *ChordNode) change_predecessor() error {
	if !CheckOnline(this.predecessor) {
		//In func CheckOnline involved the addr is nil
		return nil
	}
	this.rwLock.Lock()
	this.predecessor = ""
	this.rwLock.Unlock()
	//then put backup into dataset
	this.dataLock.Lock()
	this.backupLock.Lock()
	for key, value := range this.backupSet {
		this.dataSet[key] = value
	}
	this.dataLock.Unlock()
	this.backupLock.Unlock()
	//then add new back up
	var succAddr string
	tmp_err := this.find_first_online_succ(&succAddr)
	if tmp_err != nil {
		log.Errorln("In function change_predecessor can not find a succ")
		return tmp_err
	}
	tmp_err = RemoteCall(succAddr, "WrapNode.AddBackup", this.backupSet, 0)
	this.backupLock.Lock()
	this.backupSet = make(map[string]string)
	this.backupLock.Unlock()
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
			tmp_err := this.change_predecessor()
			if tmp_err != nil {
				log.Errorln("In bgMaintain change_pre error")
			}
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

func (this *ChordNode) stabilize() {
	var succAddr string
	var preAddr string
	var newSuccAddr string
	this.find_first_online_succ(&succAddr)
	tmp_err := RemoteCall(succAddr, "WrapNode.GetPredecessor", 0, &preAddr)
	if tmp_err != nil {
		log.Errorln("In stabilize get pre error")
		return
	}
	if preAddr != "" && inDur(ConsistentHash(preAddr), this.ID, ConsistentHash(succAddr), false) {
		newSuccAddr = preAddr
	}
	var tmpSuccList [successorListLength]string
	tmp_err = RemoteCall(newSuccAddr, "WrapNode.GetSuccessorList", 0, tmpSuccList)
	if tmp_err != nil {
		log.Errorln("In stabilize GetSuccessorList error")
		return
	}
	this.set_list(&tmpSuccList)
	tmp_err = RemoteCall(newSuccAddr, "WrapNode.notify", this.address, 0)
	if tmp_err != nil {
		log.Errorln("In func satbilize can not let succ notify")
	}
}

func (this *ChordNode) fix_fingerTable() {
	var aimSucc string
	tmp_err := this.innner_find_successor(getID(this.ID, this.next), &aimSucc)
	if tmp_err != nil {
		log.Errorln("In function fix_finger find successor error")
		return
	}
	this.rwLock.Lock()
	this.fingerTable[this.next] = aimSucc
	this.next++
	if this.next >= fingerTableLength {
		this.next = 1
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

func (this *ChordNode) set_list(succ *[successorListLength]string) {
	this.rwLock.Lock()
	this.successorList[0] = this.address
	this.fingerTable[0] = this.address
	for i := 1; i < successorListLength; i++ {
		this.successorList[i] = (*succ)[i-1]
	}
	this.rwLock.Unlock()
}

//func for hash table:
func (this *ChordNode) insert_pair_inData(p KeyValuePair) error {
	this.dataLock.Lock()
	this.dataSet[p.key] = p.value
	this.dataLock.Unlock()
	var succAddr string
	tmp_err := this.find_first_online_succ(&succAddr)
	if tmp_err != nil {
		log.Warningln("Can not find a succ", p)
	}
	if succAddr != "" {
		tmp_err = RemoteCall(succAddr, "WrapNode.InsertPairInBackup", p, 0)
		if tmp_err != nil {
			log.Warningln("Can not success store pair in backup", p)
		}
	}
	return nil
}

func (this *ChordNode) insert_pair_inBackup(p KeyValuePair) error {
	this.backupLock.Lock()
	this.backupSet[p.key] = p.value
	this.backupLock.Unlock()
	return nil
}
