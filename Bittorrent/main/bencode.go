package main

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/jackpal/bencode-go"
	"io"
	"os"
	"time"
)

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     BencodeInfo `bencode:"info"`
}

type PiecesInfo struct {
	Index int
	Data  []byte
}

func OpenTor(f io.Reader) (*bencodeTorrent, error) {
	res := bencodeTorrent{}
	err := bencode.Unmarshal(f, &res)
	if err != nil {
		fmt.Println("Fail to unmarshl the torrent file.")
		return nil, err
	}
	return &res, nil
}

func (this *PiecesInfo) Hash() ([20]byte, error) {
	var buffer bytes.Buffer
	err := bencode.Marshal(&buffer, *this)
	if err != nil {
		return [20]byte{}, err
	}
	res := sha1.Sum(buffer.Bytes())
	return res, nil
}

func (this *BencodeInfo) InfoHash() ([20]byte, error) {
	var buffer bytes.Buffer
	err := bencode.Marshal(&buffer, *this)
	if err != nil {
		return [20]byte{}, err
	}
	res := sha1.Sum(buffer.Bytes())
	return res, nil
}

func (this *BencodeInfo) splitHash() ([][20]byte, error) {
	hlen := 20
	buffer := []byte(this.Pieces)
	//get the information of bencodeinfo
	if len(buffer)%hlen != 0 {
		fmt.Println("BencodeInfo form is illegal!")
		return nil, errors.New("illegal form of bencodeInfo")
	}
	num := len(buffer) / hlen
	res := make([][20]byte, num)
	for i := 0; i < num; i++ {
		copy(res[i][:], buffer[i*hlen:(i+1)*hlen])
	}
	return res, nil
}

func (this *bencodeTorrent) ToTorrentFile() (TorrentFile, error) {
	infoHash, err := this.Info.InfoHash()
	if err != nil {
		return TorrentFile{}, err
	}
	hashPieces, tmp_err := this.Info.splitHash()
	if tmp_err != nil {
		return TorrentFile{}, tmp_err
	}
	res := TorrentFile{
		Announce:    this.Announce,
		InfoHash:    infoHash,
		PieceHashes: hashPieces,
		PieceLength: this.Info.PieceLength,
		Length:      this.Info.Length,
		Name:        this.Info.Name,
	}
	return res, nil
}

type BencodeInfo struct {
	Pieces string `bencode:"pieces"`
	//Pieces is the whole hash.
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

func MakeTorrentFile(fileName string, aimPath string, hashes string) error {
	fileState, err := os.Stat(fileName)
	if err != nil {
		fmt.Println("Failed to get file's state.")
		return err
	}
	tmp := bencodeTorrent{
		Info: BencodeInfo{
			Pieces: hashes, PieceLength: blockLength,
			Length: int(fileState.Size()), Name: fileState.Name(),
		},
	}
	var f *os.File
	var realName string
	if aimPath == "" {
		realName = fileState.Name() + ".torrent"
	} else {
		realName = aimPath + "/" + fileState.Name() + ".torrent"
	}
	f, _ = os.Create(realName)
	err = bencode.Marshal(f, tmp)
	if err != nil {
		fmt.Println("Fail to marshal the info.")
		return err
	}
	fmt.Println("Make torrent file successfully!")
	time.Sleep(insInterval)
	fmt.Println("The torrent file is in", fileState.Name()+".torrent")
	return nil
}
