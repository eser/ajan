package datafx

import (
	"context"
	"database/sql"
)

type Datasource struct {
	connection DataConnection
}

func NewDatasource(dataConnection DataConnection) *Datasource {
	return &Datasource{
		connection: dataConnection,
	}
}

func (datasource *Datasource) ExecuteUnitOfWork(
	ctx context.Context,
	fn func(uow *UnitOfWork) error,
) error {
	rawConnection, _ := datasource.connection.GetRawConnection().(*sql.DB)
	uow := NewUnitOfWork(rawConnection)

	return uow.Execute(ctx, fn)
}
