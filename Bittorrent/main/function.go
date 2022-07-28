package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"
)

const blockLength = 262144
const maxUploadTime = 45 * time.Second
const maxDownloadTime = 100 * time.Second
const waitTime = 3 * time.Second

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

type UploadPackage struct {
	hashes [20]byte
	index  int
}

type DownloadPackage struct {
	data  []byte
	index int
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
			go uploadPieces(node, index, fileContent[l:r], ch1, ch2)
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
	time.Sleep(insInterval)
	hashString := string(hashPieces)
	err = MakeTorrentFile(fileName, aimPath, hashString)
	if err != nil {
		fmt.Println("Finished making .torrent file.")
	}
	return err
}

func uploadPieces(node *dhtNode, index int, data []byte, ch1 chan int, ch2 chan UploadPackage) {
	info := PiecesInfo{index, data}
	hashKey, err := info.Hash()
	if err != nil {
		fmt.Println("Fail to hash pieces. The error is", err)
		ch1 <- index
		return
	}
	key := fmt.Sprintf("%x", hashKey)
	flag := (*node).Put(key, string(data))
	time.Sleep(insInterval)
	if flag == false {
		fmt.Println("Fail to put pair into network")
		ch1 <- index
	}
	ch2 <- UploadPackage{hashKey, index}
	return
}

func download(torName string, aimPath string, node *dhtNode) error {
	fmt.Println("Begin to open the torrent file...")
	time.Sleep(insInterval)
	torFile, err := os.Open(torName)
	if err != nil {
		fmt.Println("Fail to open the torrent file. Please retry.")
		time.Sleep(insInterval)
		return err
	}
	bt, tmp_err := OpenTor(torFile)

	if tmp_err != nil {
		fmt.Println("Fail to unmarshal bt. Please retry.")
		return tmp_err
	}
	allInfo, err := bt.ToTorrentFile()
	if err != nil {
		fmt.Println("Fail to transform bt.")
		return err
	}
	fmt.Println("Begin to download from network...")
	time.Sleep(insInterval)
	content := make([]byte, allInfo.Length)
	blockSize := len(allInfo.PieceHashes)
	ch1 := make(chan int, blockSize+10)
	ch2 := make(chan DownloadPackage, blockSize+10)
	for i := 1; i <= blockSize; i++ {
		ch1 <- i
	}
	flag1, flag2 := true, true
	//download pieces
	for flag1 {
		select {
		case index := <-ch1:
			go downloadPieces(node, allInfo.PieceHashes[index-1], index, ch1, ch2)
		case <-time.After(waitTime):
			fmt.Println("Download finished!")
			flag1 = false
		}
		time.Sleep(100 * time.Millisecond)
	}
	//store data into the aimPath
	for flag2 {
		select {
		case pack := <-ch2:
			l := allInfo.PieceLength * (pack.index - 1)
			r := allInfo.PieceLength * pack.index
			if r > allInfo.Length {
				r = allInfo.Length
			}
			copy(content[l:r], pack.data[:])
		default:
			var dfile string
			if aimPath == "" {
				dfile = allInfo.Name
			} else {
				dfile = aimPath + "/" + allInfo.Name
			}
			err = ioutil.WriteFile(dfile, content, 0644)
			if err != nil {
				fmt.Println("Write into aimPath error, msg = ", err)
				return err
			}
			flag2 = false
		}
	}
	return nil
}

func downloadPieces(node *dhtNode, hashKey [20]byte, index int, ch1 chan int, ch2 chan DownloadPackage) {
	key := fmt.Sprintf("%x", hashKey)
	flag, value := (*node).Get(key)
	if flag == false {
		fmt.Println("Fail to get pieces", index)
		ch1 <- index
		return
	}
	//begin to verify:
	info := PiecesInfo{index, []byte(value)}
	ver, err := info.Hash()
	if err != nil {
		fmt.Println("Fail to get verify hash.")
		ch1 <- index
		return
	}
	if ver != hashKey {
		fmt.Println("Verify error!")
		ch1 <- index
		return
	}
	ch2 <- DownloadPackage{[]byte(value), index}
}
