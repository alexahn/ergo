package ergo

import (
	"reflect"
	"sync"
)

var pids = make(map[chan interface{}]([](chan interface{})))

// Spawn
// make Spawn atomic later using locks
func Spawn(f interface{}) (chan interface{}, *sync.WaitGroup) {
	pid := make(chan interface{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	fn := reflect.ValueOf(f)
	fnType := fn.Type()
	if fnType.Kind() != reflect.Func || fnType.NumIn() != 2 || fnType.NumOut() != 1 {
		panic("Expected a binary function returning a single value")
	}
	go func() {
		fn.Call([]reflect.Value{reflect.ValueOf(pid), reflect.ValueOf(&wg)})
		Kill(pid)
		wg.Done()
	}()
	pids[pid] = make([](chan interface{}), 0)
	return pid, &wg
}

// SpawnLink
// make SpawnLink atomic later using locks later
func SpawnLink(partner chan interface{}, f interface{}) (chan interface{}, *sync.WaitGroup) {
	// register the new channel under a list for the parent channel
	v, ok := pids[partner]
	if ok {
		pid, wg := Spawn(f)
		pids[partner] = append(v, pid)
		w, k := pids[pid]
		if k {
			pids[pid] = append(w, partner)
		} else {
			panic("Process pid does not exist")
		}
		return pid, wg
	} else {
		panic("Partner pid does not exist")
	}
}

// Send
// make Send atomic using locks later
func Send(pid chan interface{}, message interface{}) {
	_, ok := pids[pid]
	if ok {
		pid <- message
	} else {
		//fmt.Println("Process pid does not exist")
	}
}

// Receive
// make Receive atomic using locks later
func Receive(pid chan interface{}, f func(bool, interface{}) int) bool {
	select {
	case message, ok := (<-pid):
		if ok {
			f(ok, message)
			return true
		} else {
			// clean up and kill
			Kill(pid)
			f(ok, message)
			return false
		}
	}
}

// Kill
// make Kill atomic using locks later
func Kill(pid chan interface{}) bool {
	children, ok := pids[pid]
	if ok {
		close(pid)
		delete(pids, pid)
		for _, child := range children {
			Kill(child)
		}
		return true
	} else {
		return false
	}
}

// Monitor

// Noop
func Noop(interface{}) int {
	return 0
}

// ListProcesses
func ListProcesses() map[chan interface{}]([](chan interface{})) {
	return pids
}
