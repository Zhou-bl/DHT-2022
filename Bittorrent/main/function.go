package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"
)

const blockLength = 262144
const maxUploadTime = 45 * time.Second
const waitTime = 3 * time.Second

type UploadPackage struct {
	hashes [20]byte
	index  int
}

type DownPackage struct {
}

func upload(fileName string, aimPath string, node *dhtNode) error {
	//upload by pieces to improve the upload speed.
	fileContent, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println("Failed to open the file", fileName)
		time.Sleep(insInterval)
		return err
	}
	var blockNum int
	fileLength := len(fileContent)
	if fileLength%blockLength == 0 {
		blockNum = fileLength / blockLength
	} else {
		blockNum = fileLength/blockLength + 1
	}
	hashPieces := make([]byte, blockNum*20)
	ch1 := make(chan int, blockNum+20)
	//ch1 is a upload queue
	ch2 := make(chan UploadPackage, blockNum+20)
	for i := 1; i <= blockNum; i++ {
		ch1 <- i
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Begin to upload pieces to network...")
	t1 := time.Now()
	flag1 := true
	for flag1 {
		if time.Now().Sub(t1) > maxUploadTime {
			fmt.Println("Upload time out.")
			return errors.New("upload time out")
		}
		select {
		case index := <-ch1:
			l := (index - 1) * blockLength
			r := index * blockLength
			if r > fileLength {
				r = fileLength
			}
			go uploadPieces(blockNum, node, index, fileContent[l:r], ch1, ch2)
		case <-time.After(waitTime):
			fmt.Println("Upload finished!")
			flag1 = false
		}
	}

	flag2 := true
	for flag2 {
		select {
		case pack := <-ch2:
			index := pack.index
			copy(hashPieces[(index-1)*20:index*20], pack.hashes[:])
		default:
			fmt.Println("Begin to make .torrent file...")
			flag2 = false
		}
	}
	hashString := string(hashPieces)
	err = MakeTorrentFile(fileName, aimPath, hashString)

}
