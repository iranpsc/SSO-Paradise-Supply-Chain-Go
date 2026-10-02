// testdb manages only dedicated Playwright databases, never application data.
package main

import (
	"database/sql"
	"flag"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/envfile"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"log"
	"os"
	"regexp"
)

func main() {
	envfile.MustLoad()
	drop := flag.Bool("drop", false, "drop this isolated E2E database")
	flag.Parse()
	name := os.Getenv("MYSQL_DATABASE")
	if !regexp.MustCompile(`^paradise_e2e_[0-9]+_[0-9]+$`).MatchString(name) {
		log.Fatal("testdb only accepts paradise_e2e_<timestamp>_<pid> databases")
	}
	config := mysql.ConfigFromEnv()
	config.DBName = ""
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	command := "CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"
	if *drop {
		command = "DROP DATABASE IF EXISTS `" + name + "`"
	}
	if _, err = db.Exec(command); err != nil {
		log.Fatal(err)
	}
}
