package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func startREPL() {
    reader := bufio.NewReader(os.Stdin)
    fmt.Println("codename: Gao")
    fmt.Println("by: @gaetal | version:", buildVersion)
    fmt.Printf("[*] Session id: %d\n", time.Now().Unix())
    for {
        fmt.Print("ap1 > ")
        line, err := reader.ReadString('\n')
        if err != nil {
            fmt.Fprintln(os.Stderr, "error reading input:", err)
            return
        }
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        parts := strings.Fields(line)
        cmd := parts[0]
        args := []string{}
        if len(parts) > 1 {
            args = parts[1:]
        }

        switch cmd {
        case "exit", "quit":
            cmdExit()
            return
        case "help":
            usage()
        case "status":
            cmdStatus()
        case "health":
            cmdHealth()
        case "config":
            cmdConfig()
        case "start":
            cmdStart(args)
        case "stop":
            cmdStop(args)
        case "clients":
            cmdClients(args)
        case "ap":
            cmdAP(args)
        case "set":
            cmdSet(args)
        case "presets":
            cmdPresets(args)
        case "unset":
            cmdUnset(args)
        case "ignore":
            cmdIgnore(args)
        case "restore":
            cmdRestore(args)
        case "info":
            cmdInfo(args)
        case "jobs":
            cmdJobs(args)
        case "mode":
            cmdMode(args)
        case "profiles":
            cmdProfiles(args)
        case "plugins":
            cmdPlugins(args)
        case "proxies":
            cmdProxies(args)
        case "show":
            cmdShow(args)
        case "search":
            cmdSearch(args)
        case "use":
            cmdUse(args)
        case "dump":
            cmdDump(args)
        case "dhcpconf":
            cmdDhcpconf(args)
        case "dhcpmode":
            cmdDhcpmode(args)
        case "update":
            cmdUpdate(args)
        case "banner":
            cmdBanner(args)
        case "clear":
            cmdClear(args)
        case "interfaces":
            cmdInterfaces(args)
        case "recon":
            cmdRecon(args)
        case "portal":
            cmdPortal(args)
        case "system":
            cmdSystem(args)
        case "deauth":
            cmdDeauth(args)
        case "eviltwin":
            cmdEvilTwin(args)
        case "beacon":
            cmdBeacon(args)
        case "monitor":
            cmdMonitor(args)
        case "templates":
            cmdTemplates(args)
        case "version":
            cmdVersion()
        default:
            fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
        }
    }
}
