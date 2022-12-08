package main

import (
	"context"
	"sync"
	"time"

	"github.com/GDVFox/gostreaming/lib/go-actionlib"
	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
)

type positions struct {
	client *sdk.SandboxRestClient

	posMutex sync.RWMutex
	pos      map[string]sdk.PositionBalance
}

func newPositions(client *sdk.SandboxRestClient) *positions {
	return &positions{
		client: client,
		pos:    make(map[string]sdk.PositionBalance),
	}
}

func (p *positions) updateLoop() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		<-ticker.C
		portfolio, err := p.client.Portfolio(context.Background(), sdk.DefaultAccount)
		if err != nil {
			actionlib.WriteError(err)
			continue
		}

		postions := make(map[string]sdk.PositionBalance, 0)
		for _, pos := range portfolio.Positions {
			postions[pos.FIGI] = pos
		}

		p.posMutex.Lock()
		p.pos = postions
		p.posMutex.Unlock()
	}
}

func (p *positions) getPosition(figi string) (sdk.PositionBalance, bool) {
	p.posMutex.RLock()
	defer p.posMutex.RUnlock()

	position, ok := p.pos[figi]
	return position, ok
}
