package storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestStore(t *testing.T) (Repository, func(...string)) {
	s, err := newStore(DatabaseTestURL)
	assert.NoError(t, err)

	return s, func(tables ...string) {
		if len(tables) > 0 {
			if _, err := s.db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE", strings.Join(tables, ", "))); err != nil {
				t.Fatal(err)
			}
		}
		s.db.Close()
	}

}
