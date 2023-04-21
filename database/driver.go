package database

type Driver interface {
	Upsert(object interface{}, model interface{}) error
	UpsertBatch(objects []interface{}, model interface{}) error
	Find(object interface{}, model interface{}) ([]interface{}, error)
	Delete(object interface{}, model interface{}) error
}
