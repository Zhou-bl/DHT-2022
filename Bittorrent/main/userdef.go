package main

import (
	"../../Bittorrent/chord"
)

func NewNode(ip string) dhtNode {
	ptr := new(chord.ChordNode)
	ptr.Init(ip)
	return ptr
}
