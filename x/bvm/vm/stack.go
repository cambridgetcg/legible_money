package vm

import (
	"errors"
	"math/big"
)

var (
	ErrStackOverflow  = errors.New("stack overflow")
	ErrStackUnderflow = errors.New("stack underflow")
)

const DefaultMaxStackDepth = 1024

type Stack struct {
	items    []*big.Int
	maxDepth int
}

func NewStack() *Stack {
	return &Stack{
		items:    make([]*big.Int, 0, 64),
		maxDepth: DefaultMaxStackDepth,
	}
}

func (s *Stack) Push(v *big.Int) error {
	if len(s.items) >= s.maxDepth {
		return ErrStackOverflow
	}
	s.items = append(s.items, new(big.Int).Set(v))
	return nil
}

func (s *Stack) Pop() (*big.Int, error) {
	if len(s.items) == 0 {
		return nil, ErrStackUnderflow
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, nil
}

// Peek returns the item at depth from top of stack (0 = top).
func (s *Stack) Peek(depth int) (*big.Int, error) {
	idx := len(s.items) - 1 - depth
	if idx < 0 || depth < 0 {
		return nil, ErrStackUnderflow
	}
	return s.items[idx], nil
}

// Swap swaps top-of-stack with the element at depth (1 = second from top).
func (s *Stack) Swap(depth int) error {
	top := len(s.items) - 1
	idx := top - depth
	if idx < 0 || depth < 1 {
		return ErrStackUnderflow
	}
	s.items[top], s.items[idx] = s.items[idx], s.items[top]
	return nil
}

// Dup duplicates the element at depth (1 = top) and pushes it.
func (s *Stack) Dup(depth int) error {
	idx := len(s.items) - depth
	if idx < 0 || depth < 1 {
		return ErrStackUnderflow
	}
	if len(s.items) >= s.maxDepth {
		return ErrStackOverflow
	}
	s.items = append(s.items, new(big.Int).Set(s.items[idx]))
	return nil
}

func (s *Stack) Depth() int {
	return len(s.items)
}
