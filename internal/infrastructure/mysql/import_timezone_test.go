package mysql_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func TestLaravelTehranWallTimeImportsAsSameUTCInstant(t *testing.T) {
	dsn := mysqltest.Database(t)
	raw := rawDB(t, dsn)
	_, err := raw.Exec(`CREATE TABLE users(id BIGINT PRIMARY KEY,mobile VARCHAR(255),updated_at DATETIME(6),name VARCHAR(255),email VARCHAR(255),password VARCHAR(255),code VARCHAR(255),referral VARCHAR(255),email_verified_at DATETIME(6),created_at DATETIME(6),wallet_address VARCHAR(255));
CREATE TABLE personal_infos(id BIGINT PRIMARY KEY,user_id BIGINT,is_company BOOLEAN,first_name VARCHAR(255),last_name VARCHAR(255),mobile VARCHAR(255),telephone VARCHAR(255),national_code VARCHAR(255),address VARCHAR(255),company_name VARCHAR(255),company_address VARCHAR(255),company_registration_number VARCHAR(255),company_national_number VARCHAR(255),company_tax_number VARCHAR(255),company_executive_name VARCHAR(255),is_verified BOOLEAN,verification_messages JSON);
CREATE TABLE oauth_clients(id BIGINT PRIMARY KEY,name VARCHAR(255),secret VARCHAR(100),redirect TEXT,personal_access_client BOOLEAN,password_client BOOLEAN,revoked BOOLEAN,created_at DATETIME(6));
CREATE TABLE media(id BIGINT PRIMARY KEY,model_type VARCHAR(255),model_id BIGINT,collection_name VARCHAR(255),file_name VARCHAR(255),disk VARCHAR(255));
INSERT INTO users(id,name,email,created_at,email_verified_at,updated_at) VALUES(7,'Member','timezone@example.com','2026-10-02 15:00:00','2026-10-02 15:00:00','2026-10-02 15:00:00');`)
	if err != nil {
		t.Fatal(err)
	}
	config, err := driver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.Loc, err = time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatal(err)
	}
	config.ParseTime = true
	source, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	target := mysqltest.Open(t)
	if _, err := target.ImportLaravel(context.Background(), source, t.TempDir(), true); err != nil {
		t.Fatal(err)
	}
	u, err := target.ByID(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2026, 10, 2, 11, 30, 0, 0, time.UTC)
	if !u.CreatedAt.Equal(expected) || u.EmailVerifiedAt == nil || !u.EmailVerifiedAt.Equal(expected) || !u.UpdatedAt.Equal(expected) {
		t.Fatal("import shifted the source instant", u.CreatedAt, u.EmailVerifiedAt, u.UpdatedAt)
	}
}
