package main

import (
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()

	appContext, err := NewAppContext(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize app context: %v", err))
	}

	appContext.Logger.Info("app context initialized",
		"name", appContext.Config.AppName,
		"env", appContext.Config.AppEnv,
	)
}
