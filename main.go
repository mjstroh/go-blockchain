package main


//
// This program manages a blockchain and send it to a network of nodes. See link below for more details:
//
// https://medium.com/@mycoralhealth/part-2-networking-code-your-own-blockchain-in-less-than-200-lines-of-go-17fe1dad46e1
//
//
// Here are the imports
//
import (
"bufio"
"crypto/sha256"
"encoding/hex"
"encoding/json"
"io"
"log"
"net"
"os"
"strconv"
"time"

"github.com/davecgh/go-spew/spew"
"github.com/joho/godotenv"
	"sync"
)

//
// Here we define the struct of each of our blocks that will make up the blockchain
//
// Each Block contains data that will be written to the blockchain, and represents each case when you took your pulse rate
//
// Index is the position of the data record in the blockchain
// Timestamp is automatically determined and is the time the data is written
// BPM or beats per minute, is your pulse rate
// Hash is a SHA256 identifier representing this data record
// PrevHash is the SHA256 identifier of the previous record in the chain

type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
}
//
// The blockchain is a slice of block
//
var Blockchain []Block

// bcServer handles incoming concurrent Blocks
var bcServer chan []Block
var mutex = &sync.Mutex{}


//////////////////////// calculateHash //////////////////////////////////////////
//
// This function takes our Block data and creates a SHA256 hash of it
//
// It concatenates Index, Timestamp, BPM, PrevHash of the Block we provide as an
// argument and returns the SHA256 hash as a string
//
////////////////////////////////////////////////////////////////////////////////
func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
} // End calculateHash


/////////////////////////// generateBlock //////////////////////////////////////
//
// This function generates a new block in the blockchain. It uses the join of the
// previous block's hash to do that and the calculateHash function does the work
// to create the new hash.
//
////////////////////////////////////////////////////////////////////////////////
func generateBlock(oldBlock Block, BPM int) (Block, error) {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
} // End generateBlock


//////////////////////// isBlockValid //////////////////////////////////////////
//
// This function checks to see if the block is valid.
// It makes sure the blocks haven’t been tampered with.
// We do this by checking Index to make sure they’ve incremented as expected.
// We also check to make sure our PrevHash is indeed the same as the Hash of the
// previous block.
// Lastly, we want to double check the hash of the current block by running the
// calculateHash function again on the current block.
//
////////////////////////////////////////////////////////////////////////////////
func isBlockValid(newBlock, oldBlock Block) bool {

	var isValid bool = true

	if oldBlock.Index+1 != newBlock.Index {
		isValid = false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		isValid = false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		isValid = false
	}

	return isValid
} // End isBlockValid


//////////////////////////// replaceChain /////////////////////////////////////
//
// This function chooses the longest chain if there are two competing chains
//
// Rationale is this: Two well meaning nodes may simply have different chain
// lengths, so naturally the longer one will be the most up to date and have
// the latest blocks
//
///////////////////////////////////////////////////////////////////////////////
func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
} // End replaceChain


/////////////////////// handleConn ////////////////////////////////////////////
//
// This function handles the connections.
//
// It only takes in one argument, a nicely packaged net.Conn interface.
//
// Algorithm:
//
// Marshall our new blockchain into JSON so we can read it nicely
// write the new blockchain to the consoles of each of our connections
// set a timer to do this periodically so we’re not getting inundated with blockchain data.
// This is also what you see in live blockchain networks, where new blockchains are broadcast every X minutes. We’ll use 30 seconds
// Pretty print the main blockchain to the first terminal, just so we can see what’s going on and ensure blocks being
// added by different nodes are indeed being integrated into the main blockchain
//
///////////////////////////////////////////////////////////////////////////////
func handleConn(conn net.Conn) {

	defer conn.Close()

	io.WriteString(conn, "Enter a new BPM:")

	scanner := bufio.NewScanner(conn)

	// take in BPM from stdin and add it to blockchain after conducting necessary validation
	go func() {
		for scanner.Scan() {
			bpm, err := strconv.Atoi(scanner.Text())
			if err != nil {
				log.Printf("%v not a number: %v", scanner.Text(), err)
				continue
			}
			newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], bpm)
			if err != nil {
				log.Println(err)
				continue
			}
			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
				newBlockchain := append(Blockchain, newBlock)
				replaceChain(newBlockchain)
			}

			bcServer <- Blockchain
			io.WriteString(conn, "\nEnter a new BPM:")
		}
	}()

	// simulate receiving broadcast
	go func() {
		for {
			time.Sleep(30 * time.Second)
			mutex.Lock()
			output, err := json.Marshal(Blockchain)
			if err != nil {
				log.Fatal(err)
			}
			mutex.Unlock()
			io.WriteString(conn, string(output))
		}
	}()

	for _ = range bcServer {
		spew.Dump(Blockchain)
	}


}


// MAIN PROGRAM

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	bcServer = make(chan []Block)

	// create genesis block
	currentTime := time.Now()
	
	// Create the first block
	genesisBlock := Block{0, 
						currentTime.String(), 
						0, 
						"", 
						""}

	spew.Dump(genesisBlock)

	Blockchain = append(Blockchain, genesisBlock)

	// start TCP and serve TCP server
	server, err := net.Listen("tcp", ":"+os.Getenv("ADDR"))
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	// create a new connection each time we receive a connection request, and we need to serve it.
	//
	// We just made an infinite loop where we accept new connections. We want to concurrently deal
	// with each connection through a separate handler in a Go routine go handleConn(conn), so we
	// don’t clog up our for loop. This is how we can serve multiple connections concurrently.

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}

} // End MAIN PROGRAM
