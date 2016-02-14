# ergo
A simple attempt at simulating Erlang processes in Go. Mostly an exercise to get familiar with Go, but might someday prove to be useful for building large distributed applications.

## Spawn
The `Spawn` function accepts as input a function (`f`) that accepts as input (`pid`) `chan interface{}` and (`wg`)  `*sync.WaitGroup`. The `Spawn` function returns a `pid` of type `chan interface{}` and `wg` of type `*sync.WaitGroup`.

Messages are received via `pid`, which can be used directly, but it is recommended to use the `Send` and `Receive` functions to better manage process state.
```go
pid, wg := ergo.Spawn(func (pid chan interface{}, wg *sync.WaitGroup) int {
  fmt.Println(<-pid)
  return 0
})
pid <- "hello"
```

Wait conditions can be specified using `wg` in the body of function `f`.
```go
pid, wg := ergo.Spawn(func (pid chan interface{}, wg *sync.WaitGroup) int {
  wg.Add(1)
  go func() {
    time.Sleep(1000 * time.Millisecond)
    wg.Done()
  }()
  return 0
})
wg.Wait()
```

Closures can be used to encapsulate more variables to build richer functions.
```go
func counter(n int) func (chan interface{}, *sync.WaitGroup) int
  return func(pid chan interface{}, wg *sync.WaitGroup) int {
    fmt.Println("counter value", n)
    x := <-pid
    return counter(n + x.(int))(pid, wg)
  }
}
func main() {
  pid, wg := ergo.Spawn(counter(0))
  pid <- 10
  pid <- 20
}
```

## SpawnLink
The `SpawnLink` function is similar to `Spawn` except that it accepts an argument (`partner`) `chan interface{}`. If either the spawned process or the bound `partner` is killed, then both processes will be killed.
```go
func worker(pid chan interface{}, wg *sync.WaitGroup) int {
  time.Sleep(5000 * time.Millisecond)
  return 0
}
func main () {
  pid1, wg1 := ergo.Spawn(worker)
  pid2, wg2 := ergo.SpawnLink(pid1, worker)
  fmt.Println(ergo.ListProcesses())
  ergo.Kill(pid1)
  fmt.Println(ergo.ListProcesses())
}
```

## Receive
The `Receive` function is a wrapper around receiving messages using the `select` keyword for a `pid` channel. `Receive` accepts as arguments a `pid` and a function that accepts as input (`alive`) `bool` and (`message`) `interface{}`. The `alive` variable should be used to control recursive branching logic to terminate the process (ie `return`).
```go
pid, wg := ergo.Spawn(func (pid chan interface{}, wg *sync.WaitGroup) int {
  ergo.Receive(pid, func(alive bool, message interface{}) int {
    if (alive) {
      fmt.Println(message)
    } else {
      return 0;
    }
  })
  return 0
})
pid <- "hello"
```

## Send
The `Send` function is a wrapper around sending messages using the `pid` channel. The `Send` function will make sure killed processes do not received messages.
```go
pid, wg := ergo.Spawn(func (pid chan interface{}, wg *sync.WaitGroup) int {
  ergo.Receive(pid, func(alive bool, message interface{}) int {
    if (alive) {
      fmt.Println(message)
    } else {
      return 0;
    }
  })
  return 0
})
ergo.Send(pid, "hello")
```

## Kill
The `Kill` function is a wrapper around closing a `pid` channel, that also takes care of killing linked processes. In Go, goroutines cannot be killed from the outside, so the best we can do is organize logic such that the goroutines terminate, ie `return`. If you have a blocking infinite loop inside a process, then `Kill` will not help you.
```go
func worker(pid chan interface{}, wg *sync.WaitGroup) int {
  time.Sleep(5000 * time.Millisecond)
  return 0
}
func main () {
  pid1, wg1 := ergo.Spawn(worker)
  pid2, wg2 := ergo.SpawnLink(pid1, worker)
  fmt.Println(ergo.ListProcesses())
  ergo.Kill(pid1)
  fmt.Println(ergo.ListProcesses())
}
```

## Full Example

```go
func ping(count int, pong_pid chan interface{}) func(chan interface{}, *sync.WaitGroup) int {
	if count > 0 {
		return ping_n(count, pong_pid)
	} else {
		return ping_0(count, pong_pid)
	}
}

func ping_0(count int, pong_pid chan interface{}) func(chan interface{}, *sync.WaitGroup) int {
	return func(pid chan interface{}, wg *sync.WaitGroup) int {
		ergo.Send(pong_pid, Message{pid, "finished"})
		fmt.Println("Ping finished")
		return 0
	}
}

func ping_n(count int, pong_pid chan interface{}) func(chan interface{}, *sync.WaitGroup) int {
	return func(pid chan interface{}, wg *sync.WaitGroup) int {
		time.Sleep(1000 * time.Millisecond)
		ergo.Send(pong_pid, Message{pid, "ping"})
		ergo.Receive(pid, func(alive bool, message interface{}) int {
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
	ergo.Receive(pid, func(alive bool, message interface{}) int {
		if alive {
			switch from, token := message.(Message).from, message.(Message).token; token {
			case "finished":
				fmt.Println("Pong finished")
				return 0
			case "ping":
				fmt.Println("Pong received ping")
				ergo.Send(from, Message{pid, "pong"})
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

func main () {
  p1, wg1 := ergo.Spawn(pong)
	p2, wg2 := ergo.SpawnLink(p1, ping(3, p1))
	fmt.Println(p1, wg1, p2, wg2)
	fmt.Println(ergo.ListProcesses())
	wg1.Wait()
	wg2.Wait()
	fmt.Println(ergo.ListProcesses())
}
```

## Todo
- Make all the functions atomic
- Use `defer` to catch process panics, and propagate errors
- Add monitor function
- Add register (get and set) functions
