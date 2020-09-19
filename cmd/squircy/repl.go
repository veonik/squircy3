package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peterh/liner"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squircy3/cli"
	"code.dopame.me/veonik/squircy3/vm"
)

func Repl(manager *cli.Manager) {
	hist := filepath.Join(manager.RootDir, ".history_repl")

	input := liner.NewLiner()
	input.SetCtrlCAborts(true)
	defer func() {
		if f, err := os.Create(hist); err == nil {
			if _, err = input.WriteHistory(f); err != nil {
				logrus.Warnln("failed to write history:", err)
			}
			_ = f.Close()
		}
		_ = input.Close()
	}()

	jsVM, err := vm.FromPlugins(manager.Plugins())
	if err != nil {
		logrus.Warnln("failed to get VM from plugin manager:", err)
		return
	}

	if f, err := os.Open(hist); err == nil {
		if _, err = input.ReadHistory(f); err != nil {
			logrus.Warnln("failed to read history:", err)
		}
		_ = f.Close()
	}
	fmt.Println("Starting javascript REPL...")
	fmt.Println("Type 'exit' and hit enter to exit the REPL.")
	ctrlcs := 0
	for {
		str, err := input.Prompt("repl> ")
		if err == liner.ErrPromptAborted && ctrlcs == 0 {
			ctrlcs += 1
			fmt.Println("Press CTRL+C again to close the REPL.")
			continue
		}
		if str == "exit" || err == liner.ErrPromptAborted {
			fmt.Println("Closing REPL...")
			break
		}
		ctrlcs = 0
		if str == "" {
			continue
		}
		input.AppendHistory(str)
		v, err := jsVM.RunString(str).Await()
		if err != nil {
			logrus.Warnln("error:", err)
		}
		fmt.Println(v)
	}
}
