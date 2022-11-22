package apiserver

import (
	"bytes"
	"encoding/json"
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var (
	baseURL         = "http://" + DefaultHost
	DatabaseTestURL = ""
)

type args struct {
	method  string
	body    interface{}
	url     string
	headers map[string]string
}
type response struct {
	code    int
	body    string
	headers map[string]string
}
type testCase struct {
	name     string
	args     args
	expected response
}

func TestMain(m *testing.M) {
	DatabaseTestURL = os.Getenv("databaseTestURL")
	if DatabaseTestURL == "" {
		DatabaseTestURL = "host=localhost dbname=gophermart_test user=postgres password=123 sslmode=disable"
	}
	os.Exit(m.Run())
}

func NewTestServer() *APIServer {

	//s, teardown := storage.TestStore(t)
	//defer teardown

	store, err := storage.NewSQLStore(DatabaseTestURL)
	if err != nil {
		log.Fatal(err)
	}
	cfg := Config{
		Addr:  DefaultHost,
		Store: store,
	}
	srv := NewServer(cfg)
	//srv.ConfigurateServer()
	return srv
}

func (s *APIServer) StopTestServer() {
	defer s.useCase.CloseRepo()
}

func TestAPIServer_registerUser(t *testing.T) {
	srv := NewTestServer()
	defer srv.StopTestServer()

	url := baseURL + "/api/user/register"

	tests := []testCase{
		{name: "valid",
			args: args{
				http.MethodPost,
				map[string]string{
					"login":    "user1",
					"password": "qwerty123",
				},
				url,
				map[string]string{},
			},
			expected: response{code: 200},
		},
		{name: "not valid",
			args: args{
				http.MethodPost,
				map[string]string{
					"login":    "user1",
					"password": "123",
				},
				url,
				map[string]string{},
			},
			expected: response{code: 400},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			b := &bytes.Buffer{}
			json.NewEncoder(b).Encode(tt.args.body)
			request, _ := http.NewRequest(tt.args.method, tt.args.url, b)

			srv.ServeHTTP(rec, request)
			assert.Equal(t, tt.expected.code, rec.Code)
		})
	}
}
