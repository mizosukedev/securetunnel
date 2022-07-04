package server

import (
	"fmt"
	"sync"
)

// MemoryStore is an structure that implements Store interface.
// This structure stores information in memory.
// This structure cannot be used to spread across multiple servers.
type MemoryStore struct {
	tunnelTable     *dataTable[string, Tunnel]           // map[tunnelID]Tunnel
	connectionTable *dataTable[ConnectionID, Connection] // map[connectionID]Connection
}

func (store *MemoryStore) Init() error {

	store.tunnelTable = &dataTable[string, Tunnel]{
		name:    "tunnel",
		rwMutex: &sync.RWMutex{},
		dataMap: map[string]Tunnel{},
	}

	store.connectionTable = &dataTable[ConnectionID, Connection]{
		name:    "connection",
		rwMutex: &sync.RWMutex{},
		dataMap: map[ConnectionID]Connection{},
	}

	return nil
}

func (store *MemoryStore) GetTunnel(tunnelID string) (Tunnel, error) {

	tunnel, ok := store.tunnelTable.getWithKey(tunnelID)
	if !ok {
		return Tunnel{}, fmt.Errorf("tunnel not found id=%s", tunnelID)
	}

	return tunnel, nil
}

func (store *MemoryStore) GetTunnels(options TunnelSearchOptions) ([]Tunnel, error) {

	tunnels := store.tunnelTable.list(nil)
	return tunnels, nil
}

func (store *MemoryStore) AddTunnel(tunnel Tunnel) error {

	err := store.tunnelTable.add(tunnel)
	return err
}

func (store *MemoryStore) UpdateTunnel(tunnel Tunnel) error {

	err := store.tunnelTable.update(tunnel)
	return err
}

func (store *MemoryStore) DeleteTunnel(tunnelID string) error {

	store.tunnelTable.delete(tunnelID)
	return nil
}

func (store *MemoryStore) GetConnection(connectionID ConnectionID) (Connection, error) {

	connection, ok := store.connectionTable.getWithKey(connectionID)
	if !ok {
		return Connection{}, fmt.Errorf("connection not found id=%s", connectionID)
	}

	return connection, nil
}

func (store *MemoryStore) GetConnections(options ConnectionSearchOptions) ([]Connection, error) {

	predicate := func(connection Connection) bool {

		if options.TunnelID == "" {
			return true
		}

		if connection.ID.TunnelID() == options.TunnelID {
			return true
		}

		return false
	}

	connections := store.connectionTable.list(predicate)
	return connections, nil
}

func (store *MemoryStore) AddConnection(connection Connection) error {

	err := store.connectionTable.add(connection)
	return err
}

func (store *MemoryStore) UpdateConnection(connection Connection) error {

	err := store.connectionTable.update(connection)
	return err
}

func (store *MemoryStore) DeleteConnection(connectionID ConnectionID) error {

	store.connectionTable.delete(connectionID)
	return nil
}

type dataTableElement[TKey comparable] interface {
	GetKey() TKey
	GetRev() uint
	SetRev(rev uint)
}

type dataTable[TKey comparable, TElem dataTableElement[TKey]] struct {
	name    string
	rwMutex *sync.RWMutex
	dataMap map[TKey]TElem
}

func (table *dataTable[TKey, TElem]) getWithKey(key TKey) (TElem, bool) {

	table.rwMutex.RLock()
	defer table.rwMutex.RUnlock()

	elem, ok := table.dataMap[key]

	return elem, ok
}

func (table *dataTable[TKey, TElem]) list(predicate func(elem TElem) bool) []TElem {

	table.rwMutex.RLock()
	defer table.rwMutex.RUnlock()

	if predicate == nil {
		predicate = func(elem TElem) bool { return true }
	}

	list := make([]TElem, 0)

	for _, elem := range table.dataMap {
		if predicate(elem) {
			list = append(list, elem)
		}
	}

	return list
}

func (table *dataTable[TKey, TElem]) add(value TElem) error {

	table.rwMutex.Lock()
	defer table.rwMutex.Unlock()

	_, ok := table.dataMap[value.GetKey()]
	if ok {
		return fmt.Errorf("%s data has already existed. key=%v", table.name, value.GetKey())
	}

	table.dataMap[value.GetKey()] = value

	return nil
}

func (table *dataTable[TKey, TElem]) update(value TElem) error {

	table.rwMutex.Lock()
	defer table.rwMutex.Unlock()

	target, ok := table.dataMap[value.GetKey()]
	if !ok {
		return fmt.Errorf("%s data not found. key=%v", table.name, value.GetKey())
	}

	if target.GetRev() != value.GetRev() {
		return fmt.Errorf("%s data update conflict. key=%v", table.name, value.GetKey())
	}

	value.SetRev(value.GetRev() + 1)
	table.dataMap[value.GetKey()] = value

	return nil
}

func (table *dataTable[TKey, TElem]) delete(key TKey) {

	table.rwMutex.Lock()
	defer table.rwMutex.Unlock()

	delete(table.dataMap, key)
}
