package orchestrator

import (
	"container/list"
	"extendable_storage/internal/entities"
	"extendable_storage/internal/service/storager"
	"fmt"
	"hash/crc32"
	"sync"
)

const (
	circleSectors = 360
)

type dataKeeperContainer struct {
	state    int
	position int
	serverID string
	storage  storager.DataKeeper
}

type Circle struct {
	mu            sync.RWMutex
	serversList   *list.List
	serversMap    map[int]*list.Element
	servers       []*dataKeeperContainer
	activeServers int
}

func NewCircle() *Circle {
	return &Circle{
		servers:     make([]*dataKeeperContainer, circleSectors),
		serversMap:  make(map[int]*list.Element),
		serversList: list.New(),
	}
}

func (c *Circle) AddServer(serverID string, srv storager.DataKeeper) error {
	container := &dataKeeperContainer{
		serverID: serverID,
		storage:  srv,
	}
	switch c.activeServers {
	case 0:
		c.mu.Lock()
		container.position = 360 - 1
		el := c.serversList.PushBack(container)
		c.serversMap[container.position] = el
		c.servers[container.position] = container
		c.mu.Unlock()
	case 1:
		c.mu.Lock()
		container.position = 180 - 1
		el := c.serversList.PushFront(container)
		c.serversMap[container.position] = el
		c.servers[container.position] = container
		c.mu.Unlock()
	default:
		from, to, err := c.findExtendCandidate()
		if err != nil {
			return fmt.Errorf("error find extend candidate: %w", err)
		}
		position := (from + to) / 2
		container.position = position
		el := c.serversList.InsertBefore(container, c.serversMap[to])
		c.serversMap[position] = el
		c.servers[position] = container
	}
	c.mu.Lock()
	c.activeServers++
	c.mu.Unlock()
	return nil
}

func (c *Circle) MarkServerReady(serverID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, server := range c.servers {
		if server == nil {
			continue
		}
		if server.serverID == serverID {
			server.state = nodeStateReady
			return
		}
	}
}

// GetServerForChunk returns server which can serve chunk
// can be optimized for fewer iterations
func (c *Circle) GetServerForChunk(chunk *entities.FileChunk) (srv storager.DataKeeper, serverID string, err error) {
	chunkHash := crc32.ChecksumIEEE([]byte(chunk.String()))
	circlePosition := chunkHash % circleSectors
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i := circlePosition; i < circleSectors; i++ {
		if c.servers[i] == nil {
			continue
		}
		if c.servers[i].state == nodeStateReady {
			return c.servers[i].storage, c.servers[i].serverID, nil
		}
		// probably server not rebalanced yet. expect that next server on circle is ready and can serve
		if int(i+1) >= len(c.servers) {
			// this is the latest server in circle, so we need to check first server in circle
			for j := 0; j < len(c.servers); j++ {
				if c.servers[j] == nil {
					continue
				}
				if c.servers[j].state == nodeStateReady {
					return c.servers[j].storage, c.servers[i].serverID, nil
				}
			}
		}
	}
	return nil, "", fmt.Errorf("no servers in circle")
}

func (c *Circle) findExtendCandidate() (from, to int, err error) {
	utilization := int64(0)
	candidate := 0
	var (
		wg      sync.WaitGroup
		errList = make([]error, 0, len(c.servers))
	)

	for i := 0; i < len(c.servers); i++ {
		if c.servers[i] == nil {
			continue
		}
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			usage, err := c.servers[j].storage.GetUsage()
			if err != nil {
				c.mu.Lock()
				defer c.mu.Unlock()
				errList = append(errList, err)
				return
			}
			c.mu.Lock()
			if usage >= utilization {
				utilization = usage
				candidate = j
			}
			c.mu.Unlock()
		}(i)
	}
	wg.Wait()
	prevID := 0
	prev := c.serversMap[candidate].Prev()
	if prev != nil {
		prevID = prev.Value.(*dataKeeperContainer).position
	}
	return prevID, candidate, nil
}

func (c *Circle) PrintServerPositions() {
	prevID := 0
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i, server := range c.servers {
		if server == nil {
			continue
		}
		fmt.Printf("%s(%d): [%d-%d]\n", server.serverID, server.position, prevID, i)
		prevID = i
	}
}