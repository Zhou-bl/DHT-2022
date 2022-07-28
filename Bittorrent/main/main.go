package main

import (
	"fmt"
	"time"
)

var myNode dhtNode
var myIP string
var running bool
var isOnline bool
var para1, para2, para3, para4 string

const insInterval = 1500 * time.Millisecond

func init() {
	running = true
	para1 = ""
	para2 = ""
	para3 = ""
	para4 = ""
	fmt.Println("Hello, welcome to Bittorrent!")
	time.Sleep(insInterval)
	fmt.Println("Input \"cmd\" to get the commands of Bittorrent.")
}

func MyInit() {
	fmt.Println("Hello, welcome to init!")
	fmt.Println("Please input your IP to initialize:")
	fmt.Scanln(&myIP)
	fmt.Println("Programme begin to initialize...")
	time.Sleep(insInterval)
	myNode = NewNode(myIP)
	myNode.Run()
	isOnline = true
	fmt.Println("Programme finished initialing.")
	time.Sleep(insInterval)
}

func main() {
	for running {
		time.Sleep(insInterval)
		para1 = ""
		para2 = ""
		para3 = ""
		para4 = ""
		fmt.Println("Please input your command:")
		fmt.Scanln(&para1, &para2, &para3, &para4)

		if para1 == "cmd" {
			fmt.Println("Below are all commands for this app : ")
			fmt.Println("<--------------------------------------------------------------------------------------------------------------------------->")
			fmt.Println("|  init                               #To initialize your node by your ip address.                                          |")
			fmt.Println("|  bye                                #Shut down the programme and quit your node from network automaticly.                 |")
			fmt.Println("|  cmd                                #Get all commands of this application.                                                |")
			fmt.Println("|  quit                   			   #Quit your node from network.                                                         |")
			fmt.Println("|  create                 			   #Create a new network base your node.                                                 |")
			fmt.Println("|  join [IP address]      			   #Join a network by node [IP address].                                                 |")
			fmt.Println("|  upload [file path] [aim path]      #upload a file in [file path] and the .torrent will be in [ai, path].                 |")
			fmt.Println("|  download [file path] [aim path]	   #down by .torrent in [file path] into [aim path].                                     |")
			fmt.Println("<--------------------------------------------------------------------------------------------------------------------------->")
			continue
		}

		if para1 == "bye" {
			if isOnline {
				myNode.Quit()
				isOnline = false
			}
			time.Sleep(insInterval)
			fmt.Println("Bye bye~")
			running = false
			continue
		}

		if para1 == "quit" {
			if isOnline {
				myNode.Quit()
			}
			time.Sleep(insInterval)
			fmt.Println("Node", myIP, "quit from network success.")
			continue
		}

		if para1 == "create" {
			myNode.Create()
			time.Sleep(insInterval)
			fmt.Println("Node", myIP, "create a new network success.")
			continue
		}

		if para1 == "join" {
			flag := myNode.Join(para2)
			time.Sleep(insInterval)
			if flag {
				fmt.Println("Node", myIP, "join success.")
			} else {
				fmt.Println("Node", myIP, "join failed, please retry.")
			}
			continue
		}

		if para1 == "upload" {
			err := upload(para2, para3, &myNode)
			time.Sleep(insInterval)
			if err != nil {
				fmt.Println("Failed to upload the file ", para2)
				fmt.Println("Please retry.")
			}
			continue
		}

		if para1 == "download" {
			err := download(para2, para3, &myNode)
			time.Sleep(insInterval)
			if err != nil {
				fmt.Println("Fail to download the file by", para2, "into", para3)
				fmt.Println("Please retry.")
			}
			continue
		}
		fmt.Println("Illegal Command, please correct and input your command again.")
	}
}
