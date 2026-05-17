package store

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

var ErrTodoNotFound = errors.New("todo not found")

type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TodoStore struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]Todo
}

func NewTodoStore() *TodoStore {
	return &TodoStore{
		nextID: 1,
		items:  make(map[int64]Todo),
	}
}

func (s *TodoStore) List(ctx context.Context) ([]Todo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := make([]Todo, 0, len(s.items))
	for _, todo := range s.items {
		todos = append(todos, todo)
	}
	slices.SortFunc(todos, func(a, b Todo) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		default:
			return 0
		}
	})

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return todos, nil
}

func (s *TodoStore) Get(ctx context.Context, id int64) (Todo, error) {
	if err := ctx.Err(); err != nil {
		return Todo{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	todo, ok := s.items[id]
	if !ok {
		return Todo{}, ErrTodoNotFound
	}
	return todo, nil
}

func (s *TodoStore) Create(ctx context.Context, title string) (Todo, error) {
	if err := ctx.Err(); err != nil {
		return Todo{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	todo := Todo{
		ID:        s.nextID,
		Title:     title,
		Done:      false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.items[todo.ID] = todo
	s.nextID++

	return todo, nil
}

func (s *TodoStore) Update(ctx context.Context, id int64, title *string, done *bool) (Todo, error) {
	if err := ctx.Err(); err != nil {
		return Todo{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.items[id]
	if !ok {
		return Todo{}, ErrTodoNotFound
	}

	if title != nil {
		todo.Title = *title
	}
	if done != nil {
		todo.Done = *done
	}
	todo.UpdatedAt = time.Now().UTC()
	s.items[id] = todo

	return todo, nil
}

func (s *TodoStore) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return ErrTodoNotFound
	}
	delete(s.items, id)
	return nil
}
