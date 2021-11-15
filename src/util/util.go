package util

func CkErr(err error) {
	if err != nil {
		panic(err)
	}
}

type Closable interface {
	Close() error
}

func Close(thing Closable) {
	CkErr(thing.Close())
}
