package ds

import (
	"github.com/jmoiron/sqlx"
	"github.com/juju/errors"
	"golang.org/x/net/context"
)

const DEFAULT_DATASOURCE = "default"

var DataSources = map[string]*sqlx.DB{}

func RegisterDataSource(ds string, db *sqlx.DB) {
	DataSources[ds] = db
}

const (
	DATASOURCE_KEY  = "_ds_"
	DS_EXECUTOR_KEY = "_executor_"
)

type IExecutor interface {
	sqlx.Ext
}

func Executor(c context.Context) (IExecutor, error) {
	exe := c.Value(DS_EXECUTOR_KEY)
	if exe != nil {
		return exe.(IExecutor), nil
	}
	return nil, errors.New("Context without any sql executor")
}

func DoNoTx(c context.Context, f func(c context.Context) (interface{}, error)) (v interface{}, err error) {
	dsKey := c.Value(DATASOURCE_KEY)
	if dsKey == nil {
		dsKey = DEFAULT_DATASOURCE
	}
	c = context.WithValue(c, DS_EXECUTOR_KEY, DataSources[dsKey.(string)])
	return f(c)
}

func DoTx(c context.Context, f func(c context.Context) (interface{}, error), noRollbackErrs ...error) (v interface{}, err error) {
	dsKey := c.Value(DATASOURCE_KEY)
	if dsKey == nil {
		dsKey = DEFAULT_DATASOURCE
	}
	var tx *sqlx.Tx
	tx, err = DataSources[dsKey.(string)].Beginx()
	if err != nil {
		return
	}
	c = context.WithValue(c, DS_EXECUTOR_KEY, tx)
	v, err = f(c)
	if err != nil && !isRollbackErr(err, noRollbackErrs) {
		err2 := tx.Rollback()
		if err2 != nil {
			err = err2
		}
		return
	}
	err = tx.Commit()
	return

}

func isRollbackErr(err error, noRollbackErrs []error) bool {
	for _, noRollbackErr := range noRollbackErrs {
		if err == noRollbackErr {
			return false
		}
	}
	return true
}
