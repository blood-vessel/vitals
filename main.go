package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/blood-vessel/vitals/api"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func main() {
	config := viper.New()
	config.AutomaticEnv()

	var envPath string
	flag.StringVar(&envPath, "env", "", "path to .env file to use")
	flag.Parse()

	var err error
	if envPath != "" {
		err = godotenv.Load(envPath)
	} else {
		err = godotenv.Load()
	}
	if err != nil {
		fmt.Printf("failed to load .env file %s\n", err)
	}

	config.SetDefault("PORT", 8080)
	port := config.GetInt("PORT")
	log.Info("listening", "port", port)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("failed to listen", "err", err)
	}

	ctx := context.Background()
	opts := &api.RunOptions{
		Writer:   os.Stdout,
		Listener: ln,
		Config:   config,
	}
	if err := api.Run(ctx, opts); err != nil {
		log.Fatal("failed to run", "err", err)
	}
}
