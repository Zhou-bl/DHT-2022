package chord

import "math/big"

type WrapNode struct {
	node *ChordNode
}

func (this *WrapNode) FindSuccessor(aimID *big.Int, res *string) error {
	//find aimID's successor
	return this.node.innner_find_successor(aimID, res)
}

func (this *WrapNode) GetSuccessorList(_ int, res *[successorListLength]string) error {
	return this.node.get_successor_list(res)
}

func (this *WrapNode) TransferData(preNode string, data *map[string]string) error {
	return this.node.transfer_data(preNode, data)
}

func (this *WrapNode) SubBackup(data map[string]string, _ int) error {
	return this.node.sub_backup(data)
}

func (this *WrapNode) AddBackup(data map[string]string, _ int) error {
	return this.node.add_backup(data)
}

func (this *WrapNode) SetBackup(_ int, backup *map[string]string) error {
	return this.node.set_backup(backup)
}

func (this *WrapNode) ChangePredecessor(_ int, _ int) error {
	return this.node.change_predecessor()
}

//func for hash table:
func (this *WrapNode) InsertPairInData(p KeyValuePair, _ int) error {
	return this.node.insert_pair_inData(p)
}

func (this *WrapNode) InsertPairInBackup(p KeyValuePair, _ int) error {
	return this.node.insert_pair_inBackup(p)
}

func (this *WrapNode) GetValue(key string, res *string) error {
	return this.node.get_value(key, res)
}

func (this *WrapNode) ErasePairInData(key string) error {
	return this.node.erase_pair_inData(key)
}

func (this *WrapNode) ErasePairInBackup(key string) error {
	return this.node.erase_data_inBackup(key)
}
