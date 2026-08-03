// Команда migrate катает миграции goose из вшитой FS.
//
//	migrate [up|down|status|version|up-to N|...]   (по умолчанию up)
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/sweetlife999/chain-of-trades-avito/migrations"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL не задан")
	}

	cmd := "up"
	var args []string
	if len(os.Args) > 1 {
		cmd, args = os.Args[1], os.Args[2:]
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("подключение к БД: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("диалект goose: %v", err)
	}
	if err := goose.RunContext(context.Background(), cmd, db, ".", args...); err != nil {
		log.Fatalf("миграция %q: %v", cmd, err)
	}
}
