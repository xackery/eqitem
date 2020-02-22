package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	//mysql db
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-colorable"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xackery/eqemuconfig"
)

func main() {
	start := time.Now()

	//logger prep
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}
	if runtime.GOOS == "windows" {
		output = zerolog.ConsoleWriter{Out: colorable.NewColorableStdout()}
	}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%3s", i))
	}
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s: ", i)
	}
	output.FormatFieldValue = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()

	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	//run program
	err := run()
	if err != nil {
		log.Error().Err(err).Msg("failed")
	}
	log.Info().Msgf("completed in %0.1f seconds", time.Since(start).Seconds())
}

func run() error {
	if len(os.Args) < 2 {
		fmt.Println("usage: itemimport items.txt [itemid]")
		os.Exit(1)
	}

	cfg, err := eqemuconfig.GetConfig()
	if err != nil {
		return err
	}

	conn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Db)
	db, err := sqlx.Open("mysql", conn)
	if err != nil {
		return errors.Wrap(err, "sql open")
	}
	defer db.Close()

	path := os.Args[1]
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	var itemid int64
	if len(os.Args) > 2 {
		itemid, err = strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			return err
		}
	}

	r := csv.NewReader(f)
	r.Comma = '|'
	r.LazyQuotes = true

	total := 0
	err = db.Get(&total, "SELECT COUNT(id) FROM items")
	if err != nil {
		return errors.Wrap(err, "item count")
	}

	ids := []string{}

	header := []string{}
	lineCount := 0
	for {
		lineCount++
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Warn().Err(err).Int("line", lineCount).Msg("read")
			continue
		}
		if lineCount == 1 {
			header = record
			continue
		}

		//temporary
		item, err := NewItem(header, record)
		if err != nil {
			log.Warn().Err(err).Int("line", lineCount).Msg("newItem")
		}

		if itemid > 0 && item.ID != itemid {
			continue
		}

		oldItem := new(EQEmuItem)
		row := db.QueryRowx("SELECT * FROM items where id = ?", item.ID)
		if err = row.StructScan(oldItem); err != nil {
			if err == sql.ErrNoRows {
				if _, err = db.NamedExec(item.insertQuery(), item); err != nil {
					return errors.Wrapf(err, "insert %d", item.ID)
				}
				fmt.Println("inserted", item.ID)
				ids = append(ids, fmt.Sprintf("%d", item.ID))
				continue
			}
			return errors.Wrap(err, "old item")
		}
		if lineCount%1000 == 0 {
			fmt.Println(lineCount)
		}
	}

	log.Debug().Msgf("processed %d lines", lineCount)
	fmt.Println("total ids: ", strings.Join(ids, ", "))
	return nil
}
