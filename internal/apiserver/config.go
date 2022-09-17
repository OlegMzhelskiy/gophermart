package apiserver

import (
	"flag"
	"os"
	"strings"

	"github.com/OlegMzhelskiy/gophermart/internal/storage"
)

type Config struct {
	Addr      string
	DBDSN     string
	AcSysAddr string
	Store     storage.Repository
}

func NewConfig() Config {
	flagHost := flag.String("a", "", "server address")
	flagDBDSN := flag.String("d", "", "DB connection")
	flagASAddr := flag.String("r", "", "accrual system address")
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
	asAddr := getVarValue(*flagASAddr, "ACCRUAL_SYSTEM_ADDRESS", "")

	cfg := Config{
		Addr:      addr,
		DBDSN:     dbDSN,
		AcSysAddr: asAddr,
		//Store: store,
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
