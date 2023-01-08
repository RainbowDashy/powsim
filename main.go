package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	N           int
	MaxHashTime int
	Difficulty  int
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

func (w *Worker) receive() {
	var senderID int
	var accept bool
Loop:
	for {
		select {
		case msg, ok := <-Channels[w.id]:
			if !ok {
				break Loop
			}
			if w.Malicious {
				continue
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
}

func (w *Worker) lastBlock() Block {
	return w.chain[len(w.chain)-1]
}

func (w *Worker) workOneRound(wg *sync.WaitGroup) {
	defer wg.Done()
	w.receive()
	b := Block{
		prevHash: w.lastBlock().hash,
		data:     fmt.Sprintf("%d,%d", w.id, time.Now().UnixMicro()),
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
				data:     fmt.Sprintf("%d,%d", w.id, time.Now().UnixMicro()),
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

func printChain(blocks []Block) {
	for i, b := range blocks {
		datas := strings.Split(b.data, ",")
		if len(datas) != 2 {
			continue
		}
		fmt.Printf("Block %d, mined by %s at %s, hash %s\n", i, datas[0], datas[1], b.hash)
	}
}

func main() {
	flag.IntVar(&Difficulty, "difficulty", 5, "Hash difficulty")
	flag.IntVar(&MaxHashTime, "maxHashTime", 100000, "The maximum hash times for a miner each round")
	good := flag.Int("good", 10, "Number of good miners")
	evil := flag.Int("evil", 0, "Number of evil miners")
	round := flag.Int("round", 10, "Number of the rounds")
	flag.Parse()

	N = *good

	Workers = make([]Worker, N)
	for i := 0; i < N; i++ {
		Workers[i] = Worker{
			id:    i,
			chain: []Block{getGenesisBlock()},
		}
	}
	if *evil > 0 {
		Workers = append(Workers, Worker{
			id:             N,
			Malicious:      true,
			MaliciousCount: *evil,
			chain:          []Block{getGenesisBlock()},
		})
		N++
	}
	Channels = make([]chan Message, N)
	for i := 0; i < N; i++ {
		Channels[i] = make(chan Message, N)
	}
	for i := 0; i < *round; i++ {
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
		fmt.Printf("Worker %d reporting...\n", worker.id)
		printChain(worker.chain)
	}
}
