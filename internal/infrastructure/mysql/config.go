package mysql

import (
	driver "github.com/go-sql-driver/mysql"
	"net"
	"os"
	"time"
)

func ConfigFromEnv() *driver.Config {
	env := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	c := driver.NewConfig()
	c.User = env("MYSQL_USER", "root")
	c.Passwd = os.Getenv("MYSQL_PASSWORD")
	c.Net = "tcp"
	c.Addr = net.JoinHostPort(env("MYSQL_HOST", "127.0.0.1"), env("MYSQL_PORT", "3306"))
	c.DBName = env("MYSQL_DATABASE", "paradise")
	c.ParseTime = true
	c.Loc = time.UTC
	c.Timeout = 5 * time.Second
	c.ReadTimeout = 30 * time.Second
	c.WriteTimeout = 30 * time.Second
	c.Params = map[string]string{"charset": "utf8mb4", "time_zone": "'+00:00'"}
	return c
}
