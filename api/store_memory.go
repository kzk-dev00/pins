package main

import (
	"context"
	"slices"
	"strings"
	"sync"
)

// MemoryStore はテスト用のメモリ上の保存先。DynamoStore と同じ動きをする。
type MemoryStore struct {
	mu         sync.Mutex
	categories map[string]Category // キー：ownerId + categoryId
	pins       map[string]Pin      // キー：ownerId + sk
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		categories: map[string]Category{},
		pins:       map[string]Pin{},
	}
}

func memKey(partition, sort string) string {
	return partition + "\x00" + sort
}

func (s *MemoryStore) ListCategories(_ context.Context, ownerID string) ([]Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Category
	for _, c := range s.categories {
		if c.OwnerID == ownerID {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b Category) int { return strings.Compare(a.CategoryID, b.CategoryID) })
	return out, nil
}

func (s *MemoryStore) GetCategory(_ context.Context, ownerID, categoryID string) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.categories[memKey(ownerID, categoryID)]
	if !ok {
		return Category{}, ErrNotFound
	}
	return c, nil
}

func (s *MemoryStore) CreateCategory(_ context.Context, c Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memKey(c.OwnerID, c.CategoryID)
	if _, ok := s.categories[key]; ok {
		return ErrConflict
	}
	s.categories[key] = c
	return nil
}

func (s *MemoryStore) UpdateCategory(_ context.Context, c Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memKey(c.OwnerID, c.CategoryID)
	if _, ok := s.categories[key]; !ok {
		return ErrNotFound
	}
	s.categories[key] = c
	return nil
}

func (s *MemoryStore) DeleteCategory(_ context.Context, ownerID, categoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memKey(ownerID, categoryID)
	if _, ok := s.categories[key]; !ok {
		return ErrNotFound
	}
	delete(s.categories, key)
	return nil
}

func (s *MemoryStore) FindCategoryByViewKey(_ context.Context, viewKey string) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.categories {
		if c.ViewKey == viewKey {
			return c, nil
		}
	}
	return Category{}, ErrNotFound
}

func (s *MemoryStore) ListPins(_ context.Context, ownerID, categoryID string) ([]Pin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := ""
	if categoryID != "" {
		prefix = categoryID + "#"
	}
	var out []Pin
	for _, p := range s.pins {
		if p.OwnerID == ownerID && strings.HasPrefix(p.SK, prefix) {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b Pin) int { return strings.Compare(a.SK, b.SK) })
	return out, nil
}

func (s *MemoryStore) GetPin(_ context.Context, ownerID, categoryID, pinID string) (Pin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pins[memKey(ownerID, pinSK(categoryID, pinID))]
	if !ok {
		return Pin{}, ErrNotFound
	}
	return p, nil
}

func (s *MemoryStore) CreatePin(_ context.Context, p Pin) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memKey(p.OwnerID, p.SK)
	if _, ok := s.pins[key]; ok {
		return ErrConflict
	}
	s.pins[key] = p
	return nil
}

func (s *MemoryStore) UpdatePin(_ context.Context, p Pin) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memKey(p.OwnerID, p.SK)
	if _, ok := s.pins[key]; !ok {
		return ErrNotFound
	}
	s.pins[key] = p
	return nil
}

func (s *MemoryStore) DeletePin(_ context.Context, ownerID, categoryID, pinID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memKey(ownerID, pinSK(categoryID, pinID))
	if _, ok := s.pins[key]; !ok {
		return ErrNotFound
	}
	delete(s.pins, key)
	return nil
}
