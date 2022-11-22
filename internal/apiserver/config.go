package apiserver

import (
	"flag"
	"github.com/OlegMzhelskiy/gophermart/pkg/logging"
	"os"
	"strings"

	"github.com/OlegMzhelskiy/gophermart/internal/storage"
)

type Config struct {
	Addr      string
	DBDSN     string
	AcSysAddr string
	Store     storage.Repository
	Logger    logging.Loggerer
	Prod      bool
}

func NewConfig() Config {
	flagHost := flag.String("a", "", "server address")
	flagDBDSN := flag.String("d", "", "DB connection")
	flagASAddr := flag.String("r", "", "accrual system address")
	flagProd := flag.Bool("prod", false, "product logging mode")
	flag.Parse()

	if len(*flagHost) > 0 {
		if strings.HasPrefix(*flagHost, ":") {
			*flagHost = "localhost" + *flagHost
		} else {
			st := strings.Split(*flagHost, ":")
			if len(st) == 1 {
				*flagHost = st[0] + ":" + "8080"
			}
		}
	}

	addr := getVarValue(*flagHost, "RUN_ADDRESS", DefaultHost)
	dbDSN := getVarValue(*flagDBDSN, "DATABASE_URI", DefaultDBDSN)
	asAddr := getVarValue(*flagASAddr, "ACCRUAL_SYSTEM_ADDRESS", "http://localhost:8080")

	log := logging.NewLogger(*flagProd)

	cfg := Config{
		Addr:      addr,
		DBDSN:     dbDSN,
		AcSysAddr: asAddr,
		//Store: store,
		Logger: log,
		Prod:   *flagProd,
	}
	return cfg
}

func getVarValue(flagValue, envVarName, defValue string) string {
	var ok bool
	varVal := flagValue
	if len(flagValue) == 0 {
		varVal, ok = os.LookupEnv(envVarName) //URLdata.json
		if !ok || varVal == "" {
			varVal = defValue
		}
	}
	return varVal
}
