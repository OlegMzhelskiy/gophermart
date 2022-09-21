package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/OlegMzhelskiy/gophermart/internal/models"
	"github.com/OlegMzhelskiy/gophermart/internal/usecase"
)

type ctxKey string

var (
	DefaultHost         = "localhost:8088"
	DefaultDBDSN        = "host=localhost dbname=gophermart user=postgres password=123 sslmode=disable"
	ctxKeyUserID ctxKey = "userID"
)

type APIServer struct {
	//config *Config
	addr    string
	router  *chi.Mux
	useCase usecase.UseCases
	//store   storage.Repository
	done chan struct{}
}

func NewServer(cfg Config) *APIServer {
	done := make(chan struct{})
	uc := usecase.NewUseCases(cfg.Store, done, cfg.AcSysAddr)
	srv := &APIServer{
		addr:    cfg.Addr,
		useCase: *uc,
		done:    done,
	}
	srv.configureRouter()

	//uc.Order.RunWorkerGettingOrderStatus()

	return srv
}

func (s *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Run starting api server
func (s *APIServer) Run() error {
	//if err := s.ConfigurateServer(); err != nil {
	//	return err
	//}
	//defer s.store.Close()

	return http.ListenAndServe(s.addr, s.router)
}

func (s *APIServer) Stop() {
	close(s.done)
	s.useCase.CloseRepo()
	time.Sleep(2 * time.Second)
}

func (s *APIServer) ConfigurateServer() error {
	s.configureRouter()
	//if err := s.configureStore(); err != nil {
	//	return err
	//}
	return nil
}

//func (s *APIServer) configureStore() error {
//	//store := storage.NewStore(s.config.storeCfg)
//	if err := s.store.Open(); err != nil {
//		return err
//	}
//	return nil
//}

func (s *APIServer) configureRouter() {
	s.router = chi.NewRouter()

	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	s.router.Get("/", s.handlerF) //test
	s.router.Post("/api/user/register", s.RegisterUser)
	s.router.Post("/api/user/login", s.AuthUser)

	//routes for "api" resource
	s.router.Route("/api/user", func(r chi.Router) {
		//r.With(s.authenticateUser).Get("/id", s.getUserID)
		r.Use(s.authenticateUser)
		r.Get("/id", s.getUserID)
		r.Route("/orders", func(ord chi.Router) {
			ord.Post("/", s.UploadOrder)
			ord.Get("/", s.GetOrderList)
		})
		r.Get("/withdrawals", s.GetWithdrawals)
		r.Route("/balance", func(bal chi.Router) {
			bal.Get("/", s.GetBalance)
			bal.Post("/withdraw", s.Withdraw)
		})
	})
}

func (s *APIServer) respondJSON(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	if data != nil {
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, fmt.Errorf("marshal error: %w", err))
		}
		w.WriteHeader(code)
		io.WriteString(w, string(jsonData))
	} else {
		w.WriteHeader(code)
	}
}

func (s *APIServer) respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	if data != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(data)
	} else {
		w.WriteHeader(code)
	}
}

func (s *APIServer) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	s.respond(w, r, code, map[string]string{"error": err.Error()})
	log.Printf("handler error: %s", err.Error())
}

func (s *APIServer) handlerF(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "ok")
}

type requestAuth struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *APIServer) RegisterUser(w http.ResponseWriter, r *http.Request) {
	request := &requestAuth{}
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		s.error(w, r, http.StatusBadRequest, err)
		return
	}
	user := models.User{Login: request.Login, Password: request.Password}
	err := s.useCase.User.CreateUser(&user)
	if err != nil {
		if errors.Is(err, usecase.ErrLoginAlreadyExists) {
			s.error(w, r, http.StatusConflict, err)
		} else if errors.Is(err, usecase.ErrLoginIsEmpty) || errors.Is(err, usecase.ErrPasswordTooShort) {
			s.error(w, r, http.StatusBadRequest, err)
		} else {
			s.error(w, r, http.StatusInternalServerError, err)
		}
		return
	}
	s.respondGeneratedToken(w, r, user)
}

func (s *APIServer) AuthUser(w http.ResponseWriter, r *http.Request) {
	request := &requestAuth{}
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		s.error(w, r, http.StatusBadRequest, err)
		return
	}
	user := models.User{Login: request.Login, Password: request.Password}
	err := s.useCase.User.AuthUser(&user)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidLoginOrPassword) {
			s.error(w, r, http.StatusUnauthorized, err)
		} else {
			s.error(w, r, http.StatusInternalServerError, err)
		}
		return
	}

	//token, err := s.useCase.User.GenerateToken(user)
	//if err != nil {
	//	s.error(w, r, http.StatusInternalServerError, err)
	//	return
	//}
	//respToken := map[string]string{"token": token}
	//s.respond(w, r, http.StatusOK, respToken)
	s.respondGeneratedToken(w, r, user)
}

//Generate token and set respond
func (s *APIServer) respondGeneratedToken(w http.ResponseWriter, r *http.Request, user models.User) {
	token, err := s.useCase.User.GenerateToken(user)
	if err != nil {
		s.error(w, r, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Authorization", token)
	respToken := map[string]string{"token": token}
	s.respond(w, r, http.StatusOK, respToken)
}

//middleware for auth user
func (s *APIServer) authenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			s.error(w, r, http.StatusUnauthorized, errors.New("auth header is empty"))
			return
		}
		//ms := strings.Split(auth, " ")
		//if len(ms) != 2 || (len(ms) == 2 && (ms[0] != "Bearer" || ms[1] == "")) {
		//	s.error(w, r, http.StatusUnauthorized, errors.New("invalid auth header"))
		//	return
		//}
		//token := ms[1]
		token := strings.Replace(auth, "Bearer ", "", 1)
		valid, claims, err := s.useCase.User.ParseToken(token)
		if err != nil {
			s.error(w, r, http.StatusUnauthorized, fmt.Errorf("parse token failed: %w", err))
			return
		}
		if !valid {
			s.error(w, r, http.StatusUnauthorized, errors.New("invalid token"))
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *APIServer) getUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxKeyUserID)
	s.respond(w, r, http.StatusOK, userID)
}

func (s *APIServer) UploadOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	//if err := json.NewDecoder(r.Body).Decode(&orderNumber); err != nil {
	if err != nil {
		s.error(w, r, http.StatusBadRequest, err)
		return
	}
	if len(body) == 0 {
		s.error(w, r, http.StatusBadRequest, errors.New("request doesn't have order number"))
		return
	}
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.error(w, r, http.StatusInternalServerError, errors.New("invalid type user ID"))
		return
	}
	order := models.Order{
		UserID:     userID,
		Number:     models.OrderNumber(body),
		UploadedAt: time.Now(),
	}
	if err := s.useCase.Order.UploadOrder(order); err != nil {
		if errors.Is(err, usecase.ErrOrderAlreadyUploadAnotherUser) {
			s.error(w, r, http.StatusConflict, err)
		} else if errors.Is(err, usecase.ErrOrderAlreadyUploadThisUser) {
			s.error(w, r, http.StatusOK, err)
		} else if errors.Is(err, usecase.ErrInvalidOrderNumber) {
			s.error(w, r, http.StatusUnprocessableEntity, err)
		} else {
			s.error(w, r, http.StatusInternalServerError, err)
		}
		return
	}
	s.respond(w, r, http.StatusAccepted, nil)
}

func (s *APIServer) GetOrderList(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.error(w, r, http.StatusInternalServerError, errors.New("invalid type user ID"))
		return
	}
	list, err := s.useCase.Order.GetOrderList(userID)
	if err != nil {
		s.error(w, r, http.StatusInternalServerError, fmt.Errorf("get list order failed: %w", err))
	} else if len(list) == 0 {
		s.respond(w, r, http.StatusNoContent, list)
	} else {
		s.respondJSON(w, r, http.StatusOK, list)
	}
}

func (s *APIServer) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.error(w, r, http.StatusBadRequest, errors.New("invalid type user ID"))
		return
	}
	userBal, err := s.useCase.User.GetUserBalanceAndWithdrawals(userID)
	if err != nil {
		s.error(w, r, http.StatusInternalServerError, errors.New("internal server error"))
	} else {
		s.respondJSON(w, r, http.StatusOK, userBal)
	}
}

func (s *APIServer) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.error(w, r, http.StatusInternalServerError, errors.New("invalid type user ID"))
		return
	}
	userWith, err := s.useCase.Order.GetWithdrawals(userID)
	if err != nil {
		s.error(w, r, http.StatusInternalServerError, errors.New("internal server error"))
	} else {
		if len(userWith) == 0 {
			s.respond(w, r, http.StatusNoContent, "there are no withdrawals")
		} else {
			s.respondJSON(w, r, http.StatusOK, userWith)
		}
	}
}

func (s *APIServer) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.error(w, r, http.StatusInternalServerError, errors.New("invalid type user ID"))
		return
	}
	wReq := models.WithdrawRequest{}
	if err := json.NewDecoder(r.Body).Decode(&wReq); err != nil {
		s.error(w, r, http.StatusBadRequest, errors.New("bad request"))
		return
	}
	err := s.useCase.Order.Withdraw(userID, wReq)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidOrderNumber) {
			s.error(w, r, http.StatusUnprocessableEntity, err)
		} else if errors.Is(err, usecase.ErrNotEnoughFunds) {
			s.error(w, r, http.StatusPaymentRequired, err)
		} else if errors.Is(err, usecase.ErrWithdrawAlreadyExist) {
			s.error(w, r, http.StatusBadRequest, err)
		} else {
			s.error(w, r, http.StatusInternalServerError, errors.New("internal server error"))
		}
		return
	}
	s.respond(w, r, http.StatusOK, nil)
}
