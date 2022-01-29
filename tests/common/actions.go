package common

import (
	"bytes"

	"github.com/netrixframework/netrix/testlib"
	"github.com/netrixframework/netrix/types"
	"github.com/netrixframework/tendermint-test/util"
)

func RecordProposal(key string) testlib.Action {
	return func(e *types.Event, c *testlib.Context) (messages []*types.Message) {
		tMsg, ok := util.GetMessageFromEvent(e, c)
		if !ok {
			return
		}
		proposalS, ok := util.GetProposalBlockIDS(tMsg)
		if !ok {
			return
		}
		c.Vars.Set(key, proposalS)
		return
	}
}

func ChangeVoteToNil() testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		if !e.IsMessageSend() {
			return []*types.Message{}
		}
		message, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(message)
		if !ok {
			return []*types.Message{}
		}
		if tMsg.Type != util.Precommit && tMsg.Type != util.Prevote {
			return []*types.Message{}
		}
		valAddr, ok := util.GetVoteValidator(tMsg)
		if !ok {
			return []*types.Message{}
		}
		var replica *types.Replica = nil
		for _, r := range c.Replicas.Iter() {
			addr, err := util.GetReplicaAddress(r)
			if err != nil {
				continue
			}
			if bytes.Equal(addr, valAddr) {
				replica = r
				break
			}
		}
		if replica == nil {
			return []*types.Message{}
		}
		newVote, err := util.ChangeVoteToNil(replica, tMsg)
		if err != nil {
			return []*types.Message{}
		}
		msgB, err := newVote.Marshal()
		if err != nil {
			return []*types.Message{}
		}
		return []*types.Message{c.NewMessage(message, msgB)}
	}
}

func ChangeVoteToProposalMessage(proposalMessageLabel string) testlib.Action {
	return func(e *types.Event, c *testlib.Context) []*types.Message {
		newProposalMessageI, ok := c.Vars.Get(proposalMessageLabel)
		if !ok {
			return []*types.Message{}
		}
		newProposalMessage, ok := newProposalMessageI.(*types.Message)
		if !ok {
			return []*types.Message{}
		}
		message, ok := c.GetMessage(e)
		if !ok {
			return []*types.Message{}
		}
		tMsg, ok := util.GetParsedMessage(newProposalMessage)
		if !ok {
			return []*types.Message{}
		}
		blockID, ok := util.GetProposalBlockID(tMsg)
		if !ok {
			return []*types.Message{}
		}
		valAddr, ok := util.GetVoteValidator(tMsg)
		if !ok {
			return []*types.Message{}
		}
		var replica *types.Replica = nil
		for _, r := range c.Replicas.Iter() {
			addr, err := util.GetReplicaAddress(r)
			if err != nil {
				continue
			}
			if bytes.Equal(addr, valAddr) {
				replica = r
				break
			}
		}
		if replica == nil {
			return []*types.Message{}
		}
		newVote, err := util.ChangeVote(replica, tMsg, blockID)
		if err != nil {
			return []*types.Message{}
		}
		msgB, err := newVote.Marshal()
		if err != nil {
			return []*types.Message{}
		}
		return []*types.Message{c.NewMessage(message, msgB)}
	}
}
