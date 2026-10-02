package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/cache"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/envfile"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	envfile.MustLoad()
	if len(os.Args) < 2 {
		log.Fatal("usage: manage client:create --name NAME --redirect-uri URI [--public] | cleanup [--apply]")
	}
	db, err := mysql.Open(mysql.ConfigFromEnv().FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	switch os.Args[1] {
	case "client:create":
		flags := flag.NewFlagSet("client:create", flag.ExitOnError)
		name := flags.String("name", "", "client display name")
		uris := flags.String("redirect-uri", "", "comma-separated exact redirect URIs")
		public := flags.Bool("public", false, "public PKCE client without secret")
		flags.Parse(os.Args[2:])
		secret, hash := "", ""
		if !*public {
			b := make([]byte, 32)
			if _, err = rand.Read(b); err != nil {
				log.Fatal(err)
			}
			secret = hex.EncodeToString(b)
			hash, err = (security.Bcrypt{Cost: 10}).Hash(secret)
			if err != nil {
				log.Fatal(err)
			}
		}
		redirects := strings.Split(*uris, ",")
		for i := range redirects {
			redirects[i] = strings.TrimSpace(redirects[i])
		}
		id, e := db.CreateOAuthClient(ctx, *name, hash, redirects)
		if e != nil {
			log.Fatal(e)
		}
		json.NewEncoder(os.Stdout).Encode(map[string]any{"client_id": id, "client_secret": secret, "redirect_uris": redirects})
	case "cleanup":
		flags := flag.NewFlagSet("cleanup", flag.ExitOnError)
		apply := flags.Bool("apply", false, "delete unverified users older than 24h and expired records; default is dry-run")
		flags.Parse(os.Args[2:])
		store := &cache.Store{DB: db}
		addr := os.Getenv("REDIS_ADDR")
		if addr == "" {
			addr = "127.0.0.1:6379"
		}
		number, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
		client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("REDIS_PASSWORD"), DB: number})
		defer client.Close()
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if client.Ping(pingCtx).Err() == nil {
			store.Redis = client
		}
		cancel()
		result, e := store.Cleanup(ctx, time.Now(), *apply)
		if e != nil {
			log.Fatal(e)
		}
		fmt.Printf("dry_run=%t candidates=%d deleted_users=%d expired_records=%d\n", !*apply, len(result.Users), result.DeletedUsers, result.ExpiredRecords)
	default:
		log.Fatal("unknown command")
	}
}
