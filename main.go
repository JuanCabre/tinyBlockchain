// This code was inspired/copied from Coral Health
// https://medium.com/@mycoralhealth/code-your-own-blockchain-in-less-than-200-lines-of-go-e296282bcffc
// and from Ivan Kuznetsov
// https://jeiwan.cc/posts/building-blockchain-in-go-part-2/
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/davecgh/go-spew/spew"

	"github.com/gorilla/mux"
)

// Block is the main structure of the blocks in the blockchain.
type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
	Nonce     uint64
}

// Blockchain is the current blockchain that we have.
var Blockchain []Block

// targetBits represent the dificulty of the proof of work
const targetBits = 16

// calculateHash calculates the hash of a block
func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash + string(block.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// proofOfWork Performs the proof of work and returns the hash and the Nonce
func proofOfWork(block Block) (string, uint64) {
	var hashInt big.Int
	var hash [32]byte
	var nonce uint64
	// testArray := [3]byte{0, 0, 0}

	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	for nonce = uint64(0); nonce < math.MaxUint64; nonce++ {
		record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash + string(nonce)
		hash = sha256.Sum256([]byte(record))

		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(target) == -1 {
			break
		}

		// if compare(testArray[:], hash[:dificulty]) {
		// 	break
		// }

		if nonce%100000 == 0 {
			fmt.Println(nonce)
		}
	}
	return hex.EncodeToString(hash[:]), nonce
}

// func compare(small, big []byte) bool {
// 	for i := range small {
// 		if small[i] != big[i] {
// 			return false
// 		}
// 	}
// 	return true
// }

// generateBlock generates a new block. The current time is calculated the BPM
// data is added, and the PrevHash value is added given the hash of the
// oldBlock.
func generateBlock(oldBlock Block, BPM int) (Block, error) {
	var newBlock Block

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = time.Now().String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash, newBlock.Nonce = proofOfWork(newBlock)

	return newBlock, nil
}

// isBlockValid returns true if the newBlock is valid given oldBlock
func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

// replaceChain compares the len of the new chain and the chain we have. If the
// new chain is longer, copy that instead.
func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

// run is the server that we will call to show our blockchain in a browser
func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", os.Getenv("ADDR"))

	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	err := s.ListenAndServe()

	return err
}

// makeMuxRouter will define all our handlers. We need 2. If we get a GET, we'll
// view our blockchain. If we get a POST request, we can write to it.
func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlockchain).Methods("POST")

	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

func handleWriteBlockchain(w http.ResponseWriter, r *http.Request) {
	var m Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}

	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		spew.Dump(newBlock)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}

	w.WriteHeader(code)
	w.Write(response)
}

type Message struct {
	BPM int
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Godotenv error %v", err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block{
			Index:     0,
			Timestamp: t.String(),
			BPM:       0,
			Hash:      "",
			PrevHash:  "",
			Nonce:     0}

		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()

	log.Fatal(run())
}
