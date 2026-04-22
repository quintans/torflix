package gracefull

import "sync"

type Gracefull struct {
	wg      sync.WaitGroup
	mu      sync.Mutex
	closing bool
}

func New() *Gracefull {
	return &Gracefull{}
}

func (g *Gracefull) Enter() {
	g.wg.Add(1)
}

func (g *Gracefull) Leave() {
	g.wg.Done()
}

func (g *Gracefull) Shutdown() {
	g.mu.Lock()
	g.closing = true
	g.mu.Unlock()
	g.wg.Wait()
}

func (g *Gracefull) IsShuttingDown() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.closing
}
