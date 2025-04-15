package main

import (
	"context"
	"fmt"

	"github.com/eser/ajan/datafx"
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

	business(ctx, appContext)
}

func business(ctx context.Context, appContext *AppContext) {
	datasource := appContext.Data.GetDefault()

	err := datasource.ExecuteUnitOfWork(ctx, func(uow *datafx.UnitOfWork) error {
		// service1.DoSomething(uow)
		// service2.DoSomething(uow)
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to do unit of work: %v", err))
	}
}
