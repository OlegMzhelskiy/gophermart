package main

import (
	"fmt"
	"github.com/OlegMzhelskiy/gophermart/internal/apiserver"
	"github.com/OlegMzhelskiy/gophermart/internal/storage"
	"log"
)

func main() {

	//baseURL := "http://" + addr + "/"

	//dbURI := getFlValue("DATABASE_URI", "d", "", "database uri")
	//asAddr := getFlValue("ACCRUAL_SYSTEM_ADDRESS", "r", "host=localhost dbname=gophermart user=postgres password=123 sslmode=disable", "accrual system address")

	//store, err := storage.NewSQLStore(dbDSN)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	////init server and start
	//cfg := &apiserver.Config{
	//	Addr:  addr,
	//	Store: store,
	//}
	//srv := apiserver.NewServer(cfg)
	//log.Fatal(srv.Run())

	log.Fatal(run())
}

func run() error {
	cfg := apiserver.NewConfig()
	store, err := storage.NewSQLStore(cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("db connection error: %w", err)
	}
	cfg.Store = store
	//init and start server
	//cfg := &apiserver.Config{
	//	Addr:  addr,
	//	Store: store,
	//}
	srv := apiserver.NewServer(cfg)
	if err := srv.Run(); err != nil {
		return fmt.Errorf("http-server failed: %w", err)
	}
	srv.Stop()
	return nil
}

//func getFlValue(envName, flName, defValue, usage string) string {
//	val, _ := os.LookupEnv(envName)
//	flVal := flag.String(flName, val, usage) //SERVER_ADDRESS
//	val = *flVal
//	if val == "" {
//		val = defValue
//	}
//	return val
//}

//func getVarValue(flagValue, envVarName, defValue string) string {
//	var ok bool
//	varVal := flagValue
//	if len(flagValue) == 0 {
//		varVal, ok = os.LookupEnv(envVarName) //URLdata.json
//		if !ok || varVal == "" {
//			varVal = defValue
//		}
//	}
//	return varVal
//}
