package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

func startREPL() {
	completer := readline.NewPrefixCompleter(
		readline.PcItem("help"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("status"),
		readline.PcItem("health"),
		readline.PcItem("config"),
		readline.PcItem("start"),
		readline.PcItem("stop"),
		readline.PcItem("clients"),
		readline.PcItem("ap",
			readline.PcItem("status"),
			readline.PcItem("start"),
			readline.PcItem("stop"),
			readline.PcItem("show"),
		),
		readline.PcItem("set",
			readline.PcItem("interface"),
			readline.PcItem("ssid"),
			readline.PcItem("channel"),
			readline.PcItem("password"),
			readline.PcItem("security"),
			readline.PcItem("api"),
			readline.PcItem("docker"),
		),
		readline.PcItem("presets",
			readline.PcItem("open_nav"),
			readline.PcItem("google_phish"),
			readline.PcItem("router_attack"),
		),
		readline.PcItem("unset"),
		readline.PcItem("ignore"),
		readline.PcItem("restore"),
		readline.PcItem("info",
			readline.PcItem("proxy"),
			readline.PcItem("plugin"),
		),
		readline.PcItem("jobs"),
		readline.PcItem("mode",
			readline.PcItem("set"),
		),
		readline.PcItem("profiles",
			readline.PcItem("list"),
			readline.PcItem("select"),
			readline.PcItem("create"),
			readline.PcItem("update"),
			readline.PcItem("delete"),
		),
		readline.PcItem("plugins",
			readline.PcItem("list"),
			readline.PcItem("toggle"),
			readline.PcItem("start"),
			readline.PcItem("stop"),
		),
		readline.PcItem("proxies",
			readline.PcItem("list"),
		),
		readline.PcItem("show",
			readline.PcItem("modules"),
			readline.PcItem("plugins"),
			readline.PcItem("proxies"),
			readline.PcItem("commands"),
		),
		readline.PcItem("search"),
		readline.PcItem("use"),
		readline.PcItem("dump",
			readline.PcItem("credentials"),
		),
		readline.PcItem("dhcpconf"),
		readline.PcItem("dhcpmode",
			readline.PcItem("status"),
			readline.PcItem("list"),
			readline.PcItem("set"),
		),
		readline.PcItem("update"),
		readline.PcItem("banner"),
		readline.PcItem("clear"),
		readline.PcItem("interfaces"),
		readline.PcItem("recon",
			readline.PcItem("scan"),
		),
		readline.PcItem("portal",
			readline.PcItem("status"),
			readline.PcItem("credentials"),
			readline.PcItem("start"),
			readline.PcItem("stop"),
		),
		readline.PcItem("system"),
		readline.PcItem("firewall",
			readline.PcItem("apply"),
			readline.PcItem("clear"),
		),
		readline.PcItem("deauth"),
		readline.PcItem("eviltwin"),
		readline.PcItem("beacon",
			readline.PcItem("stop"),
		),
		readline.PcItem("monitor"),
		readline.PcItem("logs"),
		readline.PcItem("tui"),
		readline.PcItem("templates",
			readline.PcItem("list"),
			readline.PcItem("import"),
		),
		readline.PcItem("version"),
	)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          colorText(ansiCyan, "ap1 > "),
		HistoryFile:     "/tmp/ap1_history.tmp",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error initializing readline:", err)
		return
	}
	defer rl.Close()

	fmt.Println(colorText(ansiGreen, "codename: Gao"))
	fmt.Println(colorText(ansiCyan, "by: @gaetal | version: "+buildVersion))
	fmt.Printf("[*] Session id: %d\n", time.Now().Unix())
	fmt.Println(colorText(ansiYellow, "Use TAB for autocompletion"))

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					break
				} else {
					continue
				}
			} else if err == io.EOF {
				break
			}
			fmt.Fprintln(os.Stderr, "error reading input:", err)
			break
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

		handleGlobalCommand(cmd, args)
	}
}
