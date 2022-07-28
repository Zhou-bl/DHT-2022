package main

type dhtNode interface {
	Run()
	Create()
	Join(addr string) bool
	Quit()
	ForceQuit()
	Ping(addr string) bool
	Put(key string, value string) bool
	Get(key string) (bool, string)
	Delete(key string) bool
}
