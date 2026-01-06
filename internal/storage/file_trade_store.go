package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/PxPatel/trading-system/internal/types"
)

// FileTradeStore implements TradeStore using append-only file writes.
// Writes are asynchronous for performance. Read operations return empty
// (file is write-only, use CompositeTradeStore with InMemoryTradeStore for reads).
type FileTradeStore struct {
	file    *os.File
	encoder *json.Encoder
	mutex   sync.Mutex
}

// NewFileTradeStore creates a new file-based trade store
func NewFileTradeStore(filePath string) (*FileTradeStore, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open trade log: %w", err)
	}

	return &FileTradeStore{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

func (s *FileTradeStore) Save(trade *types.Trade) error {
	// Async write to avoid blocking matching engine
	go func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		_ = s.encoder.Encode(trade)
	}()
	return nil
}

func (s *FileTradeStore) SaveBatch(trades []*types.Trade) error {
	// Async batch write
	go func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		for _, trade := range trades {
			_ = s.encoder.Encode(trade)
		}
	}()
	return nil
}

func (s *FileTradeStore) GetRecent(limit int) ([]*types.Trade, error) {
	// File store is write-only, no read support
	// Use CompositeTradeStore with InMemoryTradeStore for reads
	return []*types.Trade{}, nil
}

func (s *FileTradeStore) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
