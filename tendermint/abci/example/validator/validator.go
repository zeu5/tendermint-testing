package validator

import (
	"strconv"
	"strings"
	"sync"

	abci "github.com/tendermint/tendermint/abci/types"
	encoding "github.com/tendermint/tendermint/crypto/encoding"
	"github.com/tendermint/tendermint/types"
)

const (
	TxOK        uint32 = 0
	TxMalformed uint32 = 1
	TxInvalid   uint32 = 2
)

type App struct {
	abci.BaseApplication
	validators map[string]*types.Validator
	vMtx       *sync.Mutex

	curUpdates []*types.Validator
	mtx        *sync.Mutex
}

func NewApplication() *App {
	return &App{
		validators: make(map[string]*types.Validator),
		vMtx:       new(sync.Mutex),
		curUpdates: make([]*types.Validator, 0),
		mtx:        new(sync.Mutex),
	}
}

func (a *App) hash() string {
	a.vMtx.Lock()
	vals := a.validators
	a.vMtx.Unlock()
	data := ""
	for k, v := range vals {
		data += k + ":" + strconv.Itoa(int(v.VotingPower)) + " "
	}
	return data
}

func (a *App) Info(req abci.RequestInfo) abci.ResponseInfo {
	return abci.ResponseInfo{Data: a.hash()}
}

func (a *App) DeliverTx(req abci.RequestDeliverTx) abci.ResponseDeliverTx {
	// Transaction is of the form "address,votingpower"
	// To update just address and voting power is enough
	// To delete a validator set power to 0
	// To add a new one address will be treated as a ED25519 public key
	tx := string(req.Tx)

	parts := strings.Split(tx, ",")
	if len(parts) != 2 {
		return abci.ResponseDeliverTx{
			Code: TxMalformed,
			Log:  "Tx not in the right format: addr,voting_power",
		}
	}
	addr, powS := parts[0], parts[1]
	pow, err := strconv.Atoi(powS)
	if err != nil {
		return abci.ResponseDeliverTx{
			Code: TxMalformed,
			Log:  "Voting power should be an interger",
		}
	}
	a.vMtx.Lock()
	_, exists := a.validators[addr]
	a.vMtx.Unlock()

	if !exists && pow == 0 {
		return abci.ResponseDeliverTx{
			Code: TxInvalid,
			Log:  "Cannot delete a non existant validator",
		}
	}

	a.vMtx.Lock()
	validator, ok := a.validators[addr]
	if ok {
		validator.VotingPower = int64(pow)
	} else {
		pubKey, err := byteToPubKey([]byte(addr))
		if err != nil {
			a.vMtx.Unlock()
			return abci.ResponseDeliverTx{
				Code: TxMalformed,
				Log:  err.Error(),
			}
		}
		validator = types.NewValidator(pubKey, int64(pow))
		a.validators[pubKey.Address().String()] = validator
	}
	a.vMtx.Unlock()

	a.mtx.Lock()
	a.curUpdates = append(a.curUpdates, validator)
	a.mtx.Unlock()

	return abci.ResponseDeliverTx{Code: TxOK}
}

func (a *App) CheckTx(req abci.RequestCheckTx) abci.ResponseCheckTx {
	tx := string(req.Tx)

	parts := strings.Split(tx, ",")
	if len(parts) != 2 {
		return abci.ResponseCheckTx{
			Code: TxMalformed,
			Log:  "Tx not in the right format: addr,voting_power",
		}
	}
	addr, powS := parts[0], parts[1]
	pow, err := strconv.Atoi(powS)
	if err != nil {
		return abci.ResponseCheckTx{
			Code: TxMalformed,
			Log:  "Voting power should be an interger",
		}
	}
	a.vMtx.Lock()
	_, exists := a.validators[addr]
	a.vMtx.Unlock()

	if !exists && pow == 0 {
		return abci.ResponseCheckTx{
			Code: TxInvalid,
			Log:  "Cannot delete a non existant validator",
		}
	}

	a.vMtx.Lock()
	_, ok := a.validators[addr]
	if !ok {
		_, err := byteToPubKey([]byte(addr))
		if err != nil {
			a.vMtx.Unlock()
			return abci.ResponseCheckTx{
				Code: TxMalformed,
				Log:  err.Error(),
			}
		}
	}
	a.vMtx.Unlock()

	return abci.ResponseCheckTx{Code: TxOK}
}

func (a *App) Query(req abci.RequestQuery) abci.ResponseQuery {
	res := abci.ResponseQuery{}
	switch req.Path {
	case "power":
		a.vMtx.Lock()
		val, ok := a.validators[req.Path]
		a.vMtx.Unlock()
		if !ok {
			res.Code = TxInvalid
			res.Log = "Validator not found"
		} else {
			res.Code = TxOK
			res.Value = []byte(strconv.Itoa(int(val.VotingPower)))
		}
	case "pubkey":
		a.vMtx.Lock()
		val, ok := a.validators[req.Path]
		a.vMtx.Unlock()
		if !ok {
			res.Code = TxInvalid
			res.Log = "Validator not found"
		} else {
			pubKey, err := pubKeyToByte(val.PubKey)
			if err != nil {
				res.Code = TxInvalid
				res.Log = err.Error()
			} else {
				res.Code = TxOK
				res.Value = pubKey
			}
		}
	default:
		res.Code = TxInvalid
		res.Log = "Expected query to be one of power,pubkey"
	}
	return res
}

func (a *App) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	valid := make([]abci.ValidatorUpdate, 0)
	for _, v := range req.Validators {
		pubKey, err := encoding.PubKeyFromProto(v.PubKey)
		if err == nil {
			valid = append(valid, v)
			a.vMtx.Lock()
			a.validators[pubKey.Address().String()] = types.NewValidator(pubKey, v.Power)
			a.vMtx.Unlock()
		}
	}
	return abci.ResponseInitChain{
		Validators: valid,
		AppHash:    []byte(a.hash()),
	}
}

func (a *App) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	a.mtx.Lock()
	vUpdates := a.curUpdates
	a.mtx.Unlock()
	updates := make([]abci.ValidatorUpdate, 0)
	for _, v := range vUpdates {
		pubKey, err := encoding.PubKeyToProto(v.PubKey)
		if err == nil {
			updates = append(updates, abci.ValidatorUpdate{
				PubKey: pubKey,
				Power:  v.VotingPower,
			})
		}
	}
	return abci.ResponseEndBlock{
		ValidatorUpdates: updates,
	}
}

func (a *App) Commit() abci.ResponseCommit {
	a.mtx.Lock()
	a.curUpdates = make([]*types.Validator, 0)
	a.mtx.Unlock()
	return abci.ResponseCommit{
		Data: []byte(a.hash()),
	}
}
