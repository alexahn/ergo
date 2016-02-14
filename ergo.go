package ergo

import (
	"reflect"
	"sync"
)

var pids = make(map[chan interface{}]([](chan interface{})))
var mutex = &sync.Mutex{}

// Spawn
func Spawn(f interface{}) (chan interface{}, *sync.WaitGroup) {
	mutex.Lock()
	defer mutex.Unlock()
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
func SpawnLink(partner chan interface{}, f interface{}) (chan interface{}, *sync.WaitGroup) {
	// register the new channel under a list for the parent channel
	mutex.Lock()
	v, ok := pids[partner]
	if ok {
		mutex.Unlock()
		pid, wg := Spawn(f)
		mutex.Lock()
		pids[partner] = append(v, pid)
		w, k := pids[pid]
		if k {
			pids[pid] = append(w, partner)
		} else {
			panic("Process pid does not exist")
		}
		mutex.Unlock()
		return pid, wg
	} else {
		mutex.Unlock()
		panic("Partner pid does not exist")
	}
}

// Send
func Send(pid chan interface{}, message interface{}) {
	// we could use a lock here, but that might not be a good idea
	_, ok := pids[pid]
	if ok {
		pid <- message
	} else {
		//fmt.Println("Process pid does not exist")
	}
}

// Receive
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
func Kill(pid chan interface{}) bool {
	mutex.Lock()
	children, ok := pids[pid]
	if ok {
		close(pid)
		delete(pids, pid)
		mutex.Unlock()
		for _, child := range children {
			Kill(child)
		}
		return true
	} else {
		mutex.Unlock()
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
