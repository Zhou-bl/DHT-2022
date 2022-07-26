package main

import (
	"kademlia"
)

/* In this file, you should implement function "NewNode" and
 * a struct which implements the interface "dhtNode".
 */

func NewNode(port int) dhtNode {
	// Todo: create a node and then return it.
	ptr := new(kademlia.KadNode)
	ptr.Init(port)
	return ptr
}

// Todo: implement a struct which implements the interface "dhtNode".
