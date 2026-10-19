package main

import (
	"log/slog"
	"os"

	"rednote/internal/server"
)

func main() {
	if err := server.Init(); err != nil {
		slog.Error("初始化失败", "err", err)
		os.Exit(1)
	}
	server.NewServer()
}
