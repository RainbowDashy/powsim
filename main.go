package main

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	N           int = 10
	MaxHashTime int = 1000000
	Difficulty  int = 5
	Channels    []chan Message
	Workers     []Worker
	RoundID     int = 0
)

type Message struct {
	id    int
	chain []Block
}

type Worker struct {
	id             int
	Malicious      bool
	MaliciousCount int
	chain          []Block
}

type Block struct {
	hash     string
	data     string
	prevHash string
	nonce    int
}

func getGenesisBlock() Block {
	b := Block{
		data: "genesis",
	}
	b.calcHash()
	return b
}

func (w *Worker) getMessage() Message {
	var chain []Block
	chain = append(chain, w.chain...)

	return Message{
		id:    w.id,
		chain: chain,
	}
}

func (w *Worker) broadcast() {
	for i, ch := range Channels {
		msg := w.getMessage()
		if i == w.id {
			continue
		}
		ch <- msg
	}
}

func (w *Worker) lastBlock() Block {
	return w.chain[len(w.chain)-1]
}

func (w *Worker) workOneRound(wg *sync.WaitGroup) {
	defer wg.Done()
	var senderID int
	var accept bool
Loop:
	for {
		select {
		case msg, ok := <-Channels[w.id]:
			if !ok {
				break Loop
			}
			if !isChainValid(msg.chain) {
				continue
			}
			if len(msg.chain) > len(w.chain) {
				accept = true
				senderID = msg.id
				w.chain = msg.chain
			}
		default:
			break Loop
		}
	}
	if accept {
		fmt.Printf("%d,%d: accept blockchain from %d\n", RoundID, w.id, senderID)
	}
	b := Block{
		prevHash: w.lastBlock().hash,
		data:     fmt.Sprintf("This is %d at %d", w.id, time.Now().Unix()),
	}
	mineTimes := 1
	if w.Malicious {
		mineTimes = w.MaliciousCount
	}
	for i := 0; i < mineTimes; i++ {
		if b.mine() {
			w.chain = append(w.chain, b)
			if !w.Malicious {
				break
			}
			b = Block{
				prevHash: w.lastBlock().hash,
				data:     fmt.Sprintf("This is %d at %d", w.id, time.Now().Unix()),
			}
		}
	}
	w.broadcast()
}

func (b *Block) calcHash() string {
	var h []byte
	h = append(h, []byte(b.data)...)
	h = append(h, []byte(b.prevHash)...)
	h = append(h, []byte(strconv.Itoa(b.nonce))...)
	b.hash = fmt.Sprintf("%x", sha256.Sum256(h))
	return b.hash
}

func checkHash(h string) bool {
	return strings.HasPrefix(h, strings.Repeat("0", Difficulty))
}

func isChainValid(blocks []Block) bool {
	for i, block := range blocks {
		if i == 0 {
			continue
		}
		if block.prevHash != blocks[i-1].hash {
			return false
		}
		if !checkHash(block.hash) {
			return false
		}
	}
	return true
}

func (b *Block) mine() bool {
	cnt := 0
	for cnt < MaxHashTime {
		h := b.calcHash()
		if checkHash(h) {
			return true
		}
		cnt++
		b.nonce++
	}
	return false
}

func main() {
	Channels = make([]chan Message, N)
	Workers = make([]Worker, N)
	for i := 0; i < N; i++ {
		Channels[i] = make(chan Message, N)
		Workers[i] = Worker{
			id:    i,
			chain: []Block{getGenesisBlock()},
		}
	}
	Workers[0].Malicious = true
	Workers[0].MaliciousCount = 5
	fmt.Println("heelo world")
	b := Block{
		prevHash: "123",
		data:     "hi",
		nonce:    222,
	}
	fmt.Println(b.calcHash())
	fmt.Println(getGenesisBlock().hash)
	for i := 0; i < 10; i++ {
		wg := sync.WaitGroup{}
		for i := range Workers {
			worker := &Workers[i]
			wg.Add(1)
			go worker.workOneRound(&wg)
		}
		wg.Wait()
		RoundID++
	}
	for _, worker := range Workers {
		fmt.Printf("valid %v, %+v\n", isChainValid(worker.chain), worker)
	}
}
