package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	//_ "github.com/lib/pq"
	//_ "github.com/jackc/pgx"
	"github.com/jmoiron/sqlx"
	//"database/sql"
	"github.com/jackc/pgerrcode"

	"github.com/OlegMzhelskiy/gophermart/internal/models"
)

var DatabaseTestURL string

type Store struct {
	//config *Config
	databaseURL string
	db          *sqlx.DB
}

func NewSQLStore(databaseURL string) (Repository, error) {
	s, err := newStore(databaseURL)
	return s, err
}

func newStore(databaseURL string) (*Store, error) {
	store := &Store{databaseURL: databaseURL}
	if err := store.Open(); err != nil {
		var pqError *pq.Error
		if errors.As(err, &pqError) {
			pqerr, _ := err.(*pq.Error)
			if pqerr.Code != pgerrcode.InvalidCatalogName { //"3D000"
				return nil, err
			}
			//create db
			if err := createDataBase(store.databaseURL); err != nil {
				return nil, fmt.Errorf("create db error: %w", err)
			}
			//reconnection
			if err := store.Open(); err != nil {
				return nil, fmt.Errorf("open db error: %w", err)
			}
		}
	}
	//create tables
	_, err := store.db.Exec(`CREATE TABLE IF NOT EXISTS users(
		id SERIAL PRIMARY KEY,	
    	login TEXT UNIQUE NOT NULL,
		encrypted_password TEXT NOT NULL);
	CREATE TABLE IF NOT EXISTS orders(
	    number TEXT UNIQUE NOT NULL,
	    status VARCHAR(25),
	    sum NUMERIC DEFAULT 0,
	    user_id INTEGER NOT NULL,
	    uploaded_at TIMESTAMPTZ,
	    updated_at TIMESTAMPTZ DEFAULT '0001-01-01 00:00:00 +0000');
	CREATE TABLE IF NOT EXISTS withdrawals(
	    order_number TEXT PRIMARY KEY NOT NULL,
	    sum NUMERIC DEFAULT 0,
	    user_id INTEGER NOT NULL,
	    processed_at TIMESTAMPTZ);`)

	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("new store exec query error: %w", err)
	}
	return store, nil
}

func (s *Store) Open() error {
	db, err := sqlx.Open("postgres", s.databaseURL)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("open db error: %w", err)
	}
	if err := db.Ping(); err != nil {
		return err
	}
	s.db = db
	return nil
}

func createDataBase(databaseURL string) error {
	db, err := sqlx.Open("postgres", getPostgresConn(databaseURL))
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("open db error: %w", err)
	}

	//parse bd name
	dbname := ""
	ss := strings.Split(databaseURL, " ")
	for _, str := range ss {
		if strings.HasPrefix(str, "dbname=") {
			dbname = strings.Replace(str, "dbname=", "", 1)
			break
		}
	}

	if dbname == "" {
		return errors.New("db name is empty")
	}
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbname))
	if err != nil {
		fmt.Println(err)
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	db.Close()
	return nil
}

//TODO: ???????????????????? ?????? ?????????? ???????????? ?????????????????????? ?????????????????? ?? ???? ?????? ?????????????????? ?????? ????????
func getPostgresConn(databaseURL string) string {
	return "host=localhost user=postgres password=123 sslmode=disable"
}

func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Store) CreateUser(login, encryptedPas string) (string, error) {
	var userID int
	err := s.db.QueryRowx("INSERT INTO users (login, encrypted_password) VALUES ($1, $2) RETURNING id",
		login, encryptedPas).Scan(&userID)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(userID), nil
}

func (s *Store) UserExist(login string) (bool, error) {
	var id int
	err := s.db.QueryRowx("SELECT id FROM users WHERE login=$1", login).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Store) GetUserHashPassword(login string) (string, error) {
	var encryptedPas string
	err := s.db.QueryRowx("SELECT encrypted_password FROM users WHERE login=$1", login).Scan(&encryptedPas)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrUserNotFound
		}
		return "", err
	}
	return encryptedPas, nil
}

func (s *Store) GetUserByLogin(login string) (models.User, error) {
	user := models.User{}
	//err := s.db.QueryRowx("SELECT * FROM users WHERE login=$1", login).Scan(&user)
	err := s.db.Get(&user, "SELECT * FROM users WHERE login=$1", login)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, ErrUserNotFound
		}
		return user, err
	}
	return user, nil
}

func (s *Store) GetOrderByNumber(number models.OrderNumber) (models.Order, error) {
	order := models.Order{}
	err := s.db.Get(&order, "SELECT * FROM orders WHERE number=$1", number)
	if err != nil {
		if err == sql.ErrNoRows {
			return order, ErrOrderNotFound
		}
		return order, err
	}
	return order, nil
}

func (s *Store) CreateOrder(order models.Order) error {
	_, err := s.db.Exec("INSERT INTO orders (number, user_id, uploaded_at, status) VALUES ($1, $2, $3, $4)",
		order.Number, order.UserID, order.UploadedAt, models.OrderStatusNew)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) GetOrderListByUserID(userID string) ([]models.Order, error) {
	orderList := []models.Order{}
	err := s.db.Select(&orderList, "SELECT * FROM orders WHERE user_id=$1 ORDER BY uploaded_at ASC", userID)
	if err != nil && err != sql.ErrNoRows {
		return orderList, err
	} else {
		return orderList, nil
	}
}

func (s *Store) GetBalanceByUserID(userID string) (models.SumScore, error) {
	var bal models.SumScore = 0
	//err := s.db.Get(&bal, "SELECT coalesce(SUM(sum), 0) FROM orders WHERE user_id=$1", userID)
	err := s.db.Get(&bal, `SELECT coalesce(SUM(sum), 0)
    FROM (
		SELECT sum FROM orders WHERE user_id=$1
		UNION ALL 
		SELECT -sum FROM withdrawals WHERE user_id=$1
		) AS q`, userID)
	if err != nil && err != sql.ErrNoRows {
		return -1, err
	}
	return bal, nil
}

func (s *Store) GetWithdrawalsByUserID(userID string) (models.SumScore, error) {
	var bal models.SumScore = 0
	err := s.db.Get(&bal, "SELECT coalesce(SUM(sum), 0) FROM withdrawals WHERE user_id=$1", userID)
	if err != nil {
		return -1, err
	}
	return bal, nil
}

func (s *Store) GetWithdrawalsListByUserID(userID string) ([]models.OrderWithdraw, error) {
	withdrawList := []models.OrderWithdraw{}
	err := s.db.Select(&withdrawList, `SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id=$1 
                                                 ORDER BY processed_at ASC`, userID)
	if err != nil && err != sql.ErrNoRows {
		return withdrawList, err
	} else {
		return withdrawList, nil
	}
}

func (s *Store) CreateWithdraw(userID string, withdraw models.WithdrawRequest) error {
	_, err := s.db.Exec("INSERT INTO withdrawals (user_id, order_number, sum, processed_at) VALUES ($1, $2, $3, $4)",
		userID, withdraw.OrderNumber, withdraw.Sum, time.Now())
	if err != nil {
		if errPq, ok := err.(*pq.Error); ok && errPq.Code == "23505" {
			return ErrWithdrawAlreadyExist //err.Code.Name()
		}
		return err
	}
	return nil
}

func (s *Store) GetOrdersWithStatus(status ...models.OrderStatus) ([]models.OrderNumber, error) {
	orderNumbers := []models.OrderNumber{}
	if len(status) == 0 {
		return orderNumbers, errors.New("there is no status")
	}
	// join condition
	var str string
	strStat := make([]string, 0, len(status))
	for _, v := range status {
		strStat = append(strStat, fmt.Sprintf("status='%s'", v))
	}
	if len(status) >= 1 {
		str = strings.Join(strStat, " OR ")
	} else {
		str = strStat[0]
	}
	err := s.db.Select(&orderNumbers,
		fmt.Sprintf("SELECT number FROM orders WHERE %s ORDER BY uploaded_at ASC", str))
	if err != nil && err != sql.ErrNoRows {
		return orderNumbers, err
	}
	return orderNumbers, nil
}

func (s *Store) UpdateOrder(ord models.Order) error {
	_, err := s.db.Exec("UPDATE orders SET status=$1, sum=$2, updated_at=$3 WHERE number=$4",
		ord.Status, ord.Accrual, time.Now(), ord.Number)
	if err != nil {
		return err
	}
	return nil
}
