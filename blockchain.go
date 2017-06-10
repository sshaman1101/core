package blockchainApi
import (
	"fmt"
	"log"
	"math/big"
	"strings"
	"github.com/sonm-io/go-ethereum/common"
	"github.com/sonm-io/go-ethereum/ethclient"
  	"github.com/sonm-io/blockchain-api/go-build/SDT"
	"github.com/sonm-io/blockchain-api/go-build/Factory"
	"github.com/sonm-io/blockchain-api/go-build/Whitelist"
	"github.com/sonm-io/blockchain-api/go-build/HubWallet"
	"github.com/sonm-io/blockchain-api/go-build/MinWallet"
	"github.com/sonm-io/go-ethereum/accounts/abi/bind"
	"encoding/json"
	"io/ioutil"
	"os/user"
	"github.com/sonm-io/go-ethereum/core/types"
	"github.com/ipfs/go-ipfs/repo/config"
)
//----ServicesSupporters Allocation---------------------------------------------

//For rinkeby testnet
const confFile = ".rinkeby/keystore/key.json"

//create json for writing KEY
type MessageJson struct {
	Key       string     `json:"Key"`
	}
//Reading KEY
func readKey() MessageJson{
	usr, err := user.Current();
	file, err := ioutil.ReadFile(usr.HomeDir+"/"+confFile)
	if err != nil {
		fmt.Println(err)
	}
	var m MessageJson
	err = json.Unmarshal(file, &m)
	if err != nil {
		fmt.Println(err)
	}
	return m
}

type PasswordJson struct {
	Password		string	`json:"Password"`
}

//Reading user password
// ВОПРОС - Это возвращает JSON структуру или строку?
func readPwd() PasswordJson{
	usr, err := user.Current();
	// User password file JSON should be in root of home directory
	file, err := ioutil.ReadFile(usr.HomeDir+"/")
	if err != nil {
		fmt.Println(err)
	}

	var m PasswordJson
	err = json.Unmarshal(file, &m)
	if err != nil {
		fmt.Println(err)
	}
	return m
}

//--Services Getters-----------------------------
/*
	Those functions allows someone behind library gets
	conn and auth for further interaction

*/

//Establish Connection to geth IPC
// Create an IPC based RPC connection to a remote node
func cnct() {
	// NOTE there is should be wildcard but not username.
	// Try ~/.rinkeby/geth.ipc
	conn, err := ethclient.Dial("/home/cotic/.rinkeby/geth.ipc")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	//return connection obj
  	return conn
}

// Create an authorized transactor
func getAuth() {

	key:=readKey()
	pass:=readPwd()

	auth, err := bind.NewTransactor(strings.NewReader(key), pass)
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}
	return auth
}

//---Defines Binds-----------------------------------------

/*
	Those should be internal functions for internal usage (but not for sure)

*/


//Token Defines
func GlueToken(conn ethclient.Client) (*Token.SDT) {
	// Instantiate the contract
	token, err := Token.NewSDT(common.HexToAddress("0x09e4a2de83220c6f92dcfdbaa8d22fe2a4a45943"), conn)
	if err != nil {
		log.Fatalf("Failed to instantiate a Token contract: %v", err)
	}
	return token
}

func GlueFactory(conn ethclient.Client) (*Factory.Factory) {
	//Define factory
	factory, err := Factory.NewFactory(common.HexToAddress("0xfadcd0e54a6bb4c8e1130b4da6022bb29540c1a1"), conn)
	if err != nil {
		log.Fatalf("Failed to instantiate a Factory contract: %v", err)
	}
	return factory
}

func GlueWhitelist(conn ethclient.Client) (*Whitelist.Whitelist)  {
	//Define whitelist
	whitelist, err := Whitelist.NewWhitelist(common.HexToAddress("0x833865a1379b9750c8a00b407bd6e2f08e465153"), conn)
	if err != nil {
		log.Fatalf("Failed to instantiate a Whitelist contract: %v", err)
	}
	return whitelist
}

func GlueHubWallet(conn ethclient.Client, wb common.Address) (*Hubwallet.HubWallet)  {
	//Define HubWallet
	hw, err := Hubwallet.NewHubWallet(wb, conn)
	if err != nil {
		log.Fatalf("Failed to instantiate a HubWallet contract: %v", err)
	}
	return hw
}

func GlueMinWallet(conn ethclient.Client, mb common.Address) (*MinWallet.MinerWallet) {
	//Define MinerWallet
	mw, err := MinWallet.NewMinerWallet(mb, conn)
	if err != nil {
		log.Fatalf("Failed to instantiate a MinWallet contract: %v", err)
	}
	return mw
}


func main(){}
// sdt was here???
//-------------------------------------------------------------------------

//--MAIN LIBRARY-----------------------------------------------------------

/*
		HOW THIS SHOULD WORK?

		MainProgram				ThisLibrary
		A<---------------->B

	First A program wants to get Auth and Connection,
	So it will ask for getAuth and cnct functions from b
	and should store those objects inside.

	Therefore A want to interact with smart contracts functions from Blockchain,
	so it is call such functions from B with conn and auth in parameters.
*/
/*
Example
*/
func getBalance(conn ethclient.Client, mb common.Address) (*types.Transaction) {
	token:=GlueToken(conn)
	bal, err := token.BalanceOf(&bind.CallOpts{Pending: true},mb)
	if err != nil {
		log.Fatalf("Failed to request token balance: %v", err)
	}
	return bal
}


func HubTransfer(conn ethclient.Client, auth *bind.TransactOpts, wb common.Address, to common.Address,amount big.Int) (*types.Transaction)  {
	hw:=GlueHubWallet(conn,wb)
	am = big.NewInt(amount *10^17)

	tx, err := hw.Transfer(auth,to,am)
	if err != nil {
		log.Fatalf("Failed to request hub transfer: %v", err)
	}
	fmt.Println(" pending: 0x%x\n", tx.Hash())
	return tx
}

func WhiteListCall (conn ethclient.Client,)(){
	wl:= GlueWhitelist(conn)
	dp, err := wl.WhitelistCaller()
	if err != nil{
		log.Fatalf("Failed whiteList: %v", err)
	}
	return dp
}
func WhiteListTransactor (conn ethclient.Client,)(){
	wl:= GlueWhitelist(conn)
	dp, err := wl.WhitelistTransactor()
	if err != nil{
		log.Fatalf("Failed whiteList: %v", err)
	}
	return dp
}
func CreateMiner (conn ethclient.Client)(){
	factory := GlueFactory(conn)
	rc, err := factory.FactoryTransactor.CreateMiner()
	if err!= nil{ log.Fatal("Failed to create miner")}
	return  rc

}
func RegisterMiner (conn ethclient.Client)(){
	rm := GlueWhitelist(conn)
	dp, err := rm.WhitelistTransactor.RegisterMin()
	if err!= nil {
		log.Fatal("Failed register miner")
	}
	return dp
}
func UnRegisterMiner (conn ethclient.Client)(){
	unreg := GlueWhitelist(conn)
	ur, err := unreg.WhitelistTransactor.UnRegisterMiner()
	if err!= nil{
		log.Fatal("Failed unregistered miner")
	}
	return ur
}
func CreateHub (conn ethclient.Client)(){
	factory := GlueFactory(conn)
	chub, err := factory.FactoryTransactor.CreateHub()
	if err!= nil{ log.Fatal("Failed to create hub")}
	return  chub

}

func RegisterHub (conn ethclient.Client)(){
	rm := GlueWhitelist(conn)
	rhub, err := rm.WhitelistTransactor.RegisterHub()
	if err!= nil {
		log.Fatal("Failed redister hub")
	}
	return rhub
}
func UnRegisterHub (conn ethclient.Client)(){
	unreg := GlueWhitelist(conn)
	ur, err := unreg.WhitelistTransactor.UnRegisterHub()
	if err!= nil{
		log.Fatal("Failed unregistered hub")
	}
	return ur
}






