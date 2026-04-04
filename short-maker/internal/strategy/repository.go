package strategy

import (
	"encoding/json"
	"fmt"
	"os"
)

type Repository struct {
	strategies []*Strategy
	index      map[string]*Strategy
}

func LoadFromJSON(data []byte) (*Repository, error) {
	var strategies []*Strategy
	if err := json.Unmarshal(data, &strategies); err != nil {
		return nil, fmt.Errorf("parse strategies JSON: %w", err)
	}
	repo := &Repository{
		strategies: strategies,
		index:      make(map[string]*Strategy, len(strategies)),
	}
	for _, s := range strategies {
		repo.index[s.ID] = s
	}
	return repo, nil
}

func LoadFromFile(path string) (*Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read strategies file: %w", err)
	}
	return LoadFromJSON(data)
}

func (r *Repository) All() []*Strategy {
	return r.strategies
}

func (r *Repository) Get(id string) *Strategy {
	return r.index[id]
}
