package ergo

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// because we can't overload with pattern matching, we can break the function into separate pieces
func ping(count int, pong_pid chan interface{}) func(chan interface{}, *sync.WaitGroup) int {
	if count > 0 {
		return ping_n(count, pong_pid)
	} else {
		return ping_0(count, pong_pid)
	}
}

func ping_0(count int, pong_pid chan interface{}) func(chan interface{}, *sync.WaitGroup) int {
	return func(pid chan interface{}, wg *sync.WaitGroup) int {
		Send(pong_pid, Message{pid, "finished"})
		fmt.Println("Ping finished")
		return 0
	}
}

func ping_n(count int, pong_pid chan interface{}) func(chan interface{}, *sync.WaitGroup) int {
	return func(pid chan interface{}, wg *sync.WaitGroup) int {
		time.Sleep(1000 * time.Millisecond)
		Send(pong_pid, Message{pid, "ping"})
		Receive(pid, func(alive bool, message interface{}) int {
			if alive {
				switch _, token := message.(Message).from, message.(Message).token; token {
				case "pong":
					fmt.Println("Ping received pong")
					return ping(count-1, pong_pid)(pid, wg)
				}
			}
			return 0
		})
		return 0
	}
}

func pong(pid chan interface{}, wg *sync.WaitGroup) int {
	Receive(pid, func(alive bool, message interface{}) int {
		if alive {
			switch from, token := message.(Message).from, message.(Message).token; token {
			case "finished":
				fmt.Println("Pong finished")
				return 0
			case "ping":
				fmt.Println("Pong received ping")
				Send(from, Message{pid, "pong"})
				return pong(pid, wg)
			}
		}
		return 0
	})
	return 0
}

type Message struct {
	from  chan interface{}
	token string
}

func TestSpawn(t *testing.T) {
	pid := make(chan interface{})
	p1, wg1 := Spawn(pong)
	p2, wg2 := Spawn(ping(3, p1))
	fmt.Println(pid, p1, wg1, p2, wg2)
	fmt.Println(ListProcesses())
	wg1.Wait()
	wg2.Wait()
	fmt.Println(ListProcesses())
}

func TestSpawnLinkParent(t *testing.T) {
	pid := make(chan interface{})
	p1, wg1 := Spawn(pong)
	p2, wg2 := SpawnLink(p1, ping(3, p1))
	fmt.Println(pid, p1, wg1, p2, wg2)
	fmt.Println(ListProcesses())
	time.Sleep(2000 * time.Millisecond)
	Kill(p1)
	wg1.Wait()
	wg2.Wait()
	fmt.Println(ListProcesses())
}

func TestSpawnLinkChild(t *testing.T) {
	pid := make(chan interface{})
	p1, wg1 := Spawn(pong)
	p2, wg2 := SpawnLink(p1, ping(3, p1))
	fmt.Println(pid, p1, wg1, p2, wg2)
	fmt.Println(ListProcesses())
	time.Sleep(2000 * time.Millisecond)
	Kill(p2)
	wg1.Wait()
	wg2.Wait()
	fmt.Println(ListProcesses())
}
