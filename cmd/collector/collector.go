package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jessevdk/go-flags"

	"github.com/derfenix/stats_collector/internal"
	"github.com/derfenix/stats_collector/pkg/collector"
)

// options application's options
type options struct {
	DBUser string `long:"dbuser" env:"DB_USER" description:"Database user"`
	DBPass string `long:"dbpass" env:"DB_PASS" description:"Database password"`
	DBName string `long:"dbname" env:"DB_NAME" default:"statistics" description:"Database name"`
	DBHost string `long:"dbhost" env:"DB_HOST" default:"db" description:"Database host"`
	DBPort uint   `long:"dbport" env:"DB_PORT" default:"3306" description:"Database port"`

	WSHost     string `long:"wshost" env:"WS_HOST" default:"0.0.0.0" description:"WS server host"`
	WSPort     uint   `long:"wsport" env:"WS_PORT" default:"8881" description:"WS server port"`
	WSEndpoint string `long:"wsendpoint" env:"WS_ENDPOINT" default:"/ws/" description:"WS endpoint"`

	SyncInt uint `short:"i" long:"interval" env:"SYNC_INTERVAL" default:"15" description:"Flush to database each seconds (sync interval)"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	{ // Graceful stop on KILL ot INTERRUPT signals
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt, os.Kill)
		go func() {
			s := <-ch
			log.Printf("received %s signal, stopping", s)
			cancel()

			// Force application stop after 5 seconds (emergency quit if something went wrong)
			timer := time.NewTimer(time.Second * 5)
			defer timer.Stop()
			<-timer.C
			log.Fatal("quit forced")
		}()
	}

	conf := new(options)
	{ // Parse options and env variables
		parser := flags.NewParser(conf, flags.Default)
		if _, err := parser.Parse(); err != nil {
			if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			} else {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		}
	}

	// Create database connection
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", conf.DBUser, conf.DBPass, conf.DBHost, conf.DBPort, conf.DBName),
	)
	if err != nil {
		log.Fatalf("failed to connect to the database: %s", err.Error())
	}

	wg := &sync.WaitGroup{}

	storage := collector.NewStorage(db)
	wg.Add(1)
	go storage.Syncer(ctx, wg, time.Second*time.Duration(conf.SyncInt))

	handler := collector.GetMessageHandler(storage)

	wg.Add(1)
	if err := internal.StartWSServer(ctx, wg, conf.WSHost, conf.WSPort, conf.WSEndpoint, handler); err != nil {
		log.Fatalf("failed to start ws server: %s", err.Error())
	}
	wg.Wait()
}
