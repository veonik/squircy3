package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peterh/liner"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squircy3/vm"
)

func (manager *Manager) Repl() {
	hist := filepath.Join(manager.RootDir, ".history_repl")

	cli := liner.NewLiner()
	cli.SetCtrlCAborts(true)
	defer func() {
		if f, err := os.Create(hist); err == nil {
			if _, err = cli.WriteHistory(f); err != nil {
				logrus.Warnln("failed to write history:", err)
			}
			_ = f.Close()
		}
		_ = cli.Close()
	}()

	jsVM, err := vm.FromPlugins(manager.plugins)
	if err != nil {
		logrus.Warnln("failed to get VM from plugin manager:", err)
		return
	}

	if f, err := os.Open(hist); err == nil {
		if _, err = cli.ReadHistory(f); err != nil {
			logrus.Warnln("failed to read history:", err)
		}
		_ = f.Close()
	}
	fmt.Println("Starting javascript REPL...")
	fmt.Println("Type 'exit' and hit enter to exit the REPL.")
	ctrlcs := 0
	for {
		str, err := cli.Prompt("repl> ")
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
		cli.AppendHistory(str)
		v, err := jsVM.RunString(str).Await()
		if err != nil {
			logrus.Warnln("error:", err)
		}
		fmt.Println(v)
	}
}
