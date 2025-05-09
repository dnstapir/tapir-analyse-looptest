package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

    "github.com/dnstapir/tapir-analyse-looptest/setup"
)

/* Rewritten if building with make */
var version = "BAD-BUILD"

const c_ENVVAR_OVERRIDE_NATS_URL = "TAPIR_ANALYSE_LOOPTEST_NATS_URL"

func main() {
	var configFile string
	var runVersionCmd bool
	var quietFlag bool
	var debugFlag bool
	var mainConf setup.AppConf

	flag.BoolVar(&runVersionCmd,
		"version",
		false,
		"Print version then exit",
	)
	flag.StringVar(&configFile,
		"config-file",
		"/etc/dnstapir/tapir-analyse-looptest.json",
		"Configuration file to use",
	)
	flag.BoolVar(&debugFlag,
		"debug",
		false,
		"Enable DEBUG logs",
	)
	flag.BoolVar(&quietFlag,
		"quiet",
		false,
		"Suppress INFO and DEBUG logs",
	)
	flag.Parse()

	if runVersionCmd {
		fmt.Printf("tapir-analyse-looptest version %s\n", version)
		os.Exit(0)
	}

	file, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("Couldn't open config file '%s', exiting...\n", configFile)
		os.Exit(-1)
	}

	confDecoder := json.NewDecoder(file)
	if confDecoder == nil {
		fmt.Printf("Problem decoding config file '%s', exiting...\n", configFile)
		os.Exit(-1)
	}

	confDecoder.DisallowUnknownFields()
	confDecoder.Decode(&mainConf)

	/* If set, CLI flags override config file */
	if debugFlag {
		mainConf.Debug = true
	}
	if quietFlag {
		mainConf.Quiet = true
	}

    envNatsUrl, overrideNatsUrl := os.LookupEnv(c_ENVVAR_OVERRIDE_NATS_URL)
    if overrideNatsUrl {
        mainConf.Nats.Url = envNatsUrl
    }

	application, err := setup.BuildApp(mainConf)
	if err != nil {
		fmt.Printf("Error building application: '%s', exiting...\n", err)
		os.Exit(-1)
	}

	sigChan := make(chan os.Signal, 1)
    defer close(sigChan)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	done := application.Run()

	select {
	case s := <-sigChan:
		fmt.Printf("Got signal '%s', exiting...\n", s)
	case err := <-done:
		if err != nil {
			fmt.Printf("App exited with error: '%s'\n", err)
		} else {
			fmt.Printf("Done!\n")
		}
	}

	err = application.Stop()
	if err != nil {
		fmt.Printf("Error stopping app: '%s'\n", err)
		os.Exit(-1)
	}

	os.Exit(0)
}
