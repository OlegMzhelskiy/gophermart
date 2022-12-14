package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	//"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/OlegMzhelskiy/gophermart/docs"
	"github.com/OlegMzhelskiy/gophermart/internal/models"
	"github.com/OlegMzhelskiy/gophermart/internal/usecase"
	"github.com/OlegMzhelskiy/gophermart/pkg/logging"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

type ctxKey string

var (
	DefaultHost         = "localhost:8088"
	DefaultDBDSN        = "host=localhost dbname=gophermart user=postgres password=123 sslmode=disable"
	ctxKeyUserID ctxKey = "userID"
)

type APIServer struct {
	addr    string
	router  *chi.Mux
	useCase usecase.UseCases
	done    chan struct{}
	logger  logging.Loggerer
	prod    bool
}

func NewServer(cfg Config) *APIServer {
	done := make(chan struct{})
	uc := usecase.NewUseCases(cfg.Store, done, cfg.AcSysAddr)
	srv := &APIServer{
		addr:    cfg.Addr,
		useCase: *uc,
		done:    done,
		logger:  cfg.Logger,
		prod:    cfg.Prod,
	}
	srv.configureRouter()
	return srv
}

func (s *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Run starting api server
func (s *APIServer) Run() error {
	s.logger.Info("Start server on ", s.addr)
	return http.ListenAndServe(s.addr, s.router)
}

func (s *APIServer) Stop() {
	close(s.done)
	s.useCase.CloseRepo()
	s.logger.Info("server stopped")
	time.Sleep(2 * time.Second)
}

func (s *APIServer) ConfigurateServer() error {
	s.configureRouter()
	return nil
}

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

	if s.prod {
		s.router.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("http://"+s.addr+"/swagger/doc.json")))
	}

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
			s.errorLog(w, r, http.StatusInternalServerError, fmt.Errorf("marshal error: %w", err))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		io.WriteString(w, string(jsonData))
	} else {
		w.WriteHeader(code)
	}
}

func (s *APIServer) respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	if data != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		//w.Header().Set("Access-Control-Allow-Origin", "*")
		//w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(data)
	} else {
		w.WriteHeader(code)
	}
}

func (s *APIServer) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	s.respond(w, r, code, map[string]string{"error": err.Error()})
}

func (s *APIServer) errorLog(w http.ResponseWriter, r *http.Request, code int, err error) {
	s.error(w, r, code, err)
	reqID, _ := r.Context().Value(middleware.RequestIDKey).(string)
	s.logger.LogWithFields(logging.ErrorLevel, "handler error:", logging.Fields{
		"requestID":   reqID,
		"requestURI":  r.RequestURI,
		"method":      r.Method,
		"description": err.Error(),
	})
}

func (s *APIServer) handlerF(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "ok")
}

type requestAuth struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// RegisterUser
// @Summary      RegisterUser
// @Description  Register new user
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        user body models.User true "login and password"
// @Success      200  {object}  models.User
// @Failure      400  {string}  string
// @Failure      409  {object}  string
// @Failure      500  {object}  string
// @Header       200  {string}  Authorization     "token"
// @Router       /api/user/register [post]
func (s *APIServer) RegisterUser(w http.ResponseWriter, r *http.Request) {
	request := &requestAuth{}
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		s.errorLog(w, r, http.StatusBadRequest, err)
		return
	}
	user := models.User{Login: request.Login, Password: request.Password}
	err := s.useCase.User.CreateUser(&user)
	if err != nil {
		if errors.Is(err, usecase.ErrLoginAlreadyExists) {
			s.error(w, r, http.StatusConflict, usecase.ErrLoginAlreadyExists)
		} else if errors.Is(err, usecase.ErrLoginIsEmpty) {
			s.error(w, r, http.StatusBadRequest, usecase.ErrLoginIsEmpty)
		} else if errors.Is(err, usecase.ErrPasswordTooShort) {
			s.error(w, r, http.StatusBadRequest, usecase.ErrPasswordTooShort)
		} else {
			s.errorLog(w, r, http.StatusInternalServerError, err)
		}
		return
	}
	s.respondGeneratedToken(w, r, user)
}

// AuthUser
// @Summary      AuthUser
// @Description  Auth user
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        user body models.User true "login and password"
// @Success      200  {object}  string
// @Failure      400  {object}  string
// @Failure      401  {object}  string
// @Failure      500  {object}  string
// @Header       200  {string}  Authorization     "token"
// @Router       /api/user/login [post]
func (s *APIServer) AuthUser(w http.ResponseWriter, r *http.Request) {
	request := &requestAuth{}
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		s.errorLog(w, r, http.StatusBadRequest, err)
		return
	}
	user := models.User{Login: request.Login, Password: request.Password}
	err := s.useCase.User.AuthUser(&user)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidLoginOrPassword) {
			s.error(w, r, http.StatusUnauthorized, usecase.ErrInvalidLoginOrPassword)
		} else {
			s.errorLog(w, r, http.StatusInternalServerError, err)
		}
		return
	}

	s.respondGeneratedToken(w, r, user)
}

//Generate token and set respond
func (s *APIServer) respondGeneratedToken(w http.ResponseWriter, r *http.Request, user models.User) {
	token, err := s.useCase.User.GenerateToken(user)
	if err != nil {
		s.errorLog(w, r, http.StatusInternalServerError, err)
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

// UploadOrder
// @Summary      UploadOrder
// @Security ApiKeyAuth
// @Description  Upload order
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        order_number body string true "uploading order number"
// @Success      200  {object}  string
// @Success      202  {object}  string
// @Failure      400  {object}  string
// @Failure      401  {object}  string
// @Failure      409  {object}  string
// @Failure      422  {object}  string
// @Failure      500  {object}  string
// @Router       /api/user/orders [post]
func (s *APIServer) UploadOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		s.errorLog(w, r, http.StatusBadRequest, err)
		return
	}
	if len(body) == 0 {
		s.error(w, r, http.StatusBadRequest, errors.New("request doesn't have order number"))
		return
	}
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.errorLog(w, r, http.StatusInternalServerError, errors.New("invalid type user ID"))
		return
	}
	order := models.Order{
		UserID:     userID,
		Number:     models.OrderNumber(body),
		UploadedAt: time.Now(),
	}
	if err := s.useCase.Order.UploadOrder(order); err != nil {
		if errors.Is(err, usecase.ErrOrderAlreadyUploadAnotherUser) {
			s.error(w, r, http.StatusConflict, usecase.ErrOrderAlreadyUploadAnotherUser)
		} else if errors.Is(err, usecase.ErrOrderAlreadyUploadThisUser) {
			s.error(w, r, http.StatusOK, usecase.ErrOrderAlreadyUploadThisUser)
		} else if errors.Is(err, usecase.ErrInvalidOrderNumber) {
			s.error(w, r, http.StatusUnprocessableEntity, usecase.ErrInvalidOrderNumber)
		} else {
			s.errorLog(w, r, http.StatusInternalServerError, err)
		}
		return
	}
	s.respond(w, r, http.StatusAccepted, nil)
}

// GetOrderList
// @Summary      GetOrderList
// @Security ApiKeyAuth
// @Description  Return order list
// @Tags         orders
// @Accept       json
// @Produce      json
// @Success      200  {array}  models.Order
// @Success      204  {array}	string{}
// @Failure      401  {object}  int
// @Failure      500  {object}  int
// @Router       /api/user/orders [get]
func (s *APIServer) GetOrderList(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.errorLog(w, r, http.StatusInternalServerError, errors.New("invalid type user ID"))
		return
	}
	list, err := s.useCase.Order.GetOrderList(userID)
	if err != nil {
		s.errorLog(w, r, http.StatusInternalServerError, fmt.Errorf("get list order failed: %w", err))
	} else if len(list) == 0 {
		s.respond(w, r, http.StatusNoContent, list)
	} else {
		s.respondJSON(w, r, http.StatusOK, list)
	}
}

// GetBalance
// @Summary      GetBalance
// @Security ApiKeyAuth
// @Description  Return user's balance
// @Tags         balance
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.UserBalance
// @Failure      401  {object}  string
// @Failure      500  {object}  string
// @Router       /api/user/balance [get]
func (s *APIServer) GetBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(ctxKeyUserID).(string)
	if !ok {
		s.errorLog(w, r, http.StatusBadRequest, errors.New("invalid type user ID"))
		return
	}
	userBal, err := s.useCase.User.GetUserBalanceAndWithdrawals(userID)
	if err != nil {
		s.errorLog(w, r, http.StatusInternalServerError, errors.New("internal server error"))
	} else {
		s.respondJSON(w, r, http.StatusOK, userBal)
	}
}

// GetWithdrawals
// @Summary      GetWithdrawals
// @Security ApiKeyAuth
// @Description  Getting information about withdrawal of funds
// @Tags         orders
// @Accept       json
// @Produce      json
// @Success      200  {object}  []models.OrderWithdraw
// @Success      204  {object} 	string
// @Failure      401  {object}  string
// @Failure      500  {object}  string
// @Router       /api/user/withdrawals [get]
func (s *APIServer) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.errorLog(w, r, http.StatusInternalServerError, errors.New("invalid type user ID"))
		return
	}
	userWith, err := s.useCase.Order.GetWithdrawals(userID)
	if err != nil {
		s.errorLog(w, r, http.StatusInternalServerError, errors.New("internal server error"))
	} else {
		if len(userWith) == 0 {
			s.respond(w, r, http.StatusNoContent, "there are no withdrawals")
		} else {
			s.respondJSON(w, r, http.StatusOK, userWith)
		}
	}
}

// Withdraw
// @Summary      Withdraw
// @Security ApiKeyAuth
// @Description  Request to debit funds
// @Tags         balance
// @Accept       json
// @Produce      json
// @Param    	param body models.WithdrawRequest true "order number and sum"
// @Success      200  {object}  string
// @Failure      401  {object}  string
// @Failure      402  {object}  string
// @Failure      422  {object}  string
// @Failure      500  {object}  string
// @Router       /api/user/balance/withdraw [post]
func (s *APIServer) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxKeyUserID).(string)
	if !ok {
		s.errorLog(w, r, http.StatusInternalServerError, errors.New("invalid type user ID"))
		return
	}
	wReq := models.WithdrawRequest{}
	if err := json.NewDecoder(r.Body).Decode(&wReq); err != nil {
		s.errorLog(w, r, http.StatusBadRequest, err) //errors.New("bad request"))
		return
	}
	err := s.useCase.Order.Withdraw(userID, wReq)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidOrderNumber) {
			s.error(w, r, http.StatusUnprocessableEntity, usecase.ErrInvalidOrderNumber)
		} else if errors.Is(err, usecase.ErrNotEnoughFunds) {
			s.error(w, r, http.StatusPaymentRequired, usecase.ErrNotEnoughFunds)
		} else if errors.Is(err, usecase.ErrWithdrawAlreadyExist) {
			s.error(w, r, http.StatusBadRequest, usecase.ErrWithdrawAlreadyExist)
		} else {
			s.errorLog(w, r, http.StatusInternalServerError, errors.New("internal server error"))
		}
		return
	}
	s.respond(w, r, http.StatusOK, nil)
}
