package types

import "fmt"

var (
	ErrInvalidParams    = fmt.Errorf("alignment: invalid params")
	ErrModuleInactive   = fmt.Errorf("alignment: module inactive")
	ErrInvalidThreshold = fmt.Errorf("alignment: invalid threshold value")
	ErrInvalidWeight    = fmt.Errorf("alignment: sensor weights must sum to BasisPoints")
)
