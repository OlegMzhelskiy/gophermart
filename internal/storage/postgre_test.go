package storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	DatabaseTestURL = os.Getenv("databaseTestURL")
	if DatabaseTestURL == "" {
		DatabaseTestURL = "host=localhost dbname=gophermart_test user=postgres password=123 sslmode=disable"
	}
	os.Exit(m.Run())
}

//truncate tables test db
func teardown(t *testing.T, s *Store, tables ...string) {
	if _, err := s.db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE", strings.Join(tables, ", "))); err != nil {
		t.Fatal(err)
	}
}

func TestStore_CreateUser(t *testing.T) {
	type args struct {
		login        string
		encryptedPas string
	}
	tests := []struct {
		name string
		//fields  fields
		args args
		//want    string
		wantErr bool
	}{
		{
			name: "valid",
			args: args{login: "user1",
				encryptedPas: "qwerty123"},
			wantErr: false,
			//wantErr: nil,
		},
		{
			name: "invalid: already exists",
			args: args{login: "user1",
				encryptedPas: "qwerty999"},
			wantErr: true,
			//wantErr: nil,
		},
	}

	s, teardown := TestStore(t)
	defer teardown("users")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := s.CreateUser(tt.args.login, tt.args.encryptedPas)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Equal(t, "-1", userID)
			} else {
				if err != nil || userID == "-1" {
					t.Errorf("CreateUser() err = %s and userID = %s", err, userID)
				}
			}
		})
	}
}
