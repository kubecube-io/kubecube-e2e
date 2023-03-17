package main

import (
	"github.com/kubecube-io/kubecube-e2e/e2e"
	"os"

	"github.com/kubecube-io/kubecube/pkg/clog"
)

// 兜底资源清理脚本
func main() {
	if err := e2e.Clear(); err != nil {
		clog.Error(err.Error())
		os.Exit(1)
	}

	clog.Info("resource cleared")
}
